package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"gogit/refs"
	"gogit/repo"
)

func TestLog_NoCommits(t *testing.T) {
	setupTestRepo(t)
	if err := Log(); err != nil {
		t.Fatalf("Log should not fail with no commits: %v", err)
	}
}

func TestLog_SingleCommit(t *testing.T) {
	setupTestRepoWithCommit(t)
	if err := Log(); err != nil {
		t.Fatalf("Log failed: %v", err)
	}
}

func TestLog_MultipleCommits(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("more"), 0644)
	Add([]string{"file2.txt"})
	Commit("second commit")

	if err := Log(); err != nil {
		t.Fatalf("Log failed: %v", err)
	}
}

func TestLog_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Log()
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestLog_BadCommitHash(t *testing.T) {
	dir := setupTestRepo(t)
	// Write a bad commit hash to the branch ref
	refs.WriteRef(dir, "refs/heads/main", "0000000000000000000000000000000000000000")

	err := Log()
	if err == nil {
		t.Fatal("expected error for bad commit hash")
	}
}

func TestLog_ResolveHeadError(t *testing.T) {
	dir := setupTestRepo(t)
	// Make the main ref a directory to cause ResolveHead to fail with a non-ENOENT error
	refPath := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.MkdirAll(filepath.Join(refPath, "subdir"), 0755)
	defer os.RemoveAll(refPath)

	err := Log()
	if err == nil {
		t.Fatal("expected error when ResolveHead fails")
	}
}
