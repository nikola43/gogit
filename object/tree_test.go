package object

import (
	"encoding/hex"
	"testing"

	"gogit/index"
)

func TestWriteTreeAndReadTree(t *testing.T) {
	root := setupObjectStore(t)
	blobHash, _ := WriteBlob(root, []byte("file content"))

	entries := []TreeEntry{
		{Mode: "100644", Name: "file.txt", Hash: blobHash},
	}

	treeHash, err := WriteTree(root, entries)
	if err != nil {
		t.Fatalf("WriteTree failed: %v", err)
	}

	readEntries, err := ReadTree(root, treeHash)
	if err != nil {
		t.Fatalf("ReadTree failed: %v", err)
	}

	if len(readEntries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(readEntries))
	}
	if readEntries[0].Name != "file.txt" {
		t.Errorf("expected name 'file.txt', got '%s'", readEntries[0].Name)
	}
	if readEntries[0].Hash != blobHash {
		t.Errorf("hash mismatch")
	}
	if readEntries[0].Mode != "100644" {
		t.Errorf("mode mismatch: got %s", readEntries[0].Mode)
	}
}

func TestWriteTree_MultipleEntries(t *testing.T) {
	root := setupObjectStore(t)
	h1, _ := WriteBlob(root, []byte("aaa"))
	h2, _ := WriteBlob(root, []byte("bbb"))

	entries := []TreeEntry{
		{Mode: "100644", Name: "b.txt", Hash: h2},
		{Mode: "100644", Name: "a.txt", Hash: h1},
	}

	treeHash, err := WriteTree(root, entries)
	if err != nil {
		t.Fatalf("WriteTree failed: %v", err)
	}

	readEntries, err := ReadTree(root, treeHash)
	if err != nil {
		t.Fatalf("ReadTree failed: %v", err)
	}

	if readEntries[0].Name != "a.txt" {
		t.Errorf("expected sorted order, first entry is %s", readEntries[0].Name)
	}
}

func TestWriteTree_DirectorySorting(t *testing.T) {
	root := setupObjectStore(t)
	blobHash, _ := WriteBlob(root, []byte("content"))

	subEntries := []TreeEntry{
		{Mode: "100644", Name: "file.txt", Hash: blobHash},
	}
	subTreeHash, _ := WriteTree(root, subEntries)

	entries := []TreeEntry{
		{Mode: "100644", Name: "zebra", Hash: blobHash},
		{Mode: "40000", Name: "dir", Hash: subTreeHash},
		{Mode: "100644", Name: "alpha", Hash: blobHash},
	}

	treeHash, err := WriteTree(root, entries)
	if err != nil {
		t.Fatalf("WriteTree failed: %v", err)
	}

	readEntries, err := ReadTree(root, treeHash)
	if err != nil {
		t.Fatalf("ReadTree failed: %v", err)
	}

	if len(readEntries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(readEntries))
	}
}

func TestWriteTree_InvalidHash(t *testing.T) {
	root := setupObjectStore(t)
	entries := []TreeEntry{
		{Mode: "100644", Name: "file.txt", Hash: "not-a-hex-string"},
	}

	_, err := WriteTree(root, entries)
	if err == nil {
		t.Fatal("expected error for invalid hash hex")
	}
}

func TestReadTree_NotFound(t *testing.T) {
	root := setupObjectStore(t)
	_, err := ReadTree(root, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for missing tree")
	}
}

