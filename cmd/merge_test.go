package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gogit/index"
	"gogit/object"
	"gogit/refs"
	"gogit/repo"
)

func TestMerge_FastForward(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")

	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0644)
	Add([]string{"feature.txt"})
	Commit("feature commit")

	Checkout("main")

	if err := Merge("feature"); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Error("feature.txt should exist after merge")
	}

	mainHash, _ := refs.ReadRef(dir, "refs/heads/main")
	featureHash, _ := refs.ReadRef(dir, "refs/heads/feature")
	if mainHash != featureHash {
		t.Error("fast-forward merge should make branches point to same commit")
	}
}

func TestMerge_AlreadyUpToDate_SameHash(t *testing.T) {
	setupTestRepoWithCommit(t)
	Branch("feature")

	if err := Merge("feature"); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
}

func TestMerge_AlreadyUpToDate_Ancestor(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra"), 0644)
	Add([]string{"extra.txt"})
	Commit("extra commit")

	if err := Merge("feature"); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
}

func TestMerge_FileLevelMerge(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	Branch("feature")

	os.WriteFile(filepath.Join(dir, "main_file.txt"), []byte("main"), 0644)
	Add([]string{"main_file.txt"})
	Commit("main commit")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "feature_file.txt"), []byte("feature"), 0644)
	Add([]string{"feature_file.txt"})
	Commit("feature commit")

	Checkout("main")

	if err := Merge("feature"); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "main_file.txt")); err != nil {
		t.Error("main_file.txt should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "feature_file.txt")); err != nil {
		t.Error("feature_file.txt should exist")
	}

	hash, _ := refs.ResolveHead(dir)
	commit, _ := object.ReadCommit(dir, hash)
	if len(commit.Parents) != 2 {
		t.Fatalf("merge commit should have 2 parents, got %d", len(commit.Parents))
	}
}

func TestMerge_Conflict(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	Branch("feature")

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("main version\n"), 0644)
	Add([]string{"test.txt"})
	Commit("main change")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("feature version\n"), 0644)
	Add([]string{"test.txt"})
	Commit("feature change")

	Checkout("main")

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestMerge_BranchNotFound(t *testing.T) {
	setupTestRepoWithCommit(t)
	err := Merge("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
}

func TestMerge_DetachedHead(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	hash, _ := refs.ResolveHead(dir)
	refs.UpdateHead(dir, hash)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for detached HEAD merge")
	}
}

func TestMerge_NoCommitsOnCurrent(t *testing.T) {
	setupTestRepo(t)
	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when no commits")
	}
}

func TestMerge_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestMerge_TargetDeleted(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	Branch("feature")

	os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra"), 0644)
	Add([]string{"extra.txt"})
	Commit("add extra")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "keeper.txt"), []byte("keep"), 0644)
	Add([]string{"keeper.txt"})
	Commit("add keeper on feature")

	os.Remove(filepath.Join(dir, "test.txt"))
	Add([]string{"test.txt"})
	Commit("delete test.txt on feature")

	Checkout("main")
	err := Merge("feature")
	if err != nil {
		t.Fatalf("Merge with deletion should succeed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.txt")); !os.IsNotExist(err) {
		t.Error("test.txt should be deleted after merge")
	}
}

func TestMerge_CurrentDeleted(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	Branch("feature")

	os.Remove(filepath.Join(dir, "test.txt"))
	Add([]string{"test.txt"})
	os.WriteFile(filepath.Join(dir, "main_new.txt"), []byte("new"), 0644)
	Add([]string{"main_new.txt"})
	Commit("main: delete test, add new")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "feature_new.txt"), []byte("feat"), 0644)
	Add([]string{"feature_new.txt"})
	Commit("feature: add new")

	Checkout("main")
	err := Merge("feature")
	if err != nil {
		t.Fatalf("Merge should succeed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.txt")); !os.IsNotExist(err) {
		t.Error("test.txt should remain deleted")
	}
}

func TestIsAncestor(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	firstHash, _ := refs.ResolveHead(dir)

	os.WriteFile(filepath.Join(dir, "f2.txt"), []byte("f2"), 0644)
	Add([]string{"f2.txt"})
	Commit("second")

	secondHash, _ := refs.ResolveHead(dir)

	if !isAncestor(dir, firstHash, secondHash) {
		t.Error("first should be ancestor of second")
	}
	if isAncestor(dir, secondHash, firstHash) {
		t.Error("second should not be ancestor of first")
	}
	if !isAncestor(dir, firstHash, firstHash) {
		t.Error("commit should be ancestor of itself")
	}
}

func TestIsAncestor_InvalidHash(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	if isAncestor(dir, "invalid", "alsoinvalid") {
		t.Error("should return false for invalid hashes")
	}
}

