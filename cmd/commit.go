package cmd

import (
	"fmt"

	"gogit/index"
	"gogit/object"
	"gogit/refs"
	"gogit/repo"
)

func Commit(message string) error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	idx, err := index.ReadIndex(root)
	if err != nil {
		return err
	}

	if len(idx.Entries) == 0 {
		return fmt.Errorf("nothing to commit")
	}

	// Build tree from index
	treeHash, err := object.BuildTreeFromIndex(root, idx)
	if err != nil {
		return err
	}

	// Get parent commit
	var parents []string
	headHash, err := refs.ResolveHead(root)
	if err != nil {
		return err
	}
	if headHash != "" {
		parents = append(parents, headHash)
	}

	commitHash, err := writeCommitAndUpdateRef(root, treeHash, parents, message)
	if err != nil {
		return err
	}

	branch, _ := refs.CurrentBranch(root)
	fmt.Printf("[%s %s] %s\n", branchDisplay(branch), commitHash[:7], message)
	return nil
}

var writeCommitFn = object.WriteCommit

func writeCommitAndUpdateRef(root, treeHash string, parents []string, message string) (string, error) {
	// Create commit object
	commitHash, err := writeCommitFn(root, treeHash, parents, message)
	if err != nil {
		return "", err
	}

	// Update branch ref
	branch, err := refs.CurrentBranch(root)
	if err != nil {
		return "", err
	}
	if branch != "" {
		if err := refs.WriteRef(root, refs.BranchRef(branch), commitHash); err != nil {
			return "", err
		}
	} else {
		// Detached HEAD
		if err := refs.UpdateHead(root, commitHash); err != nil {
			return "", err
		}
	}

	return commitHash, nil
}

func branchDisplay(branch string) string {
	if branch == "" {
		return "detached HEAD"
	}
	return branch
}
