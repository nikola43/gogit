package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// resolveSymlinks resolves symlinks in a path (needed on macOS where /var → /private/var).
func resolveSymlinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	return resolved
}

func TestFind_Success(t *testing.T) {
	dir := resolveSymlinks(t, t.TempDir())
	os.MkdirAll(filepath.Join(dir, GogitDir), 0755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	root, err := Find()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("expected %s, got %s", dir, root)
	}
}

func TestFind_FromSubdirectory(t *testing.T) {
	dir := resolveSymlinks(t, t.TempDir())
	os.MkdirAll(filepath.Join(dir, GogitDir), 0755)
	sub := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(sub, 0755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(sub)

	root, err := Find()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("expected %s, got %s", dir, root)
	}
}

func TestFind_NoRepo(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, err := Find()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGogitPath(t *testing.T) {
	got := GogitPath("/foo")
	want := filepath.Join("/foo", GogitDir)
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestObjectsPath(t *testing.T) {
	got := ObjectsPath("/foo")
	want := filepath.Join("/foo", GogitDir, "objects")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestRefsPath(t *testing.T) {
	got := RefsPath("/foo")
	want := filepath.Join("/foo", GogitDir, "refs")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestHeadPath(t *testing.T) {
	got := HeadPath("/foo")
	want := filepath.Join("/foo", GogitDir, "HEAD")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestIndexPath(t *testing.T) {
	got := IndexPath("/foo")
	want := filepath.Join("/foo", GogitDir, "index")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFindFrom_Success(t *testing.T) {
	dir := resolveSymlinks(t, t.TempDir())
	os.MkdirAll(filepath.Join(dir, GogitDir), 0755)

	root, err := FindFrom(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("expected %s, got %s", dir, root)
	}
}

func TestFindFrom_SubDir(t *testing.T) {
	dir := resolveSymlinks(t, t.TempDir())
	os.MkdirAll(filepath.Join(dir, GogitDir), 0755)
	sub := filepath.Join(dir, "a", "b")
	os.MkdirAll(sub, 0755)

	root, err := FindFrom(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("expected %s, got %s", dir, root)
	}
}

func TestFindFrom_NoRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := FindFrom(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFind_GetwdError(t *testing.T) {
	origFn := osGetwd
	osGetwd = func() (string, error) { return "", fmt.Errorf("getwd failed") }
	defer func() { osGetwd = origFn }()

	_, err := Find()
	if err == nil {
		t.Fatal("expected error when Getwd fails")
	}
	if err.Error() != "getwd failed" {
		t.Errorf("unexpected error: %v", err)
	}
}