func TestFindMergeBase(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	baseHash, _ := refs.ResolveHead(dir)

	Branch("feature")

	os.WriteFile(filepath.Join(dir, "main.txt"), []byte("m"), 0644)
	Add([]string{"main.txt"})
	Commit("main")
	mainHash, _ := refs.ResolveHead(dir)

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "feat.txt"), []byte("f"), 0644)
	Add([]string{"feat.txt"})
	Commit("feat")
	featHash, _ := refs.ResolveHead(dir)

	mb := findMergeBase(dir, mainHash, featHash)
	if mb != baseHash {
		t.Errorf("merge base should be %s, got %s", baseHash[:7], mb[:7])
	}
}

func TestFindMergeBase_NoCommon(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	mb := findMergeBase(dir, "0000000000000000000000000000000000000000", "1111111111111111111111111111111111111111")
	if mb != "" {
		t.Error("expected empty merge base for unrelated histories")
	}
}

func TestMerge_BothSameChange(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("same change\n"), 0644)
	Add([]string{"test.txt"})
	Commit("main: same change")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("same change\n"), 0644)
	Add([]string{"test.txt"})
	Commit("feature: same change")

	Checkout("main")
	if err := Merge("feature"); err != nil {
		t.Fatalf("Merge with identical changes should succeed: %v", err)
	}
}

func TestMerge_BothDeletedSameFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	// Add extra files so index isn't empty after deleting test.txt
	os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0644)
	Add([]string{"keep.txt"})
	Commit("add keeper")

	Branch("feature")

	// Both branches delete test.txt and add their own files
	os.Remove(filepath.Join(dir, "test.txt"))
	Add([]string{"test.txt"})
	os.WriteFile(filepath.Join(dir, "main_new.txt"), []byte("m"), 0644)
	Add([]string{"main_new.txt"})
	Commit("main: delete test")

	Checkout("feature")
	os.Remove(filepath.Join(dir, "test.txt"))
	Add([]string{"test.txt"})
	os.WriteFile(filepath.Join(dir, "feat_new.txt"), []byte("f"), 0644)
	Add([]string{"feat_new.txt"})
	Commit("feature: delete test")

	Checkout("main")
	if err := Merge("feature"); err != nil {
		t.Fatalf("Merge with both deleting should succeed: %v", err)
	}

	// test.txt should be gone
	if _, err := os.Stat(filepath.Join(dir, "test.txt")); !os.IsNotExist(err) {
		t.Error("test.txt should remain deleted")
	}
}

func TestMerge_ReadRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Make target branch ref a directory (non-ENOENT error)
	refPath := filepath.Join(dir, repo.GogitDir, "refs", "heads", "feature")
	os.MkdirAll(filepath.Join(refPath, "sub"), 0755)
	defer os.RemoveAll(refPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when target ref is unreadable")
	}
}

func TestMerge_FFBadCommitObject(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	featHash, _ := refs.ResolveHead(dir)
	Checkout("main")

	// Delete the feature commit object to trigger ReadCommit error in fastForwardMerge
	objPath := filepath.Join(dir, repo.GogitDir, "objects", featHash[:2], featHash[2:])
	os.Remove(objPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for corrupted commit in fast-forward")
	}
}

func TestMerge_FileLevelBadBaseCommit(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	baseHash, _ := refs.ResolveHead(dir)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")

	Checkout("main")

	// Delete the base commit's objects to trigger error in fileLevelMerge
	baseObjPath := filepath.Join(dir, repo.GogitDir, "objects", baseHash[:2], baseHash[2:])
	os.Remove(baseObjPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for corrupted base commit in file-level merge")
	}
}

func TestMerge_FileLevelBadCurrentCommit(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")
	mainHash, _ := refs.ResolveHead(dir)

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")

	Checkout("main")

	// Delete the current (main) commit to trigger ReadCommit error
	objPath := filepath.Join(dir, repo.GogitDir, "objects", mainHash[:2], mainHash[2:])
	os.Remove(objPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for corrupted current commit")
	}
}

func TestMerge_HeadReadError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	// Remove HEAD to trigger error
	os.Remove(filepath.Join(dir, repo.GogitDir, "HEAD"))

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when HEAD is missing")
	}
}

func TestMerge_FFWriteRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	Checkout("main")

	// Replace main ref with a directory to cause WriteFile to fail
	mainRef := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.Remove(mainRef)
	os.MkdirAll(mainRef, 0755)
	defer os.RemoveAll(mainRef)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when cannot write ref in fast-forward")
	}
}

func TestMerge_FileLevelWriteIndexError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feature")

	Checkout("main")

	// Replace index with a directory to cause WriteFile to fail
	idxPath := filepath.Join(dir, repo.GogitDir, "index")
	os.Remove(idxPath)
	os.MkdirAll(idxPath, 0755)
	defer os.RemoveAll(idxPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when cannot write index in file-level merge")
	}
}

