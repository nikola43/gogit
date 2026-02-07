package object

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"
)

func TestWriteCommitAndReadCommit(t *testing.T) {
	root := setupObjectStore(t)
	blobHash, _ := WriteBlob(root, []byte("file"))
	entries := []TreeEntry{{Mode: "100644", Name: "f.txt", Hash: blobHash}}
	treeHash, _ := WriteTree(root, entries)

	t.Setenv("GOGIT_AUTHOR_NAME", "Test User")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@example.com")

	hash, err := WriteCommit(root, treeHash, nil, "initial commit")
	if err != nil {
		t.Fatalf("WriteCommit failed: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %d", len(hash))
	}

	commit, err := ReadCommit(root, hash)
	if err != nil {
		t.Fatalf("ReadCommit failed: %v", err)
	}
	if commit.TreeHash != treeHash {
		t.Errorf("tree hash mismatch")
	}
	if commit.Message != "initial commit" {
		t.Errorf("message mismatch: got %q", commit.Message)
	}
	if len(commit.Parents) != 0 {
		t.Errorf("expected no parents, got %d", len(commit.Parents))
	}
	if !strings.Contains(commit.Author, "Test User") {
		t.Errorf("author should contain name: %s", commit.Author)
	}
	if !strings.Contains(commit.Author, "test@example.com") {
		t.Errorf("author should contain email: %s", commit.Author)
	}
	if !strings.Contains(commit.Committer, "Test User") {
		t.Errorf("committer should contain name: %s", commit.Committer)
	}
}

func TestWriteCommit_WithParent(t *testing.T) {
	root := setupObjectStore(t)
	blobHash, _ := WriteBlob(root, []byte("file"))
	entries := []TreeEntry{{Mode: "100644", Name: "f.txt", Hash: blobHash}}
	treeHash, _ := WriteTree(root, entries)

	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	parent, _ := WriteCommit(root, treeHash, nil, "first")
	hash, err := WriteCommit(root, treeHash, []string{parent}, "second")
	if err != nil {
		t.Fatalf("WriteCommit failed: %v", err)
	}

	commit, _ := ReadCommit(root, hash)
	if len(commit.Parents) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(commit.Parents))
	}
	if commit.Parents[0] != parent {
		t.Errorf("parent mismatch")
	}
}

func TestWriteCommit_MultipleParents(t *testing.T) {
	root := setupObjectStore(t)
	blobHash, _ := WriteBlob(root, []byte("file"))
	entries := []TreeEntry{{Mode: "100644", Name: "f.txt", Hash: blobHash}}
	treeHash, _ := WriteTree(root, entries)

	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "t@t.com")

	p1, _ := WriteCommit(root, treeHash, nil, "parent1")
	p2, _ := WriteCommit(root, treeHash, nil, "parent2")
	hash, _ := WriteCommit(root, treeHash, []string{p1, p2}, "merge")

	commit, _ := ReadCommit(root, hash)
	if len(commit.Parents) != 2 {
		t.Fatalf("expected 2 parents, got %d", len(commit.Parents))
	}
}

func TestReadCommit_NotFound(t *testing.T) {
	root := setupObjectStore(t)
	_, err := ReadCommit(root, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for missing commit")
	}
}

func TestParseCommit_HeadersOnly(t *testing.T) {
	data := []byte("tree abc123\nauthor Test <t@t>\ncommitter Test <t@t>")
	c, err := ParseCommit(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.TreeHash != "abc123" {
		t.Errorf("expected tree 'abc123', got '%s'", c.TreeHash)
	}
	if c.Message != "" {
		t.Errorf("expected empty message, got %q", c.Message)
	}
}

func TestFormatAuthor_WithEnvVars(t *testing.T) {
	t.Setenv("GOGIT_AUTHOR_NAME", "Env User")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "env@test.com")

	author := formatAuthor()
	if author != "Env User <env@test.com>" {
		t.Errorf("unexpected author: %s", author)
	}
}

func TestFormatAuthor_NoEnvVars(t *testing.T) {
	os.Unsetenv("GOGIT_AUTHOR_NAME")
	os.Unsetenv("GOGIT_AUTHOR_EMAIL")

	author := formatAuthor()
	// Should contain username@localhost when no env vars
	if !strings.Contains(author, "<") || !strings.Contains(author, ">") {
		t.Errorf("author should have email brackets: %s", author)
	}
	if !strings.Contains(author, "@localhost") {
		t.Errorf("author should fallback to @localhost: %s", author)
	}
}

func TestFormatAuthor_NameOnlyNoEmail(t *testing.T) {
	t.Setenv("GOGIT_AUTHOR_NAME", "JustName")
	os.Unsetenv("GOGIT_AUTHOR_EMAIL")

	author := formatAuthor()
	if author != "JustName <JustName@localhost>" {
		t.Errorf("unexpected author: %s", author)
	}
}

func TestFormatAuthor_UserLookupError(t *testing.T) {
	os.Unsetenv("GOGIT_AUTHOR_NAME")
	os.Unsetenv("GOGIT_AUTHOR_EMAIL")

	orig := userLookup
	userLookup = func() (*user.User, error) {
		return nil, fmt.Errorf("simulated user lookup failure")
	}
	defer func() { userLookup = orig }()

	author := formatAuthor()
	if !strings.Contains(author, "Unknown") {
		t.Errorf("expected 'Unknown' fallback, got %s", author)
	}
	if author != "Unknown <Unknown@localhost>" {
		t.Errorf("unexpected author: %s", author)
	}
}

func TestFormatTimestamp(t *testing.T) {
	ts := formatTimestamp()
	parts := strings.Fields(ts)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts in timestamp, got %d: %s", len(parts), ts)
	}
	offset := parts[1]
	if offset[0] != '+' && offset[0] != '-' {
		t.Errorf("timezone should start with + or -, got %s", offset)
	}
	if len(offset) != 5 {
		t.Errorf("timezone should be 5 chars (±HHMM), got %s", offset)
	}
}

func TestFormatTimestamp_NegativeOffset(t *testing.T) {
	// Use a timezone west of UTC to exercise negative offset branch
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("cannot load timezone")
	}
	old := time.Local
	time.Local = loc
	defer func() { time.Local = old }()

	ts := formatTimestamp()
	parts := strings.Fields(ts)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d: %s", len(parts), ts)
	}
	if parts[1][0] != '-' {
		t.Errorf("expected negative offset for US/Eastern, got %s", parts[1])
	}
}

func TestFormatTimestamp_PositiveOffset(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skip("cannot load timezone")
	}
	old := time.Local
	time.Local = loc
	defer func() { time.Local = old }()

	ts := formatTimestamp()
	parts := strings.Fields(ts)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d: %s", len(parts), ts)
	}
	if parts[1][0] != '+' {
		t.Errorf("expected positive offset for Asia/Tokyo, got %s", parts[1])
	}
}
