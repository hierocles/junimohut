package moddataset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"junimohut/internal/httpclient"
	"junimohut/internal/mods"
)

const (
	IndexRefreshInterval = 30 * 24 * time.Hour
)

var (
	indexSourceURL  = "https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/indexes/pages%20by%20mod%20ID.json"
	pageDataBaseURL = "https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/data"
)

// IndexFile is the on-disk pages-by-mod-ID cache.
type IndexFile struct {
	FetchedAt int64               `json:"fetchedAt"`
	Pages     map[string][]string `json:"pages"`
}

// Index caches UniqueID → mod page references from the Stardew mod dataset.
type Index struct {
	mu        sync.RWMutex
	cachePath string
	pagesDir  string
	file      IndexFile
}

// NewIndex loads or creates the pages-by-mod-ID index cache.
func NewIndex(cachePath, pagesDir string) (*Index, error) {
	idx := &Index{
		cachePath: cachePath,
		pagesDir:  pagesDir,
		file: IndexFile{
			Pages: map[string][]string{},
		},
	}
	if err := idx.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return idx, nil
}

func (idx *Index) load() error {
	data, err := os.ReadFile(idx.cachePath)
	if err != nil {
		return err
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	return json.Unmarshal(data, &idx.file)
}

func (idx *Index) saveLocked() error {
	if idx.file.Pages == nil {
		idx.file.Pages = map[string][]string{}
	}
	data, err := json.MarshalIndent(idx.file, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(idx.pagesDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(idx.cachePath, data, 0o644)
}

func (idx *Index) isStale() bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if idx.file.FetchedAt == 0 || len(idx.file.Pages) == 0 {
		return true
	}
	return time.Since(time.Unix(idx.file.FetchedAt, 0)) >= IndexRefreshInterval
}

// LookupPages returns mod page refs for a SMAPI UniqueID.
func (idx *Index) LookupPages(uniqueID string) []string {
	uniqueID = strings.TrimSpace(uniqueID)
	if uniqueID == "" {
		return nil
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	for _, key := range []string{uniqueID, mods.CanonicalUniqueID(uniqueID)} {
		if refs := idx.file.Pages[key]; len(refs) > 0 {
			out := make([]string, len(refs))
			copy(out, refs)
			return out
		}
	}
	return nil
}

// FirstNexusID returns the first Nexus mod page ID for a UniqueID.
func (idx *Index) FirstNexusID(uniqueID string) int {
	id, _ := FirstNexusRef(idx.LookupPages(uniqueID))
	return id
}

// PagesDir returns the directory used for per-page JSON caches.
func (idx *Index) PagesDir() string {
	return idx.pagesDir
}

// RefreshIfStaleAsync downloads a fresh index when the cache is missing or older than 30 days.
func (idx *Index) RefreshIfStaleAsync() {
	go func() {
		_ = idx.RefreshIfStale(context.Background())
	}()
}

// RefreshIfStale downloads a fresh index when the cache is missing or older than 30 days.
func (idx *Index) RefreshIfStale(ctx context.Context) error {
	if !idx.isStale() {
		return nil
	}
	return idx.refresh(ctx)
}

func (idx *Index) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexSourceURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpclient.DoWithRetry(httpclient.Default(), req, 3)
	if err != nil {
		if httpclient.IsTransient(err) && len(idx.file.Pages) > 0 {
			return nil
		}
		return fmt.Errorf("download mod dataset index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		if len(idx.file.Pages) > 0 {
			return nil
		}
		return fmt.Errorf("download mod dataset index: HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var pages map[string][]string
	if err := json.Unmarshal(raw, &pages); err != nil {
		return fmt.Errorf("parse mod dataset index: %w", err)
	}
	idx.mu.Lock()
	idx.file = IndexFile{
		FetchedAt: time.Now().Unix(),
		Pages:     pages,
	}
	err = idx.saveLocked()
	idx.mu.Unlock()
	return err
}