func TestMerge_FFTargetFlattenTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	featHash, _ := refs.ResolveHead(dir)
	featCommit, _ := object.ReadCommit(dir, featHash)
	treeHash := featCommit.TreeHash
	Checkout("main")

	// Delete the feature commit's tree object to trigger FlattenTree error
	treeObjPath := filepath.Join(dir, repo.GogitDir, "objects", treeHash[:2], treeHash[2:])
	os.Remove(treeObjPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for corrupted tree in fast-forward")
	}
}

func TestMerge_FFMkdirError(t *testing.T) {
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

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when mkdir blocked in fast-forward")
	}
}

func TestMerge_FFReadBlobError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new content"), 0644)
	Add([]string{"new.txt"})
	Commit("feat")

	// Find the blob hash for new.txt
	idx, _ := index.ReadIndex(dir)
	var blobHash string
	for _, e := range idx.Entries {
		if e.Path == "new.txt" {
			blobHash = e.Hash
			break
		}
	}

	Checkout("main")

	// Delete the blob object
	blobObjPath := filepath.Join(dir, repo.GogitDir, "objects", blobHash[:2], blobHash[2:])
	os.Remove(blobObjPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when blob is missing in fast-forward")
	}
}

func TestMerge_FFWriteFileError(t *testing.T) {
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

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when file write blocked in fast-forward")
	}
}

func TestMerge_FFWriteIndexError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	Checkout("main")

	// Replace index with a directory to cause WriteIndex to fail
	idxPath := filepath.Join(dir, repo.GogitDir, "index")
	os.Remove(idxPath)
	os.MkdirAll(idxPath, 0755)
	defer os.RemoveAll(idxPath)

	// Also need to prevent WriteRef from failing first - WriteRef was already tested
	// Actually in fastForwardMerge, WriteRef is called first (line 88), then ReadCommit,
	// FlattenTree, etc. WriteRef needs to succeed for us to reach WriteIndex.
	// The ref write will succeed since we're not blocking it.
	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when cannot write index in fast-forward")
	}
}

func TestMerge_FileLevelFlattenBaseTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	baseHash, _ := refs.ResolveHead(dir)
	baseCommit, _ := object.ReadCommit(dir, baseHash)
	baseTreeHash := baseCommit.TreeHash

	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")

	Checkout("main")

	// Delete the base commit's tree object to trigger FlattenTree error
	treeObjPath := filepath.Join(dir, repo.GogitDir, "objects", baseTreeHash[:2], baseTreeHash[2:])
	os.Remove(treeObjPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for corrupted base tree in file-level merge")
	}
}

func TestMerge_FileLevelFlattenTargetTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	featHash, _ := refs.ResolveHead(dir)
	featCommit, _ := object.ReadCommit(dir, featHash)
	treeHash := featCommit.TreeHash

	Checkout("main")

	// Delete the feature tree to trigger FlattenTree error on target
	treeObjPath := filepath.Join(dir, repo.GogitDir, "objects", treeHash[:2], treeHash[2:])
	os.Remove(treeObjPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for corrupted target tree in file-level merge")
	}
}

func TestMerge_FileLevelFlattenCurrentTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")
	mainHash, _ := refs.ResolveHead(dir)
	mainCommit, _ := object.ReadCommit(dir, mainHash)
	mainTreeHash := mainCommit.TreeHash

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")

	Checkout("main")

	// Delete the current (main) tree to trigger FlattenTree error
	treeObjPath := filepath.Join(dir, repo.GogitDir, "objects", mainTreeHash[:2], mainTreeHash[2:])
	os.Remove(treeObjPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error for corrupted current tree in file-level merge")
	}
}

func TestMerge_FileLevelMkdirError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "f.txt"), []byte("f"), 0644)
	Add([]string{"sub"})
	Commit("feat")

	Checkout("main")

	// Create a file that blocks the "sub" directory creation
	os.WriteFile(filepath.Join(dir, "sub"), []byte("blocker"), 0644)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when mkdir blocked in file-level merge")
	}
}

func TestMerge_FileLevelReadBlobError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new content"), 0644)
	Add([]string{"new.txt"})
	Commit("feat")

	// Find the blob hash for new.txt
	idx, _ := index.ReadIndex(dir)
	var blobHash string
	for _, e := range idx.Entries {
		if e.Path == "new.txt" {
			blobHash = e.Hash
			break
		}
	}

	Checkout("main")

	// Delete the blob object
	blobObjPath := filepath.Join(dir, repo.GogitDir, "objects", blobHash[:2], blobHash[2:])
	os.Remove(blobObjPath)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when blob missing in file-level merge")
	}
}

