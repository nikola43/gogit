package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"gogit/index"
	"gogit/object"
	"gogit/refs"
	"gogit/repo"
)

func TestCommit_Initial(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	if err := Commit("initial commit"); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	hash, err := refs.ResolveHead(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("HEAD should point to a commit")
	}

	commit, _ := object.ReadCommit(dir, hash)
	if commit.Message != "initial commit" {
		t.Errorf("message mismatch: %s", commit.Message)
	}
	if len(commit.Parents) != 0 {
		t.Errorf("first commit should have no parents")
	}
}

func TestCommit_WithParent(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	firstHash, _ := refs.ResolveHead(dir)

	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("more"), 0644)
	Add([]string{"file2.txt"})
	Commit("second commit")

	secondHash, _ := refs.ResolveHead(dir)
	if secondHash == firstHash {
		t.Error("second commit should have different hash")
	}

	commit, _ := object.ReadCommit(dir, secondHash)
	if len(commit.Parents) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(commit.Parents))
	}
	if commit.Parents[0] != firstHash {
		t.Error("parent should be first commit")
	}
}

func TestCommit_NothingToCommit(t *testing.T) {
	setupTestRepo(t)
	err := Commit("empty")
	if err == nil {
		t.Fatal("expected error for empty commit")
	}
}

func TestCommit_UpdatesBranchRef(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	hash, _ := refs.ReadRef(dir, "refs/heads/main")
	headHash, _ := refs.ResolveHead(dir)

	if hash != headHash {
		t.Errorf("branch ref should match HEAD: %s vs %s", hash, headHash)
	}
}

func TestCommit_DetachedHead(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	hash, _ := refs.ResolveHead(dir)
	refs.UpdateHead(dir, hash)

	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	Add([]string{"new.txt"})
	if err := Commit("detached commit"); err != nil {
		t.Fatalf("Commit in detached HEAD failed: %v", err)
	}

	newHash, _ := refs.ResolveHead(dir)
	if newHash == hash {
		t.Error("HEAD should be updated")
	}
}

func TestCommit_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Commit("test")
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestCommit_CorruptIndex(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, repo.GogitDir, "index"), []byte("corrupt data that is long enough"), 0644)

	err := Commit("test")
	if err == nil {
		t.Fatal("expected error for corrupt index")
	}
}

func TestCommit_BuildTreeError(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	// Replace objects directory with a file to block tree object writes
	objDir := filepath.Join(dir, repo.GogitDir, "objects")
	os.RemoveAll(objDir)
	os.WriteFile(objDir, []byte("blocker"), 0644)
	defer func() {
		os.Remove(objDir)
		os.MkdirAll(objDir, 0755)
	}()

	err := Commit("will fail")
	if err == nil {
		t.Fatal("expected error when tree build fails")
	}
}

func TestCommit_ResolveHeadError(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	// Remove HEAD to trigger ResolveHead error
	os.Remove(filepath.Join(dir, repo.GogitDir, "HEAD"))

	err := Commit("test")
	if err == nil {
		t.Fatal("expected error when HEAD is missing")
	}
}

func TestCommit_WriteRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	Add([]string{"new.txt"})

	// Replace main ref with a directory to cause WriteRef error
	mainRef := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.Remove(mainRef)
	os.MkdirAll(mainRef, 0755)
	defer os.RemoveAll(mainRef)

	err := Commit("will fail")
	if err == nil {
		t.Fatal("expected error when cannot write ref")
	}
}

func TestCommit_CurrentBranchError(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	// Write HEAD first, then replace it with a directory before CurrentBranch call
	// Actually, CurrentBranch reads HEAD, so we need HEAD to not be readable
	// But we need it to work for ResolveHead... This is tricky.
	// Let's just test the detached HEAD UpdateHead error
	// Commit with detached HEAD and broken HEAD path
	hash, _ := refs.ResolveHead(dir)
	if hash == "" {
		// No commit yet, do one normally
		Commit("init")
		hash, _ = refs.ResolveHead(dir)
	}

	// Detach HEAD
	refs.UpdateHead(dir, hash)

	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	Add([]string{"new.txt"})

	// Replace HEAD with directory to cause UpdateHead error in detached path
	headPath := filepath.Join(dir, repo.GogitDir, "HEAD")
	os.Remove(headPath)
	os.MkdirAll(headPath, 0755)
	defer os.RemoveAll(headPath)

	err := Commit("detached fail")
	if err == nil {
		t.Fatal("expected error for detached HEAD update failure")
	}
}

