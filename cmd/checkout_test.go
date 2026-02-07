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

func TestCheckout_SwitchBranch(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "main_only.txt"), []byte("main"), 0644)
	Add([]string{"main_only.txt"})
	Commit("main commit")

	Checkout("feature")

	if _, err := os.Stat(filepath.Join(dir, "main_only.txt")); !os.IsNotExist(err) {
		t.Error("main_only.txt should not exist on feature branch")
	}

	branch, _ := refs.CurrentBranch(dir)
	if branch != "feature" {
		t.Errorf("expected branch 'feature', got '%s'", branch)
	}
}

func TestCheckout_SwitchBack(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")

	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0644)
	Add([]string{"feature.txt"})
	Commit("feature commit")

	Checkout("main")

	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
		t.Error("feature.txt should not exist on main")
	}
}

func TestCheckout_SameCommit(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	if err := Checkout("feature"); err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	branch, _ := refs.CurrentBranch(dir)
	if branch != "feature" {
		t.Errorf("expected 'feature', got '%s'", branch)
	}
}

func TestCheckout_BranchNotFound(t *testing.T) {
	setupTestRepoWithCommit(t)
	err := Checkout("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
}

func TestCheckout_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Checkout("main")
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestCheckout_WithSubdirectory(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	os.MkdirAll(filepath.Join(dir, "sub", "dir"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "dir", "file.txt"), []byte("nested"), 0644)
	Add([]string{"sub"})
	Commit("nested commit")

	Branch("feature")
	Checkout("feature")
	Checkout("main")

	if _, err := os.Stat(filepath.Join(dir, "sub", "dir", "file.txt")); err != nil {
		t.Error("nested file should exist on main")
	}
}

func TestCheckout_CleansEmptyDirs(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")

	os.MkdirAll(filepath.Join(dir, "deep", "nested"), 0755)
	os.WriteFile(filepath.Join(dir, "deep", "nested", "file.txt"), []byte("deep"), 0644)
	Add([]string{"deep"})
	Commit("add deep file")

	Checkout("main")

	if _, err := os.Stat(filepath.Join(dir, "deep")); !os.IsNotExist(err) {
		t.Error("deep directory should be cleaned up after checkout")
	}
}

func TestCheckout_SameCommitUpdateHeadError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	// Make HEAD read-only: ResolveHead can still read it,
	// but UpdateHead's WriteFile will fail
	headPath := filepath.Join(dir, repo.GogitDir, "HEAD")
	os.Chmod(headPath, 0444)
	defer os.Chmod(headPath, 0644)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when HEAD is read-only")
	}
}

func TestCheckout_NoCurrentCommit(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	// Create a branch with a commit by manually writing a ref
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})
	Commit("init")

	Branch("feature")

	// Point main to a new commit, then make feature have a different commit
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("more"), 0644)
	Add([]string{"file2.txt"})
	Commit("second")

	// Checkout feature (which is at the old commit)
	if err := Checkout("feature"); err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}
}

func TestCheckout_WriteIndexError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra"), 0644)
	Add([]string{"extra.txt"})
	Commit("extra commit")

	// Replace the index file with a directory to cause WriteFile to fail
	idxPath := filepath.Join(dir, repo.GogitDir, "index")
	os.Remove(idxPath)
	os.MkdirAll(idxPath, 0755)
	defer os.RemoveAll(idxPath)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when cannot write index")
	}
}

func TestCleanEmptyDirs(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(deep, 0755)

	cleanEmptyDirs(dir, deep)

	if _, err := os.Stat(filepath.Join(dir, "a")); !os.IsNotExist(err) {
		t.Error("empty dirs should be cleaned up")
	}
}

func TestCleanEmptyDirs_NonEmpty(t *testing.T) {
	dir := t.TempDir()
	mid := filepath.Join(dir, "a", "b")
	os.MkdirAll(mid, 0755)
	os.WriteFile(filepath.Join(dir, "a", "keep.txt"), []byte("keep"), 0644)

	cleanEmptyDirs(dir, mid)

	if _, err := os.Stat(filepath.Join(dir, "a")); err != nil {
		t.Error("dir 'a' should still exist (not empty)")
	}
}

