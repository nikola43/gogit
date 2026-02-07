package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"gogit/refs"
	"gogit/repo"
)

func TestBranch_List(t *testing.T) {
	setupTestRepoWithCommit(t)

	if err := Branch(""); err != nil {
		t.Fatalf("Branch list failed: %v", err)
	}
}

func TestBranch_Create(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	if err := Branch("feature"); err != nil {
		t.Fatalf("Branch create failed: %v", err)
	}

	hash, _ := refs.ReadRef(dir, "refs/heads/feature")
	if hash == "" {
		t.Fatal("feature branch should exist")
	}

	headHash, _ := refs.ResolveHead(dir)
	if hash != headHash {
		t.Error("new branch should point to HEAD")
	}
}

func TestBranch_CreateAlreadyExists(t *testing.T) {
	setupTestRepoWithCommit(t)
	Branch("feature")

	err := Branch("feature")
	if err == nil {
		t.Fatal("expected error for duplicate branch")
	}
}

func TestBranch_CreateNoCommits(t *testing.T) {
	setupTestRepo(t)
	err := Branch("feature")
	if err == nil {
		t.Fatal("expected error when no commits")
	}
}

func TestBranch_ListMultiple(t *testing.T) {
	setupTestRepoWithCommit(t)
	Branch("alpha")
	Branch("beta")

	if err := Branch(""); err != nil {
		t.Fatalf("Branch list failed: %v", err)
	}
}

func TestBranch_ListShowsCurrent(t *testing.T) {
	setupTestRepoWithCommit(t)
	Branch("feature")

	if err := Branch(""); err != nil {
		t.Fatalf("Branch list failed: %v", err)
	}
}

func TestBranch_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Branch("")
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestBranch_ListBranchesError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Replace refs/heads with a file to trigger ReadDir error
	headsDir := filepath.Join(dir, repo.GogitDir, "refs", "heads")
	mainRef := filepath.Join(headsDir, "main")
	os.Remove(mainRef)
	os.RemoveAll(headsDir)
	os.WriteFile(headsDir, []byte("not a dir"), 0644)
	defer func() {
		os.Remove(headsDir)
		os.MkdirAll(headsDir, 0755)
	}()

	err := Branch("")
	if err == nil {
		t.Fatal("expected error for unreadable heads dir")
	}
}

func TestBranch_HeadError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.Remove(filepath.Join(dir, repo.GogitDir, "HEAD"))

	err := Branch("")
	if err == nil {
		t.Fatal("expected error when HEAD is missing")
	}
}

func TestBranch_CreateWriteRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Create a file where the ref needs to be
	blocker := filepath.Join(dir, repo.GogitDir, "refs", "heads", "newbranch")
	os.MkdirAll(blocker, 0755) // dir blocks file creation
	defer os.RemoveAll(blocker)

	err := Branch("newbranch")
	// Branch first checks if it exists (ReadRef) - a dir won't match
	// Then tries to WriteRef which calls WriteFile - dir blocks it
	if err == nil {
		t.Fatal("expected error when cannot write ref")
	}
}

func TestCreateBranch_WriteRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Make refs/heads read-only so WriteRef's WriteFile fails
	headsDir := filepath.Join(dir, repo.GogitDir, "refs", "heads")
	os.Chmod(headsDir, 0555)
	defer os.Chmod(headsDir, 0755)

	err := createBranch(dir, "newbranch")
	if err == nil {
		t.Fatal("expected error when cannot write ref")
	}
}

func TestCreateBranch_ReadRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Make the ref path a directory with contents to cause ReadRef error (not ENOENT)
	refPath := filepath.Join(dir, repo.GogitDir, "refs", "heads", "badref")
	os.MkdirAll(filepath.Join(refPath, "sub"), 0755)
	defer os.RemoveAll(refPath)

	err := createBranch(dir, "badref")
	if err == nil {
		t.Fatal("expected error when ReadRef fails")
	}
}

func TestBranch_CreateResolveHeadError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Replace HEAD with a directory-based path that causes ResolveHead error
	// But ReadRef is called first (to check if branch exists), and it reads refs/heads/newbranch.
	// ResolveHead reads HEAD. If HEAD is broken, ResolveHead fails.
	// But ReadRef for "newbranch" should work fine (returns "").
	// Replace the main ref (that HEAD points to) with a directory to cause ResolveHead to fail
	refPath := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.Remove(refPath)
	os.MkdirAll(filepath.Join(refPath, "subdir"), 0755)
	defer os.RemoveAll(refPath)

	err := Branch("newbranch")
	if err == nil {
		t.Fatal("expected error when ResolveHead fails in createBranch")
	}
}
