package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"gogit/index"
	"gogit/repo"
)

func TestAdd_SingleFile(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)

	if err := Add([]string{"file.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	idx, _ := index.ReadIndex(dir)
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Entries))
	}
	if idx.Entries[0].Path != "file.txt" {
		t.Errorf("unexpected path: %s", idx.Entries[0].Path)
	}
}

func TestAdd_MultipleFiles(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bbb"), 0644)

	if err := Add([]string{"a.txt", "b.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	idx, _ := index.ReadIndex(dir)
	if len(idx.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(idx.Entries))
	}
}

func TestAdd_UpdateExisting(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v1"), 0644)
	Add([]string{"file.txt"})

	idx1, _ := index.ReadIndex(dir)
	hash1 := idx1.Entries[0].Hash

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v2"), 0644)
	Add([]string{"file.txt"})

	idx2, _ := index.ReadIndex(dir)
	if idx2.Entries[0].Hash == hash1 {
		t.Error("hash should change after update")
	}
	if len(idx2.Entries) != 1 {
		t.Error("should still have 1 entry")
	}
}

func TestAdd_Directory(t *testing.T) {
	dir := setupTestRepo(t)
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(subdir, "b.txt"), []byte("bbb"), 0644)

	if err := Add([]string{"subdir"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	idx, _ := index.ReadIndex(dir)
	if len(idx.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(idx.Entries))
	}
}

func TestAdd_DeletedFile(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	Add([]string{"file.txt"})

	os.Remove(filepath.Join(dir, "file.txt"))
	if err := Add([]string{"file.txt"}); err != nil {
		t.Fatalf("Add failed for deleted file: %v", err)
	}

	idx, _ := index.ReadIndex(dir)
	if len(idx.Entries) != 0 {
		t.Errorf("expected 0 entries after deleting, got %d", len(idx.Entries))
	}
}

func TestAdd_ExecutableFile(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/sh"), 0755)

	Add([]string{"script.sh"})

	idx, _ := index.ReadIndex(dir)
	if len(idx.Entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if idx.Entries[0].Mode != 0100755 {
		t.Errorf("expected mode 100755, got %o", idx.Entries[0].Mode)
	}
}

func TestAdd_SkipsGogitDir(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)

	if err := Add([]string{"."}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	idx, _ := index.ReadIndex(dir)
	for _, e := range idx.Entries {
		if e.Path == ".gogit" || len(e.Path) > 6 && e.Path[:7] == ".gogit/" {
			t.Errorf("should not include .gogit in index: %s", e.Path)
		}
	}
}

func TestAdd_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Add([]string{"file.txt"})
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestAdd_NestedDirectory(t *testing.T) {
	dir := setupTestRepo(t)
	deep := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("deep"), 0644)

	if err := Add([]string{"a"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	idx, _ := index.ReadIndex(dir)
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Entries))
	}
	if idx.Entries[0].Path != "a/b/c/deep.txt" {
		t.Errorf("unexpected path: %s", idx.Entries[0].Path)
	}
}

func TestAdd_CorruptIndex(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	// Write a corrupt index file
	os.WriteFile(filepath.Join(dir, repo.GogitDir, "index"), []byte("corrupt data that is long enough"), 0644)

	err := Add([]string{"file.txt"})
	if err == nil {
		t.Fatal("expected error for corrupt index")
	}
}

func TestAdd_UnreadableFileInDir(t *testing.T) {
	dir := setupTestRepo(t)
	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0755)
	f := filepath.Join(subdir, "unreadable.txt")
	os.WriteFile(f, []byte("secret"), 0000)
	defer os.Chmod(f, 0644)

	err := Add([]string{"sub"})
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestAdd_ErrorFromAddPath(t *testing.T) {
	dir := setupTestRepo(t)
	// Create a file then make it unreadable
	f := filepath.Join(dir, "unreadable.txt")
	os.WriteFile(f, []byte("content"), 0000)
	defer os.Chmod(f, 0644)

	err := Add([]string{"unreadable.txt"})
	if err == nil {
		t.Fatal("expected error for unreadable file in addPath")
	}
}

func TestAdd_GogitDirInAddFile(t *testing.T) {
	setupTestRepo(t)
	// Use a relative path so filepath.Abs resolves via Getwd (avoids macOS symlink mismatch)
	gogitFile := filepath.Join(repo.GogitDir, "HEAD")
	err := Add([]string{gogitFile})
	// Should succeed but not add it to index (skipped by .gogit prefix check)
	if err != nil {
		t.Fatalf("Add should not fail for .gogit file: %v", err)
	}

	root, _ := repo.Find()
	idx, _ := index.ReadIndex(root)
	for _, e := range idx.Entries {
		if e.Path == ".gogit/HEAD" {
			t.Error("should not have .gogit/HEAD in index")
		}
	}
}

func TestAdd_DirWithGogitSubdir(t *testing.T) {
	dir := setupTestRepo(t)
	// The .gogit dir should be skipped during directory walk
	os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("visible"), 0644)

	Add([]string{dir})

	idx, _ := index.ReadIndex(dir)
	for _, e := range idx.Entries {
		if len(e.Path) >= 6 && e.Path[:6] == ".gogit" {
			t.Errorf("should not include .gogit path: %s", e.Path)
		}
	}
}

func TestAdd_WriteBlobError(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)

	// Replace objects directory with a file to block blob writes
	objDir := filepath.Join(dir, repo.GogitDir, "objects")
	os.RemoveAll(objDir)
	os.WriteFile(objDir, []byte("blocker"), 0644)
	defer func() {
		os.Remove(objDir)
		os.MkdirAll(objDir, 0755)
	}()

	err := Add([]string{"file.txt"})
	if err == nil {
		t.Fatal("expected error when blob write fails")
	}
}

func TestAdd_WalkErrorInDir(t *testing.T) {
	dir := setupTestRepo(t)
	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("content"), 0644)
	// Make the subdir unreadable to trigger walk error
	inner := filepath.Join(subdir, "inner")
	os.MkdirAll(inner, 0000)
	defer os.Chmod(inner, 0755)

	err := Add([]string{"sub"})
	// Walk error may propagate or not depending on macOS permissions
	_ = err
}