func TestCleanEmptyDirs_RootDir(t *testing.T) {
	dir := t.TempDir()
	// When dir == root, should stop immediately
	cleanEmptyDirs(dir, dir)
	// dir should still exist
	if _, err := os.Stat(dir); err != nil {
		t.Error("root dir should still exist")
	}
}

func TestCheckout_ReadRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	// Replace the feature ref file with a directory to cause ReadRef to fail (non-ENOENT)
	refPath := filepath.Join(dir, repo.GogitDir, "refs", "heads", "feature")
	os.Remove(refPath)
	os.MkdirAll(filepath.Join(refPath, "subdir"), 0755)
	defer os.RemoveAll(refPath)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when ref file is a directory")
	}
}

func TestCheckout_ResolveHeadError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	// Add a commit on main so feature and main diverge
	os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra"), 0644)
	Add([]string{"extra.txt"})
	Commit("extra")

	// Remove HEAD to trigger error
	os.Remove(filepath.Join(dir, repo.GogitDir, "HEAD"))

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when HEAD is missing")
	}
}

func TestCheckout_BadCurrentCommit(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	Checkout("main")

	// Now corrupt main's commit by pointing it to a bad hash
	refs.WriteRef(dir, "refs/heads/main", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error for bad current commit")
	}
}

func TestCheckout_BadTargetCommit(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Point feature to a bad hash
	refs.WriteRef(dir, "refs/heads/feature", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error for bad target commit")
	}
}

func TestCheckout_NoCurrentCommits(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	// Manually create a branch with a commit, without being on it
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})
	Commit("init")

	commitHash, _ := refs.ResolveHead(dir)
	Branch("feature")

	// Now reset main ref to nothing (no commits on current branch)
	os.Remove(filepath.Join(dir, repo.GogitDir, "refs", "heads", "main"))

	// Checkout feature - currentHash will be "" (empty branch)
	if err := Checkout("feature"); err != nil {
		// This should succeed - empty current tree, switch to feature
		// But it might fail because HEAD points to main which has no ref
		// That's fine - just exercise the code path
		_ = err
	}

	// Alternative: directly test with a known setup
	// Write HEAD as detached pointing to empty
	refs.UpdateHead(dir, "ref: refs/heads/main")
	// Remove main ref so ResolveHead returns ""
	os.Remove(filepath.Join(dir, repo.GogitDir, "refs", "heads", "main"))
	// Put feature back
	refs.WriteRef(dir, "refs/heads/feature", commitHash)

	// This checkout: currentHash="" (no commits on main), branchHash=commitHash
	err := Checkout("feature")
	if err != nil {
		t.Fatalf("Checkout should succeed when current has no commits: %v", err)
	}
}

func TestCheckout_FlattenCurrentTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Get the tree hash from the commit
	mainHash, _ := refs.ResolveHead(dir)
	commit, _ := object.ReadCommit(dir, mainHash)
	treeHash := commit.TreeHash

	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	Checkout("main")

	// Delete the tree object for main's commit to trigger FlattenTree error
	treeObjPath := filepath.Join(dir, repo.GogitDir, "objects", treeHash[:2], treeHash[2:])
	os.Remove(treeObjPath)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when current tree is corrupted")
	}
}

func TestCheckout_FlattenTargetTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")

	// Get feature's tree hash
	featHash, _ := refs.ResolveHead(dir)
	featCommit, _ := object.ReadCommit(dir, featHash)
	treeHash := featCommit.TreeHash

	Checkout("main")

	// Delete the feature tree object
	treeObjPath := filepath.Join(dir, repo.GogitDir, "objects", treeHash[:2], treeHash[2:])
	os.Remove(treeObjPath)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when target tree is corrupted")
	}
}

func TestCheckout_ReadBlobError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("new content"), 0644)
	Add([]string{"newfile.txt"})
	Commit("add new file")

	// Get the blob hash for newfile.txt
	idx, _ := index.ReadIndex(dir)
	var blobHash string
	for _, e := range idx.Entries {
		if e.Path == "newfile.txt" {
			blobHash = e.Hash
			break
		}
	}

	Checkout("main")

	// Delete the blob object for newfile.txt
	blobObjPath := filepath.Join(dir, repo.GogitDir, "objects", blobHash[:2], blobHash[2:])
	os.Remove(blobObjPath)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when blob is missing")
	}
}