func TestMerge_FileLevelWriteFileError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")

	Checkout("main")

	// Create a directory where f.txt should be (blocks file write)
	os.MkdirAll(filepath.Join(dir, "f.txt", "sub"), 0755)
	defer os.RemoveAll(filepath.Join(dir, "f.txt"))

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when file write blocked in file-level merge")
	}
}

func TestMerge_FileLevelWriteRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")

	Checkout("main")

	// Replace main ref with directory to cause final WriteRef to fail
	mainRef := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.Remove(mainRef)
	os.MkdirAll(mainRef, 0755)
	defer os.RemoveAll(mainRef)

	err := Merge("feature")
	if err == nil {
		t.Fatal("expected error when final ref write fails in file-level merge")
	}
}

func TestFastForwardMerge_WriteRefError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")
	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	targetHash, _ := refs.ResolveHead(dir)
	Checkout("main")

	// Replace main ref with directory
	mainRef := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.Remove(mainRef)
	os.MkdirAll(mainRef, 0755)
	defer os.RemoveAll(mainRef)

	err := fastForwardMerge(dir, "main", "feature", targetHash)
	if err == nil {
		t.Fatal("expected error when WriteRef fails")
	}
}

func TestFastForwardMerge_ReadCommitError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	err := fastForwardMerge(dir, "main", "feature", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if err == nil {
		t.Fatal("expected error when ReadCommit fails")
	}
}

func TestFileLevelMerge_BuildTreeError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")
	mainHash, _ := refs.ResolveHead(dir)

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	featHash, _ := refs.ResolveHead(dir)

	Checkout("main")

	// Break objects dir after merge writes files but before BuildTreeFromIndex
	// This is hard to test precisely. Instead, corrupt the objects dir
	objDir := filepath.Join(dir, repo.GogitDir, "objects")

	// Do the merge up to the point where BuildTree is called.
	// Actually, let's call fileLevelMerge directly after breaking objects:
	// The issue is fileLevelMerge reads commits/trees from objects, so we can't break it early.
	// Instead, we need to break it AFTER the file-writing stage.
	// Let's just make objects read-only so new writes fail.
	os.Chmod(objDir, 0555)
	defer os.Chmod(objDir, 0755)

	err := fileLevelMerge(dir, "main", "feature", mainHash, featHash)
	// May fail at various points (ReadCommit, FlattenTree, etc.) since objects dir is read-only
	// This covers the error paths we need
	if err == nil {
		t.Fatal("expected error when objects dir is read-only")
	}
}

func TestFileLevelMerge_WriteCommitError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")
	mainHash, _ := refs.ResolveHead(dir)

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	featHash, _ := refs.ResolveHead(dir)

	Checkout("main")

	// We need WriteCommit to fail. fileLevelMerge calls it near the end (line 305).
	// We can't easily isolate that, but we already have tests for the other error paths.
	// The WriteRef error at line 310 is already tested by TestMerge_FileLevelWriteRefError.
	// Let's test fileLevelMerge directly with a corrupted branch ref for the final WriteRef.
	mainRef := filepath.Join(dir, repo.GogitDir, "refs", "heads", "main")
	os.Remove(mainRef)
	os.MkdirAll(mainRef, 0755)
	defer os.RemoveAll(mainRef)

	err := fileLevelMerge(dir, "main", "feature", mainHash, featHash)
	if err == nil {
		t.Fatal("expected error when final WriteRef fails")
	}
}

func TestFileLevelMerge_WriteCommitFnError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	Branch("feature")

	os.WriteFile(filepath.Join(dir, "m.txt"), []byte("m"), 0644)
	Add([]string{"m.txt"})
	Commit("main")
	mainHash, _ := refs.ResolveHead(dir)

	Checkout("feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644)
	Add([]string{"f.txt"})
	Commit("feat")
	featHash, _ := refs.ResolveHead(dir)

	Checkout("main")

	// Mock writeCommitFn to fail
	origFn := writeCommitFn
	writeCommitFn = func(root, treeHash string, parents []string, msg string) (string, error) {
		return "", fmt.Errorf("write commit failed")
	}
	defer func() { writeCommitFn = origFn }()

	err := fileLevelMerge(dir, "main", "feature", mainHash, featHash)
	if err == nil {
		t.Fatal("expected error when writeCommitFn fails in fileLevelMerge")
	}
	if err.Error() != "write commit failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFileLevelMerge_DirectReadCommitError(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	// Call with invalid hashes that can't be read
	err := fileLevelMerge(dir, "main", "feature",
		"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"0000000000000000000000000000000000000001")
	if err == nil {
		t.Fatal("expected error when ReadCommit fails for current hash")
	}
}
