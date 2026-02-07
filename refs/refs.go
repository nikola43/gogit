package refs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gogit/repo"
)

// ReadHead reads the HEAD file and returns its content.
func ReadHead(root string) (string, error) {
	data, err := os.ReadFile(repo.HeadPath(root))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ResolveHead resolves HEAD to a commit hash. Returns "" if no commits yet.
func ResolveHead(root string) (string, error) {
	head, err := ReadHead(root)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(head, "ref: ") {
		refPath := strings.TrimPrefix(head, "ref: ")
		return ReadRef(root, refPath)
	}
	return head, nil
}

// CurrentBranch returns the current branch name, or "" if HEAD is detached.
func CurrentBranch(root string) (string, error) {
	head, err := ReadHead(root)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(head, "ref: refs/heads/") {
		return strings.TrimPrefix(head, "ref: refs/heads/"), nil
	}
	return "", nil
}

// ReadRef reads a ref file and returns the commit hash. Returns "" if not found.
func ReadRef(root, refPath string) (string, error) {
	fullPath := filepath.Join(repo.GogitPath(root), refPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteRef writes a commit hash to a ref file.
func WriteRef(root, refPath, hash string) error {
	fullPath := filepath.Join(repo.GogitPath(root), refPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(hash+"\n"), 0644)
}

// UpdateHead updates the HEAD file.
func UpdateHead(root, content string) error {
	return os.WriteFile(repo.HeadPath(root), []byte(content+"\n"), 0644)
}

// ListBranches returns all branch names.
func ListBranches(root string) ([]string, error) {
	headsDir := filepath.Join(repo.RefsPath(root), "heads")
	entries, err := os.ReadDir(headsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var branches []string
	for _, e := range entries {
		if !e.IsDir() {
			branches = append(branches, e.Name())
		}
	}
	return branches, nil
}

// BranchRef returns the ref path for a branch.
func BranchRef(name string) string {
	return fmt.Sprintf("refs/heads/%s", name)
}