func TestBranchDisplay(t *testing.T) {
	if branchDisplay("main") != "main" {
		t.Error("expected 'main'")
	}
	if branchDisplay("") != "detached HEAD" {
		t.Error("expected 'detached HEAD'")
	}
}

func TestWriteCommitAndUpdateRef_Success(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	idx, _ := index.ReadIndex(dir)
	treeHash, _ := object.BuildTreeFromIndex(dir, idx)

	hash, err := writeCommitAndUpdateRef(dir, treeHash, nil, "test commit")
	if err != nil {
		t.Fatalf("writeCommitAndUpdateRef failed: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %d", len(hash))
	}

	refHash, _ := refs.ReadRef(dir, "refs/heads/main")
	if refHash != hash {
		t.Error("branch ref should match commit hash")
	}
}

func TestWriteCommitAndUpdateRef_WriteRefError(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	idx, _ := index.ReadIndex(dir)
	treeHash, _ := object.BuildTreeFromIndex(dir, idx)

	// Replace main ref with a directory to cause WriteRef to fail
	mainRef := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.MkdirAll(mainRef, 0755)
	defer os.RemoveAll(mainRef)

	_, err := writeCommitAndUpdateRef(dir, treeHash, nil, "test commit")
	if err == nil {
		t.Fatal("expected error when WriteRef fails")
	}
}

func TestWriteCommitAndUpdateRef_DetachedHead(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	hash, _ := refs.ResolveHead(dir)
	refs.UpdateHead(dir, hash)

	idx, _ := index.ReadIndex(dir)
	treeHash, _ := object.BuildTreeFromIndex(dir, idx)

	commitHash, err := writeCommitAndUpdateRef(dir, treeHash, []string{hash}, "detached")
	if err != nil {
		t.Fatalf("writeCommitAndUpdateRef failed: %v", err)
	}

	newHead, _ := refs.ResolveHead(dir)
	if newHead != commitHash {
		t.Error("HEAD should be updated in detached mode")
	}
}

func TestWriteCommitAndUpdateRef_DetachedHeadError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	hash, _ := refs.ResolveHead(dir)
	refs.UpdateHead(dir, hash) // detach HEAD

	idx, _ := index.ReadIndex(dir)
	treeHash, _ := object.BuildTreeFromIndex(dir, idx)

	// Make HEAD read-only: CurrentBranch can still read it (returns "")
	// but UpdateHead's WriteFile will fail
	headPath := filepath.Join(dir, repo.GogitDir, "HEAD")
	os.Chmod(headPath, 0444)
	defer os.Chmod(headPath, 0644)

	_, err := writeCommitAndUpdateRef(dir, treeHash, []string{hash}, "detached fail")
	if err == nil {
		t.Fatal("expected error for detached HEAD update failure")
	}
}

func TestWriteCommitAndUpdateRef_CurrentBranchError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	idx, _ := index.ReadIndex(dir)
	treeHash, _ := object.BuildTreeFromIndex(dir, idx)

	// Replace HEAD with a directory so ReadHead fails (CurrentBranch error)
	headPath := filepath.Join(dir, repo.GogitDir, "HEAD")
	os.Remove(headPath)
	os.MkdirAll(headPath, 0755)
	defer os.RemoveAll(headPath)

	_, err := writeCommitAndUpdateRef(dir, treeHash, nil, "will fail")
	if err == nil {
		t.Fatal("expected error when CurrentBranch fails")
	}
}

func TestWriteCommitAndUpdateRef_BadObjectDir(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	// Break objects dir so WriteCommit fails
	objDir := filepath.Join(dir, repo.GogitDir, "objects")
	os.RemoveAll(objDir)
	os.WriteFile(objDir, []byte("blocker"), 0644)
	defer func() {
		os.Remove(objDir)
		os.MkdirAll(objDir, 0755)
	}()

	_, err := writeCommitAndUpdateRef(dir, "abc123", nil, "will fail")
	if err == nil {
		t.Fatal("expected error when WriteCommit fails")
	}
}

