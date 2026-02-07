package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"gogit/object"
	"gogit/refs"
	"gogit/repo"
)

func TestStatus_CleanRepo(t *testing.T) {
	setupTestRepoWithCommit(t)

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_UntrackedFiles(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("new"), 0644)

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_StagedNewFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	Add([]string{"new.txt"})

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_StagedModifiedFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("modified\n"), 0644)
	Add([]string{"test.txt"})

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_StagedDeletedFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.Remove(filepath.Join(dir, "test.txt"))
	Add([]string{"test.txt"})

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_UnstagedModified(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("changed\n"), 0644)

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_UnstagedDeleted(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.Remove(filepath.Join(dir, "test.txt"))

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_NoCommitsYet(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_DetachedHead(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	hash, _ := refs.ResolveHead(dir)
	refs.UpdateHead(dir, hash)

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Status()
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestStatus_EmptyWorkingTree(t *testing.T) {
	setupTestRepo(t)
	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_CorruptIndex(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, repo.GogitDir, "index"), []byte("corrupt data that is long enough"), 0644)

	err := Status()
	if err == nil {
		t.Fatal("expected error for corrupt index")
	}
}

func TestStatus_BadHeadRef(t *testing.T) {
	dir := setupTestRepo(t)
	// Write a HEAD pointing to a ref, then write a bad hash in that ref
	refs.WriteRef(dir, "refs/heads/main", "0000000000000000000000000000000000000000")

	err := Status()
	if err == nil {
		t.Fatal("expected error for bad commit in HEAD")
	}
}

func TestStatus_UnreadableWorkingFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	f := filepath.Join(dir, "test.txt")
	os.Chmod(f, 0000)
	defer os.Chmod(f, 0644)

	// Should still run (the error for unreadable file in unstaged check is handled with continue)
	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_ResolveHeadError(t *testing.T) {
	dir := setupTestRepo(t)
	// Write HEAD pointing to a ref whose file is a directory (causes non-ENOENT error)
	refPath := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.MkdirAll(filepath.Join(refPath, "subdir"), 0755)
	defer os.RemoveAll(refPath)

	err := Status()
	if err == nil {
		t.Fatal("expected error when HEAD resolution fails")
	}
}

func TestStatus_BadHeadCommit(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Point HEAD to a non-existent commit
	refs.WriteRef(dir, "refs/heads/main", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	err := Status()
	if err == nil {
		t.Fatal("expected error for non-existent commit")
	}
}

func TestStatus_BadTreeHash(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	hash, _ := refs.ResolveHead(dir)

	// Read the commit object, then delete the tree object
	commit, _ := object.ReadCommit(dir, hash)
	treeObjPath := filepath.Join(dir, repo.GogitDir, "objects", commit.TreeHash[:2], commit.TreeHash[2:])
	os.Remove(treeObjPath)

	err := Status()
	if err == nil {
		t.Fatal("expected error for bad tree hash")
	}
}

func TestStatus_GogitDirSkipped(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Add extra files outside .gogit to ensure untracked scan works
	os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra"), 0644)

	if err := Status(); err != nil {
		t.Fatalf("Status failed: %v", err)
	}
}

func TestStatus_CurrentBranchError(t *testing.T) {
	dir := setupTestRepo(t)
	// Write a HEAD file that causes CurrentBranch to fail
	// CurrentBranch reads HEAD. Replace HEAD with a directory.
	headPath := filepath.Join(dir, repo.GogitDir, "HEAD")
	os.Remove(headPath)
	os.MkdirAll(headPath, 0755)
	defer os.RemoveAll(headPath)

	err := Status()
	if err == nil {
		t.Fatal("expected error when CurrentBranch fails")
	}
}

func TestStatus_WalkError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Create an unreadable subdirectory to trigger walk error callback
	unreadable := filepath.Join(dir, "blocked")
	os.MkdirAll(unreadable, 0755)
	os.WriteFile(filepath.Join(unreadable, "f.txt"), []byte("f"), 0644)
	os.Chmod(unreadable, 0000)
	defer os.Chmod(unreadable, 0755)

	// Status should still succeed (walk error causes return nil)
	if err := Status(); err != nil {
		t.Fatalf("Status should handle walk error: %v", err)
	}
}
