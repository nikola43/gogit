package index

import (
	"crypto/sha1"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"gogit/repo"
)

func setupIndexDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, repo.GogitDir), 0755)
	return dir
}

func TestReadIndex_NoFile(t *testing.T) {
	root := setupIndexDir(t)
	idx, err := ReadIndex(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected empty index, got %d entries", len(idx.Entries))
	}
}

func TestWriteAndReadIndex_Roundtrip(t *testing.T) {
	root := setupIndexDir(t)

	idx := &Index{
		Entries: []Entry{
			{Ctime: 100, Mtime: 200, Size: 50, Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "file.txt"},
		},
	}

	if err := WriteIndex(root, idx); err != nil {
		t.Fatalf("WriteIndex failed: %v", err)
	}

	idx2, err := ReadIndex(root)
	if err != nil {
		t.Fatalf("ReadIndex failed: %v", err)
	}

	if len(idx2.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx2.Entries))
	}

	e := idx2.Entries[0]
	if e.Ctime != 100 {
		t.Errorf("Ctime mismatch: %d", e.Ctime)
	}
	if e.Mtime != 200 {
		t.Errorf("Mtime mismatch: %d", e.Mtime)
	}
	if e.Size != 50 {
		t.Errorf("Size mismatch: %d", e.Size)
	}
	if e.Hash != "aabbccddee00112233445566778899aabbccddee" {
		t.Errorf("Hash mismatch: %s", e.Hash)
	}
	if e.Mode != 0100644 {
		t.Errorf("Mode mismatch: %o", e.Mode)
	}
	if e.Path != "file.txt" {
		t.Errorf("Path mismatch: %s", e.Path)
	}
}

func TestWriteAndReadIndex_MultipleEntries(t *testing.T) {
	root := setupIndexDir(t)

	idx := &Index{
		Entries: []Entry{
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "z.txt"},
			{Hash: "1122334455667788990011223344556677889900", Mode: 0100644, Path: "a.txt"},
			{Hash: "ffeeddccbbaa99887766554433221100ffeeddcc", Mode: 0100755, Path: "mid.txt"},
		},
	}

	WriteIndex(root, idx)
	idx2, _ := ReadIndex(root)

	// Should be sorted by path
	if idx2.Entries[0].Path != "a.txt" {
		t.Errorf("expected first entry to be a.txt, got %s", idx2.Entries[0].Path)
	}
	if idx2.Entries[1].Path != "mid.txt" {
		t.Errorf("expected second entry to be mid.txt, got %s", idx2.Entries[1].Path)
	}
	if idx2.Entries[2].Path != "z.txt" {
		t.Errorf("expected third entry to be z.txt, got %s", idx2.Entries[2].Path)
	}
}

func TestWriteAndReadIndex_VariousPathLengths(t *testing.T) {
	root := setupIndexDir(t)

	// Test various path lengths to exercise padding logic
	idx := &Index{
		Entries: []Entry{
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "a"},           // 1 char
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "ab"},          // 2 chars
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "abc"},         // 3 chars
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "abcd"},        // 4 chars
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "abcde"},       // 5 chars
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "abcdef"},      // 6 chars (38+6=44, divisible by 4, padLen=(8-44%8)%8 = (8-4)%8=4)
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "abcdefgh"},    // 8 chars (38+8=46, padLen=(8-46%8)%8=(8-6)%8=2)
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "abcdefghij"},  // 10 chars (38+10=48, 48%8=0, padLen=0)
		},
	}

	WriteIndex(root, idx)
	idx2, err := ReadIndex(root)
	if err != nil {
		t.Fatalf("ReadIndex failed: %v", err)
	}

	if len(idx2.Entries) != 8 {
		t.Fatalf("expected 8 entries, got %d", len(idx2.Entries))
	}

	// Verify all paths survived roundtrip
	for i, e := range idx2.Entries {
		if e.Hash != "aabbccddee00112233445566778899aabbccddee" {
			t.Errorf("entry %d hash mismatch", i)
		}
	}
}

func TestReadIndex_TooShort(t *testing.T) {
	root := setupIndexDir(t)
	os.WriteFile(repo.IndexPath(root), []byte("short"), 0644)

	_, err := ReadIndex(root)
	if err == nil {
		t.Fatal("expected error for too-short index")
	}
}

