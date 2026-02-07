package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gogit/index"
	"gogit/object"
	"gogit/repo"
)

func Add(paths []string) error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	idx, err := index.ReadIndex(root)
	if err != nil {
		return err
	}

	for _, p := range paths {
		if err := addPath(root, idx, p); err != nil {
			return err
		}
	}

	return index.WriteIndex(root, idx)
}

var absFunc = filepath.Abs
var relFunc = filepath.Rel

func addPath(root string, idx *index.Index, p string) error {
	// Make path relative to repo root
	absPath, err := absFunc(p)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		// File was deleted — remove from index
		relPath, _ := filepath.Rel(root, absPath)
		relPath = filepath.ToSlash(relPath)
		idx.RemoveEntry(relPath)
		return nil
	}

	if info.IsDir() {
		return addDir(root, idx, absPath)
	}

	return addFile(root, idx, absPath, info)
}

func addDir(root string, idx *index.Index, dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == repo.GogitDir {
				return filepath.SkipDir
			}
			return nil
		}
		return addFile(root, idx, path, info)
	})
}

func addFile(root string, idx *index.Index, absPath string, info os.FileInfo) error {
	relPath, err := relFunc(root, absPath)
	if err != nil {
		return err
	}
	relPath = filepath.ToSlash(relPath)

	// Skip .gogit directory
	if strings.HasPrefix(relPath, repo.GogitDir) {
		return nil
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	hash, err := object.WriteBlob(root, content)
	if err != nil {
		return err
	}

	mode := uint32(0100644)
	if info.Mode()&0111 != 0 {
		mode = 0100755
	}

	entry := index.Entry{
		Ctime: uint32(info.ModTime().Unix()),
		Mtime: uint32(info.ModTime().Unix()),
		Size:  uint32(info.Size()),
		Hash:  hash,
		Mode:  mode,
		Path:  relPath,
	}

	idx.AddEntry(entry)
	fmt.Printf("add '%s'\n", relPath)
	return nil
}
