package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temp directory, chdirs into it, inits a gogit repo,
// and returns the directory path. Cleanup restores the original cwd.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	return dir
}

// setupTestRepoWithCommit creates a repo with one committed file.
// Returns (dir, commitHash).
func setupTestRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := setupTestRepo(t)

	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello\n"), 0644)
	if err := Add([]string{"test.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := Commit("initial commit"); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	return dir
}
