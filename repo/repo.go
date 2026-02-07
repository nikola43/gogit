package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

const GogitDir = ".gogit"

// Find walks up from the current directory to find a .gogit repository root.
func Find() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindFrom(dir)
}

// FindFrom walks up from the given directory to find a .gogit repository root.
func FindFrom(dir string) (string, error) {
	for {
		if info, err := os.Stat(filepath.Join(dir, GogitDir)); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a gogit repository (or any parent)")
		}
		dir = parent
	}
}

// GogitPath returns the full path to the .gogit directory.
func GogitPath(root string) string {
	return filepath.Join(root, GogitDir)
}

// ObjectsPath returns the path to the objects directory.
func ObjectsPath(root string) string {
	return filepath.Join(root, GogitDir, "objects")
}

// RefsPath returns the path to the refs directory.
func RefsPath(root string) string {
	return filepath.Join(root, GogitDir, "refs")
}

// HeadPath returns the path to the HEAD file.
func HeadPath(root string) string {
	return filepath.Join(root, GogitDir, "HEAD")
}

// IndexPath returns the path to the index file.
func IndexPath(root string) string {
	return filepath.Join(root, GogitDir, "index")
}
