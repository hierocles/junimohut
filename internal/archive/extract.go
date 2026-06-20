package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode"
)

// Extract unpacks zip, 7z, or rar archives to destDir.
func Extract(archivePath, destDir string) error {
	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".zip":
		return extractZip(archivePath, destDir)
	case ".7z":
		return extract7z(archivePath, destDir)
	case ".rar":
		return extractRar(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format: %s", ext)
	}
}

func extractZip(path, dest string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if err := extractZipFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, dest string) error {
	name := filepath.Clean(f.Name)
	if strings.HasPrefix(name, "..") {
		return nil
	}
	target := filepath.Join(dest, name)
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func extract7z(path, dest string) error {
	r, err := sevenzip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		name := filepath.Clean(f.Name)
		if strings.HasPrefix(name, "..") {
			continue
		}
		target := filepath.Join(dest, name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(target), 0o755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func extractRar(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	rr, err := rardecode.NewReader(f, "")
	if err != nil {
		return err
	}
	for {
		header, err := rr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(header.Name)
		if strings.HasPrefix(name, "..") {
			continue
		}
		target := filepath.Join(dest, name)
		if header.IsDir {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(target), 0o755)
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, rr)
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
