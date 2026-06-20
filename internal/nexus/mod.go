package nexus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// ModInfo is summary metadata from the Nexus mod details API.
type ModInfo struct {
	ModID      int    `json:"mod_id"`
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
}

type gameCategory struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
}

type gameInfoResponse struct {
	Categories []gameCategory `json:"categories"`
}

type modInfoResponse struct {
	ModInfo
}

var gameCategoriesCache struct {
	mu    sync.RWMutex
	names map[int]string
	err   error
}

// GetMod returns mod metadata from the Nexus API.
func (c *Client) GetMod(modID int) (ModInfo, error) {
	c.mu.RLock()
	key := c.apiKey
	c.mu.RUnlock()
	if key == "" {
		return ModInfo{}, ErrNoAPIKeyConfigured
	}
	if modID <= 0 {
		return ModInfo{}, fmt.Errorf("Invalid mod id")
	}

	url := fmt.Sprintf("%s/v1/games/%s/mods/%d.json", c.apiBaseURL(), gameDomain, modID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ModInfo{}, err
	}
	req.Header.Set("apikey", key)
	setUserAgent(req)

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return ModInfo{}, requestError("get Nexus mod info", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ModInfo{}, readAPIError(resp, "get mod info")
	}

	var data modInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ModInfo{}, err
	}
	return data.ModInfo, nil
}

// CategoryNameForMod resolves the Nexus page category name for a mod.
func (c *Client) CategoryNameForMod(modID int) (string, error) {
	mod, err := c.GetMod(modID)
	if err != nil {
		return "", err
	}
	if mod.CategoryID == 0 {
		return "", nil
	}
	names, err := c.gameCategoryNames()
	if err != nil {
		return "", err
	}
	return names[mod.CategoryID], nil
}

func (c *Client) gameCategoryNames() (map[int]string, error) {
	gameCategoriesCache.mu.RLock()
	if gameCategoriesCache.names != nil {
		names := gameCategoriesCache.names
		err := gameCategoriesCache.err
		gameCategoriesCache.mu.RUnlock()
		return names, err
	}
	gameCategoriesCache.mu.RUnlock()

	gameCategoriesCache.mu.Lock()
	defer gameCategoriesCache.mu.Unlock()
	if gameCategoriesCache.names != nil {
		return gameCategoriesCache.names, gameCategoriesCache.err
	}

	names, err := c.fetchGameCategoryNames()
	gameCategoriesCache.names = names
	gameCategoriesCache.err = err
	return names, err
}

func (c *Client) fetchGameCategoryNames() (map[int]string, error) {
	c.mu.RLock()
	key := c.apiKey
	c.mu.RUnlock()
	if key == "" {
		return nil, ErrNoAPIKeyConfigured
	}

	url := fmt.Sprintf("%s/v1/games/%s.json", c.apiBaseURL(), gameDomain)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", key)
	setUserAgent(req)

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return nil, requestError("get Nexus game info", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, readAPIError(resp, "get game info")
	}

	var data gameInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	names := make(map[int]string, len(data.Categories))
	for _, cat := range data.Categories {
		if cat.CategoryID == 0 || cat.Name == "" {
			continue
		}
		names[cat.CategoryID] = cat.Name
	}
	return names, nil
}

// ResetGameCategoriesCache clears cached game categories (for tests).
func ResetGameCategoriesCache() {
	gameCategoriesCache.mu.Lock()
	defer gameCategoriesCache.mu.Unlock()
	gameCategoriesCache.names = nil
	gameCategoriesCache.err = nil
}
