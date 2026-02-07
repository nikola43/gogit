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

func Merge(branchName string) error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	currentBranch, err := refs.CurrentBranch(root)
	if err != nil {
		return err
	}
	if currentBranch == "" {
		return fmt.Errorf("cannot merge in detached HEAD state")
	}

	// Resolve current HEAD
	currentHash, err := refs.ResolveHead(root)
	if err != nil {
		return err
	}
	if currentHash == "" {
		return fmt.Errorf("no commits on current branch")
	}

	// Resolve merge target
	targetHash, err := refs.ReadRef(root, refs.BranchRef(branchName))
	if err != nil {
		return err
	}
	if targetHash == "" {
		return fmt.Errorf("branch '%s' not found", branchName)
	}

	if currentHash == targetHash {
		fmt.Println("Already up to date.")
		return nil
	}

	// Check if fast-forward is possible (current is ancestor of target)
	if isAncestor(root, currentHash, targetHash) {
		return fastForwardMerge(root, currentBranch, branchName, targetHash)
	}

	// Check if target is ancestor of current (already merged)
	if isAncestor(root, targetHash, currentHash) {
		fmt.Println("Already up to date.")
		return nil
	}

	// File-level merge
	return fileLevelMerge(root, currentBranch, branchName, currentHash, targetHash)
}

// isAncestor checks if `ancestor` is an ancestor of `descendant`.
func isAncestor(root, ancestor, descendant string) bool {
	hash := descendant
	for hash != "" {
		if hash == ancestor {
			return true
		}
		commit, err := object.ReadCommit(root, hash)
		if err != nil {
			return false
		}
		if len(commit.Parents) > 0 {
			hash = commit.Parents[0]
		} else {
			hash = ""
		}
	}
	return false
}

func fastForwardMerge(root, currentBranch, targetBranch, targetHash string) error {
	// Update current branch to point to target
	if err := refs.WriteRef(root, refs.BranchRef(currentBranch), targetHash); err != nil {
		return err
	}

	// Update working tree
	targetCommit, err := object.ReadCommit(root, targetHash)
	if err != nil {
		return err
	}
	targetTree, err := object.FlattenTree(root, targetCommit.TreeHash, "")
	if err != nil {
		return err
	}

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

	fmt.Printf("Fast-forward merge: %s -> %s\n", currentBranch, targetHash[:7])
	return nil
}

// findMergeBase finds the common ancestor of two commits.
func findMergeBase(root, hash1, hash2 string) string {
	// Collect all ancestors of hash1
	ancestors := make(map[string]bool)
	h := hash1
	for h != "" {
		ancestors[h] = true
		commit, err := object.ReadCommit(root, h)
		if err != nil {
			break
		}
		if len(commit.Parents) > 0 {
			h = commit.Parents[0]
		} else {
			h = ""
		}
	}

	// Walk hash2's ancestors to find first common one
	h = hash2
	for h != "" {
		if ancestors[h] {
			return h
		}
		commit, err := object.ReadCommit(root, h)
		if err != nil {
			break
		}
		if len(commit.Parents) > 0 {
			h = commit.Parents[0]
		} else {
			h = ""
		}
	}
	return ""
}

func fileLevelMerge(root, currentBranch, targetBranch, currentHash, targetHash string) error {
	baseHash := findMergeBase(root, currentHash, targetHash)

	var baseTree map[string]string
	if baseHash != "" {
		baseCommit, err := object.ReadCommit(root, baseHash)
		if err != nil {
			return err
		}
		baseTree, err = object.FlattenTree(root, baseCommit.TreeHash, "")
		if err != nil {
			return err
		}
	} else {
		baseTree = make(map[string]string)
	}

	currentCommit, err := object.ReadCommit(root, currentHash)
	if err != nil {
		return err
	}
	currentTree, err := object.FlattenTree(root, currentCommit.TreeHash, "")
	if err != nil {
		return err
	}

	targetCommit, err := object.ReadCommit(root, targetHash)
	if err != nil {
		return err
	}
	targetTree, err := object.FlattenTree(root, targetCommit.TreeHash, "")
	if err != nil {
		return err
	}

	// Collect all paths
	allPaths := make(map[string]bool)
	for p := range baseTree {
		allPaths[p] = true
	}
	for p := range currentTree {
		allPaths[p] = true
	}
	for p := range targetTree {
		allPaths[p] = true
	}

	// Merge each file
	mergedTree := make(map[string]string)
	hasConflict := false

	for path := range allPaths {
		baseH := baseTree[path]
		curH := currentTree[path]
		tarH := targetTree[path]

		switch {
		case curH == tarH:
			// Both same (or both deleted)
			if curH != "" {
				mergedTree[path] = curH
			}
		case curH == baseH:
			// Only target changed
			if tarH != "" {
				mergedTree[path] = tarH
			}
			// else: target deleted, don't include
		case tarH == baseH:
			// Only current changed
			if curH != "" {
				mergedTree[path] = curH
			}
			// else: current deleted, don't include
		default:
			// Both changed differently — conflict at file level
			fmt.Printf("CONFLICT (content): Merge conflict in %s\n", path)
			hasConflict = true
			// Keep current version
			if curH != "" {
				mergedTree[path] = curH
			}
		}
	}

	if hasConflict {
		return fmt.Errorf("automatic merge failed; fix conflicts and then commit")
	}

	// Write merged files to working tree and index
	idx := &index.Index{}
	for path, hash := range mergedTree {
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

	// Remove files that were in current tree but not in merged result
	for path := range currentTree {
		if _, ok := mergedTree[path]; !ok {
			absPath := filepath.Join(root, path)
			os.Remove(absPath)
		}
	}

	// Build tree and create merge commit
	treeHash, err := object.BuildTreeFromIndex(root, idx)
	if err != nil {
		return err
	}

	parents := []string{currentHash, targetHash}
	message := fmt.Sprintf("Merge branch '%s' into %s", targetBranch, currentBranch)
	commitHash, err := object.WriteCommit(root, treeHash, parents, message)
	if err != nil {
		return err
	}

	if err := refs.WriteRef(root, refs.BranchRef(currentBranch), commitHash); err != nil {
		return err
	}

	fmt.Printf("Merge made by the 'file-level' strategy.\n")
	fmt.Printf("[%s %s] %s\n", currentBranch, commitHash[:7], message)
	return nil
}
