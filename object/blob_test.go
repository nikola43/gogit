package object

import (
	"bytes"
	"testing"
)

func TestWriteBlob(t *testing.T) {
	root := setupObjectStore(t)
	hash, err := WriteBlob(root, []byte("blob content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %d", len(hash))
	}
}

func TestHashBlob(t *testing.T) {
	content := []byte("test blob")
	hash := HashBlob(content)
	expected := HashObject("blob", content)
	if hash != expected {
		t.Errorf("HashBlob should match HashObject for blob type")
	}
}

func TestReadBlob(t *testing.T) {
	root := setupObjectStore(t)
	content := []byte("read blob test")
	hash, _ := WriteBlob(root, content)

	data, err := ReadBlob(root, hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch: got %q, want %q", data, content)
	}
}

func TestReadBlob_NotFound(t *testing.T) {
	root := setupObjectStore(t)
	_, err := ReadBlob(root, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for missing blob")
	}
}