func TestReadIndex_BadChecksum(t *testing.T) {
	root := setupIndexDir(t)

	// Write a valid-length file with bad checksum
	data := make([]byte, 32) // 12 header + 20 checksum = 32 minimum
	copy(data, indexMagic)
	binary.BigEndian.PutUint32(data[4:], indexVersion)
	binary.BigEndian.PutUint32(data[8:], 0) // 0 entries
	// Bad checksum (all zeros)
	os.WriteFile(repo.IndexPath(root), data, 0644)

	_, err := ReadIndex(root)
	if err == nil {
		t.Fatal("expected error for bad checksum")
	}
}

func TestReadIndex_BadMagic(t *testing.T) {
	root := setupIndexDir(t)

	data := make([]byte, 32)
	copy(data, "BAAD") // wrong magic
	binary.BigEndian.PutUint32(data[4:], indexVersion)
	binary.BigEndian.PutUint32(data[8:], 0)
	// Compute valid checksum
	h := sha1.Sum(data[:12])
	copy(data[12:], h[:])
	os.WriteFile(repo.IndexPath(root), data, 0644)

	_, err := ReadIndex(root)
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestReadIndex_BadVersion(t *testing.T) {
	root := setupIndexDir(t)

	data := make([]byte, 32)
	copy(data, indexMagic)
	binary.BigEndian.PutUint32(data[4:], 99) // wrong version
	binary.BigEndian.PutUint32(data[8:], 0)
	h := sha1.Sum(data[:12])
	copy(data[12:], h[:])
	os.WriteFile(repo.IndexPath(root), data, 0644)

	_, err := ReadIndex(root)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
}

func TestAddEntry_New(t *testing.T) {
	idx := &Index{}
	idx.AddEntry(Entry{Path: "file.txt", Hash: "abc"})
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Entries))
	}
}

func TestAddEntry_Update(t *testing.T) {
	idx := &Index{
		Entries: []Entry{{Path: "file.txt", Hash: "old"}},
	}
	idx.AddEntry(Entry{Path: "file.txt", Hash: "new"})
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Entries))
	}
	if idx.Entries[0].Hash != "new" {
		t.Errorf("hash should be updated to 'new', got '%s'", idx.Entries[0].Hash)
	}
}

func TestRemoveEntry_Exists(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Path: "a.txt"},
			{Path: "b.txt"},
		},
	}
	idx.RemoveEntry("a.txt")
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Entries))
	}
	if idx.Entries[0].Path != "b.txt" {
		t.Error("wrong entry removed")
	}
}

func TestRemoveEntry_NotExists(t *testing.T) {
	idx := &Index{
		Entries: []Entry{{Path: "a.txt"}},
	}
	idx.RemoveEntry("nonexistent.txt")
	if len(idx.Entries) != 1 {
		t.Errorf("should not remove anything, got %d entries", len(idx.Entries))
	}
}

func TestLookupEntry_Found(t *testing.T) {
	idx := &Index{
		Entries: []Entry{{Path: "file.txt", Hash: "abc"}},
	}
	e := idx.LookupEntry("file.txt")
	if e == nil {
		t.Fatal("expected to find entry")
	}
	if e.Hash != "abc" {
		t.Errorf("wrong entry returned")
	}
}

func TestLookupEntry_NotFound(t *testing.T) {
	idx := &Index{}
	e := idx.LookupEntry("nonexistent.txt")
	if e != nil {
		t.Error("expected nil for missing entry")
	}
}

func TestReadIndex_PermissionError(t *testing.T) {
	root := setupIndexDir(t)
	// Write a valid index file, then make it unreadable
	idx := &Index{
		Entries: []Entry{
			{Hash: "aabbccddee00112233445566778899aabbccddee", Mode: 0100644, Path: "f.txt"},
		},
	}
	WriteIndex(root, idx)
	os.Chmod(repo.IndexPath(root), 0000)
	defer os.Chmod(repo.IndexPath(root), 0644)

	_, err := ReadIndex(root)
	if err == nil {
		t.Fatal("expected error for unreadable index file")
	}
}
