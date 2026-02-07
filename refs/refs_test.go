package refs

import (
	"os"
	"path/filepath"
	"testing"

	"gogit/repo"
)

func setupRefsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, repo.GogitDir, "refs", "heads"), 0755)
	return dir
}

func TestReadHead(t *testing.T) {
	root := setupRefsDir(t)
	os.WriteFile(repo.HeadPath(root), []byte("ref: refs/heads/main\n"), 0644)

	head, err := ReadHead(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if head != "ref: refs/heads/main" {
		t.Errorf("unexpected head: %s", head)
	}
}

func TestReadHead_NoFile(t *testing.T) {
	root := setupRefsDir(t)
	_, err := ReadHead(root)
	if err == nil {
		t.Fatal("expected error when HEAD doesn't exist")
	}
}

func TestResolveHead_SymbolicRef(t *testing.T) {
	root := setupRefsDir(t)
	os.WriteFile(repo.HeadPath(root), []byte("ref: refs/heads/main\n"), 0644)
	WriteRef(root, "refs/heads/main", "abc123")

	hash, err := ResolveHead(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abc123" {
		t.Errorf("expected 'abc123', got '%s'", hash)
	}
}

func TestResolveHead_SymbolicRefNoCommits(t *testing.T) {
	root := setupRefsDir(t)
	os.WriteFile(repo.HeadPath(root), []byte("ref: refs/heads/main\n"), 0644)

	hash, err := ResolveHead(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash for no commits, got '%s'", hash)
	}
}

func TestResolveHead_DirectHash(t *testing.T) {
	root := setupRefsDir(t)
	os.WriteFile(repo.HeadPath(root), []byte("abc123def456\n"), 0644)

	hash, err := ResolveHead(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abc123def456" {
		t.Errorf("expected 'abc123def456', got '%s'", hash)
	}
}

func TestResolveHead_NoHead(t *testing.T) {
	root := setupRefsDir(t)
	_, err := ResolveHead(root)
	if err == nil {
		t.Fatal("expected error when HEAD doesn't exist")
	}
}

func TestCurrentBranch_OnBranch(t *testing.T) {
	root := setupRefsDir(t)
	os.WriteFile(repo.HeadPath(root), []byte("ref: refs/heads/main\n"), 0644)

	branch, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main', got '%s'", branch)
	}
}

func TestCurrentBranch_Detached(t *testing.T) {
	root := setupRefsDir(t)
	os.WriteFile(repo.HeadPath(root), []byte("abc123\n"), 0644)

	branch, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty string for detached HEAD, got '%s'", branch)
	}
}

func TestCurrentBranch_NoHead(t *testing.T) {
	root := setupRefsDir(t)
	_, err := CurrentBranch(root)
	if err == nil {
		t.Fatal("expected error when HEAD doesn't exist")
	}
}

func TestReadRef_Exists(t *testing.T) {
	root := setupRefsDir(t)
	refPath := "refs/heads/main"
	WriteRef(root, refPath, "hash123")

	hash, err := ReadRef(root, refPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "hash123" {
		t.Errorf("expected 'hash123', got '%s'", hash)
	}
}

func TestReadRef_NotExists(t *testing.T) {
	root := setupRefsDir(t)
	hash, err := ReadRef(root, "refs/heads/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty string, got '%s'", hash)
	}
}

func TestReadRef_PermissionError(t *testing.T) {
	root := setupRefsDir(t)
	WriteRef(root, "refs/heads/secret", "hash")
	// Make the file unreadable
	refFile := filepath.Join(repo.GogitPath(root), "refs", "heads", "secret")
	os.Chmod(refFile, 0000)
	defer os.Chmod(refFile, 0644)

	_, err := ReadRef(root, "refs/heads/secret")
	if err == nil {
		t.Fatal("expected error for unreadable ref file")
	}
}

func TestWriteRef(t *testing.T) {
	root := setupRefsDir(t)
	err := WriteRef(root, "refs/heads/test", "somehash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(repo.GogitPath(root), "refs", "heads", "test"))
	if string(data) != "somehash\n" {
		t.Errorf("unexpected ref content: %q", data)
	}
}

func TestWriteRef_CreatesDirectories(t *testing.T) {
	root := setupRefsDir(t)
	err := WriteRef(root, "refs/tags/v1.0", "taghash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash, _ := ReadRef(root, "refs/tags/v1.0")
	if hash != "taghash" {
		t.Errorf("expected 'taghash', got '%s'", hash)
	}
}

func TestWriteRef_MkdirError(t *testing.T) {
	root := setupRefsDir(t)
	// Create a file at a path where a directory needs to be, so MkdirAll fails
	blocker := filepath.Join(repo.GogitPath(root), "refs", "blocker")
	os.WriteFile(blocker, []byte("file"), 0644)

	err := WriteRef(root, "refs/blocker/branch", "hash")
	if err == nil {
		t.Fatal("expected error when cannot create directory (file in the way)")
	}
}

func TestUpdateHead(t *testing.T) {
	root := setupRefsDir(t)
	err := UpdateHead(root, "ref: refs/heads/feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	head, _ := ReadHead(root)
	if head != "ref: refs/heads/feature" {
		t.Errorf("unexpected head: %s", head)
	}
}

func TestListBranches_WithBranches(t *testing.T) {
	root := setupRefsDir(t)
	WriteRef(root, "refs/heads/main", "h1")
	WriteRef(root, "refs/heads/feature", "h2")

	branches, err := ListBranches(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}
}

func TestListBranches_NoDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, repo.GogitDir), 0755)

	branches, err := ListBranches(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branches != nil {
		t.Errorf("expected nil branches, got %v", branches)
	}
}

func TestListBranches_SkipsDirectories(t *testing.T) {
	root := setupRefsDir(t)
	WriteRef(root, "refs/heads/main", "h1")
	os.MkdirAll(filepath.Join(repo.RefsPath(root), "heads", "subdir"), 0755)

	branches, err := ListBranches(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}
	if branches[0] != "main" {
		t.Errorf("expected 'main', got '%s'", branches[0])
	}
}

func TestListBranches_PermissionError(t *testing.T) {
	root := setupRefsDir(t)
	headsDir := filepath.Join(repo.RefsPath(root), "heads")
	// Make a file in heads that causes ReadDir to succeed but test that non-ENOENT errors propagate
	// Actually, make the heads dir unreadable
	os.Chmod(headsDir, 0000)
	defer os.Chmod(headsDir, 0755)

	_, err := ListBranches(root)
	if err == nil {
		t.Fatal("expected error for unreadable heads dir")
	}
}

func TestBranchRef(t *testing.T) {
	ref := BranchRef("feature")
	if ref != "refs/heads/feature" {
		t.Errorf("unexpected ref: %s", ref)
	}
}
