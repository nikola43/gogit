package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gogit/repo"
)

func TestDiff_NoChanges(t *testing.T) {
	setupTestRepoWithCommit(t)
	if err := Diff(); err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
}

func TestDiff_ModifiedFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello\nworld\n"), 0644)

	if err := Diff(); err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
}

func TestDiff_DeletedFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	os.Remove(filepath.Join(dir, "test.txt"))

	if err := Diff(); err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
}

func TestDiff_MultipleChanges(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0644)
	Add([]string{"a.txt"})
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bbb"), 0644)
	Add([]string{"b.txt"})
	Commit("add files")

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa_modified"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bbb_modified"), 0644)

	if err := Diff(); err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
}

func TestDiff_LargeFile(t *testing.T) {
	dir := setupTestRepoWithCommit(t)

	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("line\n")
	}
	content := b.String()
	os.WriteFile(filepath.Join(dir, "big.txt"), []byte(content), 0644)
	Add([]string{"big.txt"})
	Commit("add big file")

	var b2 strings.Builder
	for i := 0; i < 10; i++ {
		b2.WriteString("line\n")
	}
	b2.WriteString("INSERTED\n")
	for i := 0; i < 10; i++ {
		b2.WriteString("line\n")
	}
	os.WriteFile(filepath.Join(dir, "big.txt"), []byte(b2.String()), 0644)

	if err := Diff(); err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
}

func TestDiff_NewFileInIndex(t *testing.T) {
	dir := setupTestRepo(t)
	t.Setenv("GOGIT_AUTHOR_NAME", "Test")
	t.Setenv("GOGIT_AUTHOR_EMAIL", "test@test.com")

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("original"), 0644)
	Add([]string{"file.txt"})

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("modified"), 0644)

	if err := Diff(); err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
}

func TestDiff_NoRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := Diff()
	if err == nil {
		t.Fatal("expected error when not in a repo")
	}
}

func TestDiff_EmptyIndex(t *testing.T) {
	setupTestRepo(t)
	if err := Diff(); err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
}

func TestDiff_CorruptIndex(t *testing.T) {
	dir := setupTestRepo(t)
	os.WriteFile(filepath.Join(dir, repo.GogitDir, "index"), []byte("corrupt data that is long enough"), 0644)

	err := Diff()
	if err == nil {
		t.Fatal("expected error for corrupt index")
	}
}

func TestDiff_BadBlobInIndex(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Corrupt the object store for the test.txt blob
	// First, modify the file so diff tries to read the old blob
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("changed"), 0644)

	// Delete the blob object to trigger the ReadBlob error path
	objDir := filepath.Join(dir, repo.GogitDir, "objects")
	filepath.Walk(objDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			os.Remove(path)
		}
		return nil
	})

	// Diff should not return error, it just skips files with bad blobs (continue)
	if err := Diff(); err != nil {
		t.Fatalf("Diff should handle bad blobs gracefully: %v", err)
	}
}

func TestComputeLCS(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"a", "c"}
	dp := computeLCS(a, b)
	if dp[3][2] != 2 {
		t.Errorf("expected LCS length 2, got %d", dp[3][2])
	}
}

func TestComputeLCS_Empty(t *testing.T) {
	dp := computeLCS(nil, nil)
	if dp[0][0] != 0 {
		t.Error("empty LCS should be 0")
	}
}

func TestComputeLCS_OneEmpty(t *testing.T) {
	a := []string{"a", "b"}
	dp := computeLCS(a, nil)
	if dp[2][0] != 0 {
		t.Error("LCS with empty should be 0")
	}
}

func TestComputeLCS_DpRightBranch(t *testing.T) {
	// Force the dp[i][j-1] > dp[i-1][j] path
	a := []string{"x", "y"}
	b := []string{"y", "x", "y"}
	dp := computeLCS(a, b)
	if dp[2][3] != 2 {
		t.Errorf("expected LCS length 2, got %d", dp[2][3])
	}
}

func TestBuildHunks_AllNew(t *testing.T) {
	old := []string{}
	new := []string{"a", "b"}
	dp := computeLCS(old, new)
	hunks := buildHunks(old, new, dp)
	if len(hunks) == 0 {
		t.Error("expected hunks for all-new content")
	}
}

func TestBuildHunks_AllRemoved(t *testing.T) {
	old := []string{"a", "b"}
	new := []string{}
	dp := computeLCS(old, new)
	hunks := buildHunks(old, new, dp)
	if len(hunks) == 0 {
		t.Error("expected hunks for all-removed content")
	}
}

func TestBuildHunks_NoChanges(t *testing.T) {
	lines := []string{"a", "b", "c"}
	dp := computeLCS(lines, lines)
	hunks := buildHunks(lines, lines, dp)
	if len(hunks) != 0 {
		t.Error("expected no hunks for identical content")
	}
}

func TestBuildHunks_MultipleSeparateChanges(t *testing.T) {
	var old, new []string
	for i := 0; i < 20; i++ {
		old = append(old, "same")
		new = append(new, "same")
	}
	old[0] = "old_first"
	new[0] = "new_first"
	old[19] = "old_last"
	new[19] = "new_last"

	dp := computeLCS(old, new)
	hunks := buildHunks(old, new, dp)
	if len(hunks) == 0 {
		t.Error("expected hunks for changes")
	}
}

func TestBuildHunks_CloseChanges(t *testing.T) {
	// Changes within 2*contextLines of each other to exercise intervening context path
	var old, new []string
	for i := 0; i < 15; i++ {
		old = append(old, "same")
		new = append(new, "same")
	}
	old[2] = "old_a"
	new[2] = "new_a"
	old[6] = "old_b"
	new[6] = "new_b"

	dp := computeLCS(old, new)
	hunks := buildHunks(old, new, dp)
	if len(hunks) == 0 {
		t.Error("expected hunks")
	}
}

func TestPrintUnifiedDiff_DeletedFile(t *testing.T) {
	printUnifiedDiff("test.txt", []string{"line1", "line2"}, nil)
}

func TestPrintUnifiedDiff_ModifiedFile(t *testing.T) {
	printUnifiedDiff("test.txt", []string{"old"}, []string{"new"})
}

func TestDiff_DeletedFileWithBadBlob(t *testing.T) {
	dir := setupTestRepoWithCommit(t)
	// Delete the working file
	os.Remove(filepath.Join(dir, "test.txt"))
	// Delete all objects to trigger ReadBlob error for the deleted file path
	objDir := filepath.Join(dir, repo.GogitDir, "objects")
	filepath.Walk(objDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			os.Remove(path)
		}
		return nil
	})

	// Diff should not error (ReadBlob error causes continue)
	if err := Diff(); err != nil {
		t.Fatalf("Diff should handle bad blob for deleted file: %v", err)
	}
}
