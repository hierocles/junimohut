package nexus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ModFile is a file entry from the Nexus mod files API.
type ModFile struct {
	FileID            int    `json:"file_id"`
	Name              string `json:"name"`
	Version           string `json:"version"`
	CategoryName      string `json:"category_name"`
	FileName          string `json:"file_name"`
	UploadedTimestamp int64  `json:"uploaded_timestamp"`
}

type modFilesResponse struct {
	Files []ModFile `json:"files"`
}

// ListModFiles returns all files for a mod from the Nexus API.
func (c *Client) ListModFiles(modID int) ([]ModFile, error) {
	c.mu.RLock()
	key := c.apiKey
	c.mu.RUnlock()
	if key == "" {
		return nil, ErrNoAPIKeyConfigured
	}

	url := fmt.Sprintf("%s/v1/games/%s/mods/%d/files.json", c.apiBaseURL(), gameDomain, modID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", key)
	setUserAgent(req)

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return nil, requestError("list Nexus mod files", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, readAPIError(resp, "list mod files")
	}

	var data modFilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Files) == 0 {
		return nil, fmt.Errorf("no files found for mod %d", modID)
	}
	return data.Files, nil
}

// ResolveLatestMainFile picks the newest MAIN file, or the newest eligible file as fallback.
func ResolveLatestMainFile(files []ModFile) (ModFile, error) {
	if len(files) == 0 {
		return ModFile{}, fmt.Errorf("no files to resolve")
	}

	var mainCandidates []ModFile
	var fallback []ModFile
	for _, f := range files {
		cat := strings.ToUpper(strings.TrimSpace(f.CategoryName))
		if cat == "REMOVED" || cat == "ARCHIVED" {
			continue
		}
		fallback = append(fallback, f)
		if cat == "MAIN" || cat == "" {
			mainCandidates = append(mainCandidates, f)
		}
	}

	pick := func(candidates []ModFile) (ModFile, error) {
		if len(candidates) == 0 {
			return ModFile{}, fmt.Errorf("no eligible files")
		}
		best := candidates[0]
		for _, f := range candidates[1:] {
			if f.UploadedTimestamp > best.UploadedTimestamp {
				best = f
			}
		}
		return best, nil
	}

	if len(mainCandidates) > 0 {
		return pick(mainCandidates)
	}
	return pick(fallback)
}

// ResolveModFile returns metadata for a specific file, or the latest MAIN file when fileID is 0.
func ResolveModFile(client *Client, modID, fileID int) (ModFile, error) {
	files, err := client.ListModFiles(modID)
	if err != nil {
		return ModFile{}, err
	}
	if fileID == 0 {
		return ResolveLatestMainFile(files)
	}
	for _, f := range files {
		if f.FileID == fileID {
			return f, nil
		}
	}
	return ModFile{}, fmt.Errorf("file %d not found for mod %d", fileID, modID)
}

func downloadDestName(file ModFile, modName string) string {
	name := strings.TrimSpace(file.FileName)
	if name == "" {
		name = fmt.Sprintf("%s.zip", sanitize(modName))
	}
	return sanitizeFileName(name)
}

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "download.zip"
	}
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, name)
	name = strings.TrimRight(name, " .")
	if name == "" {
		return "download.zip"
	}
	return name
}

func uniqueDestPath(dir, name string) string {
	dest := filepath.Join(dir, name)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func (c *Client) apiBaseURL() string {
	if c.apiBase != "" {
		return strings.TrimRight(c.apiBase, "/")
	}
	return "https://api.nexusmods.com"
}
