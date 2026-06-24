package moddataset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"junimohut/internal/httpclient"
)

// ModPageDownload summarizes one file on a mod page.
type ModPageDownload struct {
	ID          int    `json:"id"`
	DisplayName string `json:"displayName"`
	FileName    string `json:"fileName"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	ModCount    int    `json:"modCount"`
}

// ModPage is a trimmed mod page record from the dataset.
type ModPage struct {
	Site      string            `json:"site"`
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	Author    string            `json:"author"`
	PageUrl   string            `json:"pageUrl"`
	TagLine   string            `json:"tagLine"`
	Downloads []ModPageDownload `json:"downloads"`
}

type pageCacheFile struct {
	FetchedAt int64           `json:"fetchedAt"`
	Page      datasetModPage  `json:"page"`
}

type datasetModPage struct {
	Site        string              `json:"Site"`
	ID          int                 `json:"Id"`
	Name        string              `json:"Name"`
	Author      string              `json:"Author"`
	TagLine     string              `json:"TagLine"`
	PageUrl     string              `json:"PageUrl"`
	Downloads   []datasetDownload   `json:"Downloads"`
}

type datasetDownload struct {
	ID          json.Number `json:"Id"`
	DisplayName string      `json:"DisplayName"`
	FileName    string      `json:"FileName"`
	Version     string      `json:"Version"`
	Type        string      `json:"Type"`
	Mods        []struct{}  `json:"Mods"`
}

// DataBucket returns the dataset folder bucket for a mod page ID.
func DataBucket(site string, id int) int {
	switch site {
	case "CurseForge", "ModDrop":
		return id / 10000
	default:
		return id / 1000
	}
}

func pageCachePath(pagesDir, site string, id int) string {
	return filepath.Join(pagesDir, "pages", site, fmt.Sprintf("%d.json", id))
}

func pageSourceURL(site string, id int) string {
	bucket := DataBucket(site, id)
	return fmt.Sprintf("%s/%s/%d/%d.json", pageDataBaseURL, site, bucket, id)
}

// FetchModPage returns dataset metadata for a mod page, using a monthly on-disk cache.
func FetchModPage(ctx context.Context, pagesDir, site string, id int) (ModPage, error) {
	if site == "" || id <= 0 {
		return ModPage{}, fmt.Errorf("invalid mod page reference")
	}
	cachePath := pageCachePath(pagesDir, site, id)
	if page, ok, err := loadCachedPage(cachePath, site, id); err != nil {
		return ModPage{}, err
	} else if ok {
		return page, nil
	}
	return downloadModPage(ctx, pagesDir, site, id, cachePath)
}

func loadCachedPage(cachePath, site string, id int) (ModPage, bool, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ModPage{}, false, nil
		}
		return ModPage{}, false, err
	}
	var cached pageCacheFile
	if err := json.Unmarshal(data, &cached); err != nil {
		return ModPage{}, false, nil
	}
	if cached.FetchedAt == 0 || time.Since(time.Unix(cached.FetchedAt, 0)) >= IndexRefreshInterval {
		return ModPage{}, false, nil
	}
	return mapModPage(site, id, cached.Page), true, nil
}

func downloadModPage(ctx context.Context, pagesDir, site string, id int, cachePath string) (ModPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageSourceURL(site, id), nil)
	if err != nil {
		return ModPage{}, err
	}
	resp, err := httpclient.DoWithRetry(httpclient.Default(), req, 3)
	if err != nil {
		return ModPage{}, fmt.Errorf("download mod page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ModPage{}, fmt.Errorf("mod page not found in dataset")
	}
	if resp.StatusCode >= 400 {
		return ModPage{}, fmt.Errorf("download mod page: HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModPage{}, err
	}
	var record datasetModPage
	if err := json.Unmarshal(raw, &record); err != nil {
		return ModPage{}, fmt.Errorf("parse mod page: %w", err)
	}
	cached := pageCacheFile{
		FetchedAt: time.Now().Unix(),
		Page:      record,
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return ModPage{}, err
	}
	if data, err := json.MarshalIndent(cached, "", "  "); err == nil {
		_ = os.WriteFile(cachePath, data, 0o644)
	}
	return mapModPage(site, id, record), nil
}

func mapModPage(site string, id int, record datasetModPage) ModPage {
	page := ModPage{
		Site:    site,
		ID:      id,
		Name:    record.Name,
		Author:  record.Author,
		PageUrl: record.PageUrl,
		TagLine: record.TagLine,
	}
	if page.Name == "" {
		page.Name = record.Name
	}
	if record.ID > 0 {
		page.ID = record.ID
	}
	downloads := make([]ModPageDownload, 0, len(record.Downloads))
	for _, dl := range record.Downloads {
		dlID, _ := dl.ID.Int64()
		downloads = append(downloads, ModPageDownload{
			ID:          int(dlID),
			DisplayName: dl.DisplayName,
			FileName:    dl.FileName,
			Version:     dl.Version,
			Type:        dl.Type,
			ModCount:    len(dl.Mods),
		})
	}
	page.Downloads = downloads
	return page
}

// FetchModPageForUniqueID resolves the preferred page ref and fetches metadata.
func FetchModPageForUniqueID(ctx context.Context, idx *Index, uniqueID string) (ModPage, error) {
	if idx == nil {
		return ModPage{}, fmt.Errorf("mod dataset index not available")
	}
	refs := idx.LookupPages(uniqueID)
	site, id, ok := PreferNexusRef(refs)
	if !ok {
		return ModPage{}, fmt.Errorf("mod not found in dataset index")
	}
	return FetchModPage(ctx, idx.PagesDir(), site, id)
}
