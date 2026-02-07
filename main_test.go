package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupMainTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	if code := run([]string{"gogit", "init"}); code != 0 {
		t.Fatalf("init failed with code %d", code)
	}
	return dir
}

func TestRun_NoArgs(t *testing.T) {
	code := run([]string{"gogit"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	code := run([]string{"gogit", "unknown"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_Init(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	code := run([]string{"gogit", "init"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_AddNoArgs(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "add"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_Add(t *testing.T) {
	dir := setupMainTestRepo(t)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644)

	code := run([]string{"gogit", "add", "f.txt"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_Status(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "status"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_CommitNoMessage(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "commit"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_CommitNoMFlag(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "commit", "no-flag"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_CommitMFlagNoValue(t *testing.T) {
	setupMainTestRepo(t)
	// -m at the end with no following value
	code := run([]string{"gogit", "commit", "-m"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_Commit(t *testing.T) {
	dir := setupMainTestRepo(t)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644)
	run([]string{"gogit", "add", "f.txt"})

	code := run([]string{"gogit", "commit", "-m", "test"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_CommitError(t *testing.T) {
	setupMainTestRepo(t)
	// Nothing staged → error
	code := run([]string{"gogit", "commit", "-m", "empty"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_Log(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "log"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_Diff(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "diff"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_BranchNoArgs(t *testing.T) {
	dir := setupMainTestRepo(t)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644)
	run([]string{"gogit", "add", "f.txt"})
	run([]string{"gogit", "commit", "-m", "init"})

	code := run([]string{"gogit", "branch"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_BranchWithName(t *testing.T) {
	dir := setupMainTestRepo(t)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644)
	run([]string{"gogit", "add", "f.txt"})
	run([]string{"gogit", "commit", "-m", "init"})

	code := run([]string{"gogit", "branch", "feature"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_CheckoutNoArgs(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "checkout"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_Checkout(t *testing.T) {
	dir := setupMainTestRepo(t)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644)
	run([]string{"gogit", "add", "f.txt"})
	run([]string{"gogit", "commit", "-m", "init"})
	run([]string{"gogit", "branch", "feature"})

	code := run([]string{"gogit", "checkout", "feature"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_MergeNoArgs(t *testing.T) {
	setupMainTestRepo(t)
	code := run([]string{"gogit", "merge"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_Merge(t *testing.T) {
	dir := setupMainTestRepo(t)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644)
	run([]string{"gogit", "add", "f.txt"})
	run([]string{"gogit", "commit", "-m", "init"})
	run([]string{"gogit", "branch", "feature"})

	code := run([]string{"gogit", "merge", "feature"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestUsage(t *testing.T) {
	// Just make sure it doesn't panic
	usage()
}