func TestParseTree_Empty(t *testing.T) {
	entries, err := ParseTree(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseTree_InvalidNoNull(t *testing.T) {
	_, err := ParseTree([]byte("no null here"))
	if err == nil {
		t.Fatal("expected error for missing null byte")
	}
}

func TestParseTree_InvalidNoSpace(t *testing.T) {
	data := []byte("nospace\x00" + "12345678901234567890")
	_, err := ParseTree(data)
	if err == nil {
		t.Fatal("expected error for missing space in header")
	}
}

func TestParseTree_TooShort(t *testing.T) {
	data := []byte("100644 file.txt\x00short")
	_, err := ParseTree(data)
	if err == nil {
		t.Fatal("expected error for truncated entry")
	}
}

func TestBuildTreeFromIndex_Flat(t *testing.T) {
	root := setupObjectStore(t)
	h1, _ := WriteBlob(root, []byte("content1"))
	h2, _ := WriteBlob(root, []byte("content2"))

	idx := &index.Index{
		Entries: []index.Entry{
			{Path: "a.txt", Hash: h1, Mode: 0100644},
			{Path: "b.txt", Hash: h2, Mode: 0100644},
		},
	}

	treeHash, err := BuildTreeFromIndex(root, idx)
	if err != nil {
		t.Fatalf("BuildTreeFromIndex failed: %v", err)
	}

	flat, err := FlattenTree(root, treeHash, "")
	if err != nil {
		t.Fatalf("FlattenTree failed: %v", err)
	}

	if flat["a.txt"] != h1 {
		t.Error("a.txt hash mismatch")
	}
	if flat["b.txt"] != h2 {
		t.Error("b.txt hash mismatch")
	}
}

func TestBuildTreeFromIndex_Nested(t *testing.T) {
	root := setupObjectStore(t)
	h1, _ := WriteBlob(root, []byte("root file"))
	h2, _ := WriteBlob(root, []byte("nested file"))
	h3, _ := WriteBlob(root, []byte("deep file"))

	idx := &index.Index{
		Entries: []index.Entry{
			{Path: "root.txt", Hash: h1, Mode: 0100644},
			{Path: "dir/nested.txt", Hash: h2, Mode: 0100644},
			{Path: "dir/sub/deep.txt", Hash: h3, Mode: 0100644},
		},
	}

	treeHash, err := BuildTreeFromIndex(root, idx)
	if err != nil {
		t.Fatalf("BuildTreeFromIndex failed: %v", err)
	}

	flat, err := FlattenTree(root, treeHash, "")
	if err != nil {
		t.Fatalf("FlattenTree failed: %v", err)
	}

	if len(flat) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(flat))
	}
	if flat["root.txt"] != h1 {
		t.Error("root.txt hash mismatch")
	}
	if flat["dir/nested.txt"] != h2 {
		t.Error("dir/nested.txt hash mismatch")
	}
	if flat["dir/sub/deep.txt"] != h3 {
		t.Error("dir/sub/deep.txt hash mismatch")
	}
}

func TestBuildTreeFromIndex_BadBlobHash(t *testing.T) {
	root := setupObjectStore(t)
	// Use a valid hex hash that doesn't exist in the store
	// BuildTreeFromIndex doesn't actually read blobs, it just stores their hashes.
	// The error would come from WriteTree if the hash is not valid hex.
	idx := &index.Index{
		Entries: []index.Entry{
			{Path: "dir/file.txt", Hash: "not-valid-hex!!!!!!!!!!!!!!!!!!!!!!", Mode: 0100644},
		},
	}

	_, err := BuildTreeFromIndex(root, idx)
	if err == nil {
		t.Fatal("expected error for invalid blob hash in tree")
	}
}

func TestFlattenTree_WithPrefix(t *testing.T) {
	root := setupObjectStore(t)
	h1, _ := WriteBlob(root, []byte("content"))

	entries := []TreeEntry{
		{Mode: "100644", Name: "file.txt", Hash: h1},
	}
	treeHash, _ := WriteTree(root, entries)

	flat, err := FlattenTree(root, treeHash, "prefix")
	if err != nil {
		t.Fatalf("FlattenTree failed: %v", err)
	}
	if _, ok := flat["prefix/file.txt"]; !ok {
		t.Error("expected prefix/file.txt in result")
	}
}

func TestFlattenTree_NotFound(t *testing.T) {
	root := setupObjectStore(t)
	_, err := FlattenTree(root, "0000000000000000000000000000000000000000", "")
	if err == nil {
		t.Fatal("expected error for missing tree")
	}
}

func TestFlattenTree_BadSubtreeHash(t *testing.T) {
	root := setupObjectStore(t)
	// Create a tree with a subtree entry that points to a non-existent object
	fakeSubTreeHash := "1111111111111111111111111111111111111111"
	fakeHashBytes, _ := hex.DecodeString(fakeSubTreeHash)

	// Manually build tree content with a directory entry pointing to bad hash
	treeContent := []byte("40000 badsubdir\x00")
	treeContent = append(treeContent, fakeHashBytes...)

	treeHash, err := WriteObject(root, "tree", treeContent)
	if err != nil {
		t.Fatalf("WriteObject failed: %v", err)
	}

	_, err = FlattenTree(root, treeHash, "")
	if err == nil {
		t.Fatal("expected error for bad subtree hash")
	}
}