func TestCheckout_MkdirError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "f.txt"), []byte("f"), 0644)
	Add([]string{"sub"})
	Commit("feat")
	Checkout("main")

	// Create a file that blocks the "sub" directory creation
	os.WriteFile(filepath.Join(dir, "sub"), []byte("blocker"), 0644)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when dir creation blocked")
	}
}

func TestCheckout_WriteFileError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	Checkout("main")

	// Create a directory where f.txt should be (blocks file write)
	os.MkdirAll(filepath.Join(dir, "f.txt", "sub"), 0755)
	defer os.RemoveAll(filepath.Join(dir, "f.txt"))

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when file write is blocked")
	}
}

func TestCheckout_UpdateHeadAfterTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	Checkout("main")

	// Replace HEAD with a directory to cause UpdateHead to fail
	headPath := filepath.Join(dir, repo.GogitDir, "HEAD")
	os.Remove(headPath)
	os.MkdirAll(headPath, 0755)
	defer os.RemoveAll(headPath)

	err := Checkout("feature")
	if err == nil {
		t.Fatal("expected error when cannot update HEAD after tree update")
	}
}

func TestCleanEmptyDirs_GogitDir(t *testing.T) {
	dir := t.TempDir()
	gogitDir := filepath.Join(dir, ".gogit", "some", "nested")
	os.MkdirAll(gogitDir, 0755)
	// Should stop at .gogit
	cleanEmptyDirs(dir, gogitDir)
	if _, err := os.Stat(filepath.Join(dir, ".gogit")); err != nil {
		t.Error(".gogit should still exist")
	}
}

func TestUpdateWorkingTree_WriteIndexError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	targetTree := map[string]string{
		"file.txt": object.HashBlob([]byte("content")),
	}
	// Write blob so ReadBlob succeeds
	object.WriteBlob(dir, []byte("content"))

	// Replace index with a directory to cause WriteIndex to fail
	idxPath := filepath.Join(dir, repo.GogitDir, "index")
	os.Remove(idxPath)
	os.MkdirAll(idxPath, 0755)
	defer os.RemoveAll(idxPath)

	err := updateWorkingTree(dir, "test", map[string]string{}, targetTree)
	if err == nil {
		t.Fatal("expected error when cannot write index")
	}
}

func TestUpdateWorkingTree_UpdateHeadError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	targetTree := map[string]string{
		"file.txt": object.HashBlob([]byte("content")),
	}
	object.WriteBlob(dir, []byte("content"))

	// Replace HEAD with directory to cause UpdateHead to fail
	headPath := filepath.Join(dir, repo.GogitDir, "HEAD")
	os.Remove(headPath)
	os.MkdirAll(headPath, 0755)
	defer os.RemoveAll(headPath)

	err := updateWorkingTree(dir, "test", map[string]string{}, targetTree)
	if err == nil {
		t.Fatal("expected error when cannot update HEAD")
	}
}

func TestUpdateWorkingTree_ReadBlobError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	targetTree := map[string]string{
		"file.txt": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
	}

	err := updateWorkingTree(dir, "test", map[string]string{}, targetTree)
	if err == nil {
		t.Fatal("expected error when blob is missing")
	}
}

func TestUpdateWorkingTree_MkdirError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	object.WriteBlob(dir, []byte("content"))
	blobHash := object.HashBlob([]byte("content"))

	targetTree := map[string]string{
		"sub/file.txt": blobHash,
	}

	// Block sub dir creation
	os.WriteFile(filepath.Join(dir, "sub"), []byte("blocker"), 0644)

	err := updateWorkingTree(dir, "test", map[string]string{}, targetTree)
	if err == nil {
		t.Fatal("expected error when mkdir is blocked")
	}
}

func TestUpdateWorkingTree_WriteFileError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	object.WriteBlob(dir, []byte("content"))
	blobHash := object.HashBlob([]byte("content"))

	targetTree := map[string]string{
		"f.txt": blobHash,
	}

	// Block file write by creating a directory
	os.MkdirAll(filepath.Join(dir, "f.txt", "sub"), 0755)
	defer os.RemoveAll(filepath.Join(dir, "f.txt"))

	err := updateWorkingTree(dir, "test", map[string]string{}, targetTree)
	if err == nil {
		t.Fatal("expected error when file write is blocked")
	}
}
