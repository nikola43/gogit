package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gogit/repo"
)

func TestInit_Success(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, repo.GogitDir))
	if err != nil {
		t.Fatal("expected .gogit directory")
	}
	if !info.IsDir() {
		t.Fatal(".gogit should be a directory")
	}

	if _, err := os.Stat(filepath.Join(dir, repo.GogitDir, "objects")); err != nil {
		t.Fatal("expected objects directory")
	}
	if _, err := os.Stat(filepath.Join(dir, repo.GogitDir, "refs", "heads")); err != nil {
		t.Fatal("expected refs/heads directory")
	}

	data, err := os.ReadFile(filepath.Join(dir, repo.GogitDir, "HEAD"))
	if err != nil {
		t.Fatal("expected HEAD file")
	}
	if string(data) != "ref: refs/heads/main\n" {
		t.Errorf("unexpected HEAD content: %q", data)
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	Init()
	err := Init()
	if err == nil {
		t.Fatal("expected error for double init")
	}
}

func TestInit_MkdirError(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Create a file named .gogit — Stat returns err==nil so initAt returns "already exists"
	os.WriteFile(filepath.Join(dir, repo.GogitDir), []byte("blocker"), 0644)

	err := Init()
	if err == nil {
		t.Fatal("expected error when .gogit is a file")
	}
}

func TestInitAt_Success(t *testing.T) {
	dir := t.TempDir()
	if err := initAt(dir); err != nil {
		t.Fatalf("initAt failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, repo.GogitDir, "objects")); err != nil {
		t.Fatal("expected objects directory")
	}
	data, _ := os.ReadFile(filepath.Join(dir, repo.GogitDir, "HEAD"))
	if string(data) != "ref: refs/heads/main\n" {
		t.Errorf("unexpected HEAD: %q", data)
	}
}

func TestInitAt_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	initAt(dir)
	err := initAt(dir)
	if err == nil {
		t.Fatal("expected error for double init")
	}
}

func TestInitAt_MkdirAllError(t *testing.T) {
	dir := t.TempDir()
	// Make parent read-only so MkdirAll fails
	sub := filepath.Join(dir, "readonly")
	os.MkdirAll(sub, 0755)
	os.Chmod(sub, 0555)
	defer os.Chmod(sub, 0755)

	err := initAt(sub)
	if err == nil {
		t.Fatal("expected error when directory is read-only")
	}
}

func TestInitAt_WriteFileError(t *testing.T) {
	dir := t.TempDir()

	// Mock cmdWriteFile to fail
	origWF := cmdWriteFile
	cmdWriteFile = func(name string, data []byte, perm os.FileMode) error {
		return fmt.Errorf("write file failed")
	}
	defer func() { cmdWriteFile = origWF }()

	err := initAt(dir)
	if err == nil {
		t.Fatal("expected error when WriteFile fails")
	}
	if err.Error() != "write file failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInit_GetwdError(t *testing.T) {
	origFn := cmdGetwd
	cmdGetwd = func() (string, error) { return "", fmt.Errorf("getwd failed") }
	defer func() { cmdGetwd = origFn }()

	err := Init()
	if err == nil {
		t.Fatal("expected error when Getwd fails")
	}
	if err.Error() != "getwd failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

