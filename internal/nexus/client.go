package nexus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "JunimoHut"
	gameDomain  = "stardewvalley"
)

// Client interacts with the Nexus Mods API.
type Client struct {
	mu      sync.RWMutex
	apiKey  string
	apiBase string // optional override for tests
}

func NewClient() *Client {
	c := &Client{}
	key, err := keyring.Get(serviceName, "nexus-api-key")
	if err == nil {
		c.apiKey = key
	}
	return c
}

func (c *Client) SetAPIKey(key string) error {
	c.mu.Lock()
	c.apiKey = key
	c.mu.Unlock()
	if key == "" {
		return keyring.Delete(serviceName, "nexus-api-key")
	}
	return keyring.Set(serviceName, "nexus-api-key", key)
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey != ""
}

func (c *Client) ValidateKey() (bool, error) {
	c.mu.RLock()
	key := c.apiKey
	c.mu.RUnlock()
	if key == "" {
		return false, ErrNoAPIKeyConfigured
	}
	req, _ := http.NewRequest("GET", "https://api.nexusmods.com/v1/users.json", nil)
	req.Header.Set("apikey", key)
	setUserAgent(req)
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return false, requestError("validate Nexus API key", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return false, readAPIError(resp, "validate API key")
	}
	return resp.StatusCode == 200, nil
}

// EndorseMod endorses a mod on Nexus.
func (c *Client) EndorseMod(modID int, version string) error {
	c.mu.RLock()
	key := c.apiKey
	c.mu.RUnlock()
	url := fmt.Sprintf("https://api.nexusmods.com/v1/games/%s/mods/%d/endorse.json", gameDomain, modID)
	body := fmt.Sprintf(`{"version":"%s"}`, version)
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("apikey", key)
	req.Header.Set("Content-Type", "application/json")
	setUserAgent(req)
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return requestError("endorse mod on Nexus", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return readAPIError(resp, "endorse mod")
	}
	return nil
}

// DownloadEntry tracks a download.
type DownloadEntry struct {
	ID        string `json:"id"`
	ModName   string `json:"modName"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	FilePath  string `json:"filePath"`
	StartedAt int64  `json:"startedAt"`
}

// DownloadManager tracks NXM and update downloads.
type DownloadManager struct {
	mu      sync.RWMutex
	entries []DownloadEntry
	dir     string
	index   *DownloadIndex
}

func NewDownloadManager(dir string, index *DownloadIndex) *DownloadManager {
	_ = os.MkdirAll(dir, 0o755)
	return &DownloadManager{dir: dir, entries: []DownloadEntry{}, index: index}
}

func (d *DownloadManager) List() []DownloadEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]DownloadEntry, len(d.entries))
	copy(out, d.entries)
	return out
}

// DownloadFile downloads a mod file from Nexus.
// Premium accounts can omit auth; free accounts must pass the key/expiry from an nxm:// link.
func (d *DownloadManager) DownloadFile(client *Client, modID, fileID int, modName string, auth *DownloadAuth) (string, error) {
	modFile, err := ResolveModFile(client, modID, fileID)
	if err != nil {
		if fileID == 0 {
			return "", err
		}
		modFile = ModFile{FileID: fileID}
	} else {
		fileID = modFile.FileID
	}

	entryID := uuid.NewString()
	dest := uniqueDestPath(d.dir, downloadDestName(modFile, modName))
	entry := DownloadEntry{
		ID:        entryID,
		ModName:   modName,
		Status:    "downloading",
		Progress:  0,
		StartedAt: time.Now().Unix(),
	}
	d.addEntry(entry)

	client.mu.RLock()
	key := client.apiKey
	client.mu.RUnlock()
	if key == "" {
		err := ErrNoAPIKeyConfigured
		d.setEntryFailed(entryID, err)
		return "", err
	}

	linkURL := downloadLinkURL(client, modID, fileID, auth)
	req, err := http.NewRequest(http.MethodGet, linkURL, nil)
	if err != nil {
		d.setEntryFailed(entryID, err)
		return "", err
	}
	req.Header.Set("apikey", key)
	setUserAgent(req)
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		err = requestError("get Nexus download link", err)
		d.setEntryFailed(entryID, err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		err := readAPIError(resp, "get download link")
		d.setEntryFailed(entryID, err)
		return "", err
	}
	var links []struct {
		URI string `json:"URI"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil || len(links) == 0 {
		d.setEntryFailed(entryID, fmt.Errorf("no download link"))
		return "", fmt.Errorf("no download link")
	}

	d.updateEntry(entryID, func(e *DownloadEntry) {
		e.URL = links[0].URI
	})

	if err := d.downloadURL(links[0].URI, dest, entryID); err != nil {
		err = requestError("download mod archive", err)
		d.setEntryFailed(entryID, err)
		return "", err
	}

	d.updateEntry(entryID, func(e *DownloadEntry) {
		e.Status = "complete"
		e.Progress = 100
		e.FilePath = dest
	})
	if d.index != nil {
		d.index.RecordDownload(modID, dest, filepath.Base(dest))
	}
	return dest, nil
}

func (d *DownloadManager) addEntry(entry DownloadEntry) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries = append(d.entries, entry)
}

func (d *DownloadManager) updateEntry(id string, fn func(*DownloadEntry)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.entries {
		if d.entries[i].ID == id {
			fn(&d.entries[i])
			return
		}
	}
}

func (d *DownloadManager) setEntryFailed(id string, err error) {
	d.updateEntry(id, func(e *DownloadEntry) {
		e.Status = "failed"
		if err != nil {
			e.URL = err.Error()
		}
	})
}

func (d *DownloadManager) downloadURL(url, dest, entryID string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	setUserAgent(req)
	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			written += int64(n)
			if total > 0 {
				progress := int(written * 100 / total)
				d.updateEntry(entryID, func(e *DownloadEntry) {
					e.Progress = progress
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '_'
		}
		return r
	}, s)
}

// ExtractNexusID from UpdateKey "Nexus:2400".
func ExtractNexusID(updateKey string) (int, bool) {
	if !strings.HasPrefix(updateKey, "Nexus:") {
		return 0, false
	}
	var id int
	_, err := fmt.Sscanf(updateKey, "Nexus:%d", &id)
	return id, err == nil
}

// ModIDFromUpdateKeys returns the first Nexus mod ID found in manifest UpdateKeys.
func ModIDFromUpdateKeys(keys []string) int {
	for _, key := range keys {
		if id, ok := ExtractNexusID(key); ok {
			return id
		}
	}
	return 0
}
