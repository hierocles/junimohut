package mods

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ModConfigView is the payload for the integrated config editor.
type ModConfigView struct {
	ModID                  string `json:"modId"`
	ModName                string `json:"modName"`
	FolderPath             string `json:"folderPath"`
	RelPath                string `json:"relPath"`
	DisplayPath            string `json:"displayPath"`
	AbsolutePath           string `json:"absolutePath"`
	Content                string `json:"content"`
	ProfileName            string `json:"profileName"`
	ProfileSpecificConfigs bool   `json:"profileSpecificConfigs"`
}

// ModJsonSummary describes a mod that contains editable JSON files.
type ModJsonSummary struct {
	ModID         string `json:"modId"`
	ModName       string `json:"modName"`
	FolderPath    string `json:"folderPath"`
	JsonFileCount int    `json:"jsonFileCount"`
}

// ModJsonFileNode is a folder or JSON file in a mod's config tree.
type ModJsonFileNode struct {
	Name     string            `json:"name"`
	RelPath  string            `json:"relPath,omitempty"`
	IsDir    bool              `json:"isDir"`
	Children []ModJsonFileNode `json:"children,omitempty"`
}

var (
	ErrModConfigNotFound    = errors.New("config file not found")
	ErrModConfigInvalidJSON = errors.New("invalid JSONC")
	ErrModConfigInvalidPath = errors.New("invalid config file path")
)

// ModDir returns the absolute mod folder path.
func ModDir(modsRoot, folderPath string) string {
	return filepath.Join(modsRoot, filepath.FromSlash(folderPath))
}

// ConfigPathForMod returns the absolute config.json path for a mod folder under modsRoot.
func ConfigPathForMod(modsRoot, folderPath string) string {
	return filepath.Join(ModDir(modsRoot, folderPath), "config.json")
}

// ValidateModRelativeJSONPath ensures relPath is a safe .json path within a mod folder.
func ValidateModRelativeJSONPath(relPath string) error {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if relPath == "" {
		return fmt.Errorf("%w: empty path", ErrModConfigInvalidPath)
	}
	if strings.Contains(relPath, "..") {
		return fmt.Errorf("%w: path traversal", ErrModConfigInvalidPath)
	}
	if !strings.HasSuffix(strings.ToLower(relPath), ".json") {
		return fmt.Errorf("%w: not a .json file", ErrModConfigInvalidPath)
	}
	return nil
}

// ResolveModJSONPath returns the absolute path for a relative JSON file in a mod folder.
func ResolveModJSONPath(modsRoot, folderPath, relPath string) (string, error) {
	if err := ValidateModRelativeJSONPath(relPath); err != nil {
		return "", err
	}
	modDir, err := filepath.Abs(ModDir(modsRoot, folderPath))
	if err != nil {
		return "", err
	}
	abs := filepath.Join(modDir, filepath.FromSlash(filepath.ToSlash(relPath)))
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, modDir+string(os.PathSeparator)) && abs != modDir {
		if !(len(modDir) > 0 && abs == modDir) {
			return "", fmt.Errorf("%w: outside mod folder", ErrModConfigInvalidPath)
		}
	}
	// Windows: normalize prefix check
	rel, err := filepath.Rel(modDir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: outside mod folder", ErrModConfigInvalidPath)
	}
	return abs, nil
}

// IsEditableJSONFile reports whether name is an editable mod JSON file (not manifest.json).
func IsEditableJSONFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".json") &&
		!strings.EqualFold(name, "manifest.json")
}

// ListJsonFileRelPaths walks modDir and returns relative editable .json file paths (POSIX slashes).
func ListJsonFileRelPaths(modDir string) ([]string, error) {
	info, err := os.Stat(modDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("mod folder is not a directory")
	}
	var paths []string
	err = filepath.WalkDir(modDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !IsEditableJSONFile(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(modDir, path)
		if err != nil {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

// CountJsonFiles returns the number of .json files under modDir.
func CountJsonFiles(modDir string) int {
	paths, err := ListJsonFileRelPaths(modDir)
	if err != nil {
		return 0
	}
	return len(paths)
}

// DefaultJsonRelPath picks config.json when present, otherwise the first sorted path.
func DefaultJsonRelPath(paths []string) string {
	for _, p := range paths {
		if strings.EqualFold(p, "config.json") {
			return p
		}
	}
	if len(paths) == 0 {
		return "config.json"
	}
	return paths[0]
}

// BuildJsonFileTree builds a nested tree from flat relative JSON paths.
func BuildJsonFileTree(paths []string) []ModJsonFileNode {
	type node struct {
		children map[string]*node
		isFile   bool
		relPath  string
	}
	root := &node{children: map[string]*node{}}
	for _, rel := range paths {
		parts := strings.Split(rel, "/")
		cur := root
		for i, part := range parts {
			if cur.children == nil {
				cur.children = map[string]*node{}
			}
			child, ok := cur.children[part]
			if !ok {
				child = &node{children: map[string]*node{}}
				cur.children[part] = child
			}
			if i == len(parts)-1 {
				child.isFile = true
				child.relPath = rel
			}
			cur = child
		}
	}
	var build func(n *node) []ModJsonFileNode
	build = func(n *node) []ModJsonFileNode {
		names := make([]string, 0, len(n.children))
		for name := range n.children {
			names = append(names, name)
		}
		sort.Strings(names)
		out := make([]ModJsonFileNode, 0, len(names))
		for _, name := range names {
			child := n.children[name]
			if child.isFile && len(child.children) == 0 {
				out = append(out, ModJsonFileNode{
					Name:    name,
					RelPath: child.relPath,
					IsDir:   false,
				})
				continue
			}
			out = append(out, ModJsonFileNode{
				Name:     name,
				IsDir:    true,
				Children: build(child),
			})
		}
		return out
	}
	return build(root)
}

// ReadModJsonFile reads a JSON file from a mod folder.
func ReadModJsonFile(modsRoot, folderPath, relPath string) (string, error) {
	p, err := ResolveModJSONPath(modsRoot, folderPath, relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrModConfigNotFound
		}
		return "", err
	}
	return string(data), nil
}

// WriteModJsonFile validates JSONC and writes a JSON file in a mod folder.
// Comments and trailing commas are accepted; the file is stored as edited.
func WriteModJsonFile(modsRoot, folderPath, relPath, content string) error {
	if err := ValidJSONC(content); err != nil {
		return err
	}
	p, err := ResolveModJSONPath(modsRoot, folderPath, relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

// ReadModConfig reads config.json for a mod folder.
func ReadModConfig(modsRoot, folderPath string) (string, error) {
	return ReadModJsonFile(modsRoot, folderPath, "config.json")
}

// WriteModConfig validates and writes config.json for a mod folder.
func WriteModConfig(modsRoot, folderPath, content string) error {
	return WriteModJsonFile(modsRoot, folderPath, "config.json", content)
}
