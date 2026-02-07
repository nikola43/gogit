package object

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"testing"

	"gogit/repo"
)

func setupObjectStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, repo.GogitDir, "objects"), 0755)
	return dir
}

func TestHashObject_Deterministic(t *testing.T) {
	h1 := HashObject("blob", []byte("hello"))
	h2 := HashObject("blob", []byte("hello"))
	if h1 != h2 {
		t.Error("hashes should be deterministic")
	}
	if len(h1) != 40 {
		t.Errorf("hash should be 40 chars, got %d", len(h1))
	}
}

func TestHashObject_DifferentContent(t *testing.T) {
	h1 := HashObject("blob", []byte("hello"))
	h2 := HashObject("blob", []byte("world"))
	if h1 == h2 {
		t.Error("different content should produce different hashes")
	}
}

func TestHashObject_DifferentType(t *testing.T) {
	h1 := HashObject("blob", []byte("hello"))
	h2 := HashObject("tree", []byte("hello"))
	if h1 == h2 {
		t.Error("different types should produce different hashes")
	}
}

func TestWriteObject_Success(t *testing.T) {
	root := setupObjectStore(t)
	hash, err := WriteObject(root, "blob", []byte("test content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %d", len(hash))
	}

	// Verify file exists
	objPath := filepath.Join(repo.ObjectsPath(root), hash[:2], hash[2:])
	if _, err := os.Stat(objPath); err != nil {
		t.Errorf("object file should exist: %v", err)
	}
}

func TestWriteObject_Deduplication(t *testing.T) {
	root := setupObjectStore(t)
	content := []byte("dedup test")

	h1, err := WriteObject(root, "blob", content)
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	h2, err := WriteObject(root, "blob", content)
	if err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	if h1 != h2 {
		t.Error("writing same content twice should return same hash")
	}
}

func TestWriteObject_MkdirError(t *testing.T) {
	root := setupObjectStore(t)
	// Make objects dir read-only to trigger MkdirAll error
	objDir := repo.ObjectsPath(root)
	os.Chmod(objDir, 0444)
	defer os.Chmod(objDir, 0755)

	_, err := WriteObject(root, "blob", []byte("will fail"))
	if err == nil {
		t.Fatal("expected error when objects dir is read-only")
	}
}

func TestWriteObject_WriteFileError(t *testing.T) {
	root := setupObjectStore(t)
	content := []byte("write fail test")
	hash := HashObject("blob", content)

	// Create the subdirectory but make it read-only
	subDir := filepath.Join(repo.ObjectsPath(root), hash[:2])
	os.MkdirAll(subDir, 0755)
	os.Chmod(subDir, 0555)
	defer os.Chmod(subDir, 0755)

	_, err := WriteObject(root, "blob", content)
	if err == nil {
		t.Fatal("expected error when sub-dir is read-only")
	}
}

func TestReadObject_Success(t *testing.T) {
	root := setupObjectStore(t)
	content := []byte("read test")
	hash, _ := WriteObject(root, "blob", content)

	objType, data, err := ReadObject(root, hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if objType != "blob" {
		t.Errorf("expected type 'blob', got '%s'", objType)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch")
	}
}

func TestReadObject_NotFound(t *testing.T) {
	root := setupObjectStore(t)
	_, _, err := ReadObject(root, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for missing object")
	}
}

func TestReadObject_InvalidZlib(t *testing.T) {
	root := setupObjectStore(t)
	hash := "aabbccddee0000000000000000000000000000ff"
	objDir := filepath.Join(repo.ObjectsPath(root), hash[:2])
	os.MkdirAll(objDir, 0755)
	os.WriteFile(filepath.Join(objDir, hash[2:]), []byte("not zlib data"), 0644)

	_, _, err := ReadObject(root, hash)
	if err == nil {
		t.Fatal("expected error for invalid zlib")
	}
}

func TestReadObject_TruncatedZlib(t *testing.T) {
	root := setupObjectStore(t)
	hash := "aabbccddee00000000000000000000000000aacc"
	objDir := filepath.Join(repo.ObjectsPath(root), hash[:2])
	os.MkdirAll(objDir, 0755)

	// Write zlib data that starts valid but is truncated
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write([]byte("blob 100\x00some content that goes on for a while"))
	// Don't close the writer properly to create truncated data
	w.Flush()
	data := buf.Bytes()
	// Truncate to make it incomplete
	os.WriteFile(filepath.Join(objDir, hash[2:]), data[:len(data)/2], 0644)

	_, _, err := ReadObject(root, hash)
	if err == nil {
		t.Fatal("expected error for truncated zlib")
	}
}

func TestReadObject_NoNullByte(t *testing.T) {
	root := setupObjectStore(t)
	hash := "aabbccddee0000000000000000000000000000ab"
	objDir := filepath.Join(repo.ObjectsPath(root), hash[:2])
	os.MkdirAll(objDir, 0755)

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write([]byte("no null byte here"))
	w.Close()
	os.WriteFile(filepath.Join(objDir, hash[2:]), buf.Bytes(), 0644)

	_, _, err := ReadObject(root, hash)
	if err == nil {
		t.Fatal("expected error for missing null byte")
	}
}

func TestReadObject_InvalidHeader(t *testing.T) {
	root := setupObjectStore(t)
	hash := "aabbccddee0000000000000000000000000000cd"
	objDir := filepath.Join(repo.ObjectsPath(root), hash[:2])
	os.MkdirAll(objDir, 0755)

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write([]byte("badheader\x00content"))
	w.Close()
	os.WriteFile(filepath.Join(objDir, hash[2:]), buf.Bytes(), 0644)

	_, _, err := ReadObject(root, hash)
	if err == nil {
		t.Fatal("expected error for invalid header")
	}
}
