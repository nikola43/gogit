package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gogit/index"
	"gogit/object"
	"gogit/refs"
	"gogit/repo"
)

func Status() error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	branch, err := refs.CurrentBranch(root)
	if err != nil {
		return err
	}
	if branch != "" {
		fmt.Printf("On branch %s\n", branch)
	} else {
		fmt.Println("HEAD detached")
	}

	idx, err := index.ReadIndex(root)
	if err != nil {
		return err
	}

	// Get HEAD tree
	headTree := make(map[string]string)
	headHash, err := refs.ResolveHead(root)
	if err != nil {
		return err
	}
	if headHash != "" {
		commit, err := object.ReadCommit(root, headHash)
		if err != nil {
			return err
		}
		headTree, err = object.FlattenTree(root, commit.TreeHash, "")
		if err != nil {
			return err
		}
	}

	// Build index map
	indexMap := make(map[string]string)
	for _, e := range idx.Entries {
		indexMap[e.Path] = e.Hash
	}

	// Staged changes (HEAD vs index)
	var staged []string
	for path, idxHash := range indexMap {
		headHash, inHead := headTree[path]
		if !inHead {
			staged = append(staged, fmt.Sprintf("\tnew file:   %s", path))
		} else if idxHash != headHash {
			staged = append(staged, fmt.Sprintf("\tmodified:   %s", path))
		}
	}
	for path := range headTree {
		if _, inIndex := indexMap[path]; !inIndex {
			staged = append(staged, fmt.Sprintf("\tdeleted:    %s", path))
		}
	}

	// Unstaged changes (index vs working tree)
	var unstaged []string
	for _, e := range idx.Entries {
		absPath := filepath.Join(root, e.Path)
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				unstaged = append(unstaged, fmt.Sprintf("\tdeleted:    %s", e.Path))
			}
			continue
		}
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		currentHash := object.HashBlob(content)
		if currentHash != e.Hash {
			_ = info
			unstaged = append(unstaged, fmt.Sprintf("\tmodified:   %s", e.Path))
		}
	}

	// Untracked files
	var untracked []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == repo.GogitDir {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, _ := filepath.Rel(root, path)
		relPath = filepath.ToSlash(relPath)
		if _, inIndex := indexMap[relPath]; !inIndex {
			untracked = append(untracked, fmt.Sprintf("\t%s", relPath))
		}
		return nil
	})

	if len(staged) > 0 {
		fmt.Println("\nChanges to be committed:")
		for _, s := range staged {
			fmt.Println(s)
		}
	}

	if len(unstaged) > 0 {
		fmt.Println("\nChanges not staged for commit:")
		for _, s := range unstaged {
			fmt.Println(s)
		}
	}

	if len(untracked) > 0 {
		fmt.Println("\nUntracked files:")
		for _, s := range untracked {
			fmt.Println(s)
		}
	}

	if len(staged) == 0 && len(unstaged) == 0 && len(untracked) == 0 {
		fmt.Println("nothing to commit, working tree clean")
	}

	return nil
}
