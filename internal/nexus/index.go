package nexus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"junimohut/internal/mods"
)

// DownloadRecord links a saved archive to mod identity metadata.
type DownloadRecord struct {
	ArchivePath  string `json:"archivePath"`
	NexusModID   int    `json:"nexusModId,omitempty"`
	UniqueID     string `json:"uniqueId,omitempty"`
	FileName     string `json:"fileName,omitempty"`
	DownloadedAt int64  `json:"downloadedAt"`
}

type downloadIndexFile struct {
	Records []DownloadRecord `json:"records"`
}

// DownloadIndex persists download-to-mod associations in downloads.json.
type DownloadIndex struct {
	mu      sync.RWMutex
	path    string
	dir     string
	records []DownloadRecord
}

// NewDownloadIndex loads or creates the download index and reconciles with disk.
func NewDownloadIndex(dataDir, downloadsDir string) (*DownloadIndex, error) {
	_ = os.MkdirAll(downloadsDir, 0o755)
	idx := &DownloadIndex{
		path: filepath.Join(dataDir, "downloads.json"),
		dir:  downloadsDir,
	}
	if err := idx.load(); err != nil {
		return nil, err
	}
	idx.reconcile()
	return idx, nil
}

func (idx *DownloadIndex) load() error {
	data, err := os.ReadFile(idx.path)
	if err != nil {
		if os.IsNotExist(err) {
			idx.records = nil
			return nil
		}
		return err
	}
	var file downloadIndexFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	idx.records = file.Records
	return nil
}

func (idx *DownloadIndex) saveLocked() error {
	data, err := json.MarshalIndent(downloadIndexFile{Records: idx.records}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(idx.path, data, 0o644)
}

// InDownloadsDir reports whether archivePath is inside the managed downloads folder.
func (idx *DownloadIndex) InDownloadsDir(archivePath string) bool {
	if archivePath == "" {
		return false
	}
	rel, err := filepath.Rel(idx.dir, archivePath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// Record upserts a download record and persists the index.
func (idx *DownloadIndex) Record(rec DownloadRecord) error {
	if rec.ArchivePath == "" {
		return nil
	}
	if rec.DownloadedAt == 0 {
		rec.DownloadedAt = time.Now().Unix()
	}
	if rec.FileName == "" {
		rec.FileName = filepath.Base(rec.ArchivePath)
	}
	rec.ArchivePath = filepath.Clean(rec.ArchivePath)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	for i, existing := range idx.records {
		if filepath.Clean(existing.ArchivePath) == rec.ArchivePath {
			if rec.NexusModID == 0 {
				rec.NexusModID = existing.NexusModID
			}
			if rec.UniqueID == "" {
				rec.UniqueID = existing.UniqueID
			}
			if rec.FileName == "" {
				rec.FileName = existing.FileName
			}
			idx.records[i] = rec
			return idx.saveLocked()
		}
	}
	idx.records = append(idx.records, rec)
	return idx.saveLocked()
}

// List returns saved archives on disk, newest first.
func (idx *DownloadIndex) List() []DownloadRecord {
	idx.reconcile()
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	out := make([]DownloadRecord, 0, len(idx.records))
	for _, rec := range idx.records {
		if !idx.fileExists(rec.ArchivePath) {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DownloadedAt > out[j].DownloadedAt
	})
	return out
}

// Delete removes an archive from disk and the index.
func (idx *DownloadIndex) Delete(archivePath string) error {
	path := filepath.Clean(archivePath)
	if path == "" {
		return fmt.Errorf("Archive path is required")
	}
	if !idx.InDownloadsDir(path) {
		return fmt.Errorf("Archive is not in the downloads folder")
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	found := false
	pruned := idx.records[:0]
	for _, rec := range idx.records {
		if filepath.Clean(rec.ArchivePath) == path {
			found = true
			continue
		}
		pruned = append(pruned, rec)
	}
	if !found {
		return fmt.Errorf("Archive not found")
	}
	idx.records = pruned
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return idx.saveLocked()
}

// FindForMod returns the newest saved archive matching the mod by UniqueID or Nexus mod ID.
func (idx *DownloadIndex) FindForMod(uniqueID string, nexusModID int) (string, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if uniqueID != "" {
		if path, ok := idx.bestMatchLocked(func(rec DownloadRecord) bool {
			return rec.UniqueID != "" && rec.UniqueID == uniqueID
		}); ok {
			return path, true
		}
	}
	if nexusModID > 0 {
		return idx.bestMatchLocked(func(rec DownloadRecord) bool {
			return rec.NexusModID == nexusModID
		})
	}
	return "", false
}

func (idx *DownloadIndex) bestMatchLocked(match func(DownloadRecord) bool) (string, bool) {
	var best DownloadRecord
	var found bool
	for _, rec := range idx.records {
		if !match(rec) || !idx.fileExists(rec.ArchivePath) {
			continue
		}
		if !found || rec.DownloadedAt > best.DownloadedAt {
			best = rec
			found = true
		}
	}
	if !found {
		return "", false
	}
	return best.ArchivePath, true
}

func (idx *DownloadIndex) fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (idx *DownloadIndex) reconcile() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	pruned := idx.records[:0]
	seen := map[string]bool{}
	for _, rec := range idx.records {
		path := filepath.Clean(rec.ArchivePath)
		if !idx.fileExists(path) {
			continue
		}
		rec.ArchivePath = path
		pruned = append(pruned, rec)
		seen[path] = true
	}
	idx.records = pruned

	entries, err := os.ReadDir(idx.dir)
	if err != nil {
		_ = idx.saveLocked()
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".zip" && ext != ".7z" && ext != ".rar" {
			continue
		}
		path := filepath.Join(idx.dir, entry.Name())
		if seen[path] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		rec := DownloadRecord{
			ArchivePath:  path,
			FileName:     entry.Name(),
			DownloadedAt: info.ModTime().Unix(),
		}
		if manifests, err := mods.ManifestsFromArchive(path); err == nil && len(manifests) > 0 {
			rec.UniqueID = manifests[0].UniqueID
		}
		idx.records = append(idx.records, rec)
		seen[path] = true
	}
	_ = idx.saveLocked()
}

// RecordDownload archives a completed Nexus download in the index.
func (idx *DownloadIndex) RecordDownload(modID int, archivePath, fileName string) {
	if idx == nil || archivePath == "" {
		return
	}
	rec := DownloadRecord{
		ArchivePath:  archivePath,
		NexusModID:   modID,
		FileName:     fileName,
		DownloadedAt: time.Now().Unix(),
	}
	if manifests, err := mods.ManifestsFromArchive(archivePath); err == nil && len(manifests) > 0 {
		rec.UniqueID = manifests[0].UniqueID
	}
	_ = idx.Record(rec)
}

// RecordInstall associates an archive in the downloads folder with an installed mod.
func (idx *DownloadIndex) RecordInstall(archivePath, uniqueID string, nexusModID int) {
	if idx == nil || !idx.InDownloadsDir(archivePath) {
		return
	}
	rec := DownloadRecord{
		ArchivePath:  archivePath,
		UniqueID:     uniqueID,
		NexusModID:   nexusModID,
		FileName:     filepath.Base(archivePath),
		DownloadedAt: time.Now().Unix(),
	}
	if uniqueID == "" {
		if manifests, err := mods.ManifestsFromArchive(archivePath); err == nil && len(manifests) > 0 {
			rec.UniqueID = manifests[0].UniqueID
		}
	}
	_ = idx.Record(rec)
}
