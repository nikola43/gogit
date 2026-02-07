package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gogit/index"
	"gogit/object"
	"gogit/refs"
	"gogit/repo"
)

func Checkout(target string) error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	// Check if target is a branch
	branchHash, err := refs.ReadRef(root, refs.BranchRef(target))
	if err != nil {
		return err
	}
	if branchHash == "" {
		return fmt.Errorf("branch '%s' not found", target)
	}

	// Get current HEAD commit tree
	currentHash, err := refs.ResolveHead(root)
	if err != nil {
		return err
	}

	if currentHash == branchHash {
		// Already on the right commit, just switch HEAD
		if err := refs.UpdateHead(root, "ref: refs/heads/"+target); err != nil {
			return err
		}
		fmt.Printf("Switched to branch '%s'\n", target)
		return nil
	}

	// Get current tree and target tree
	var currentTree map[string]string
	if currentHash != "" {
		commit, err := object.ReadCommit(root, currentHash)
		if err != nil {
			return err
		}
		currentTree, err = object.FlattenTree(root, commit.TreeHash, "")
		if err != nil {
			return err
		}
	} else {
		currentTree = make(map[string]string)
	}

	targetCommit, err := object.ReadCommit(root, branchHash)
	if err != nil {
		return err
	}
	targetTree, err := object.FlattenTree(root, targetCommit.TreeHash, "")
	if err != nil {
		return err
	}

	if err := updateWorkingTree(root, target, currentTree, targetTree); err != nil {
		return err
	}

	fmt.Printf("Switched to branch '%s'\n", target)
	return nil
}

func updateWorkingTree(root, target string, currentTree, targetTree map[string]string) error {
	// Remove files that are in current tree but not in target tree
	for path := range currentTree {
		if _, inTarget := targetTree[path]; !inTarget {
			absPath := filepath.Join(root, path)
			os.Remove(absPath)
			// Clean up empty parent directories
			cleanEmptyDirs(root, filepath.Dir(absPath))
		}
	}

	// Write/update files from target tree
	idx := &index.Index{}
	for path, hash := range targetTree {
		absPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return err
		}

		content, err := object.ReadBlob(root, hash)
		if err != nil {
			return err
		}

		if err := os.WriteFile(absPath, content, 0644); err != nil {
			return err
		}

		info, _ := os.Stat(absPath)
		idx.AddEntry(index.Entry{
			Ctime: uint32(info.ModTime().Unix()),
			Mtime: uint32(info.ModTime().Unix()),
			Size:  uint32(info.Size()),
			Hash:  hash,
			Mode:  0100644,
			Path:  path,
		})
	}

	if err := index.WriteIndex(root, idx); err != nil {
		return err
	}

	if err := refs.UpdateHead(root, "ref: refs/heads/"+target); err != nil {
		return err
	}

	return nil
}

func cleanEmptyDirs(root, dir string) {
	for dir != root && !strings.HasSuffix(dir, repo.GogitDir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}
