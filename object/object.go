package object

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gogit/repo"
)

// HashObject computes the SHA1 hash for an object with the given type and content.
func HashObject(objType string, content []byte) string {
	header := fmt.Sprintf("%s %d\x00", objType, len(content))
	h := sha1.New()
	h.Write([]byte(header))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// WriteObject writes a compressed object to the object store and returns its hash.
func WriteObject(root, objType string, content []byte) (string, error) {
	hash := HashObject(objType, content)
	objPath := filepath.Join(repo.ObjectsPath(root), hash[:2], hash[2:])

	if _, err := os.Stat(objPath); err == nil {
		return hash, nil // already exists
	}

	if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
		return "", err
	}

	header := fmt.Sprintf("%s %d\x00", objType, len(content))
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write([]byte(header))
	w.Write(content)
	w.Close()

	if err := os.WriteFile(objPath, buf.Bytes(), 0644); err != nil {
		return "", err
	}
	return hash, nil
}

// ReadObject reads and decompresses an object, returning its type and content.
func ReadObject(root, hash string) (string, []byte, error) {
	objPath := filepath.Join(repo.ObjectsPath(root), hash[:2], hash[2:])

	data, err := os.ReadFile(objPath)
	if err != nil {
		return "", nil, fmt.Errorf("object not found: %s", hash)
	}

	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	defer r.Close()

	raw, err := io.ReadAll(r)
	if err != nil {
		return "", nil, err
	}

	// Parse header: "type size\0content"
	nullIdx := bytes.IndexByte(raw, 0)
	if nullIdx < 0 {
		return "", nil, fmt.Errorf("invalid object format")
	}

	header := string(raw[:nullIdx])
	var objType string
	var size int
	if _, err := fmt.Sscanf(header, "%s %d", &objType, &size); err != nil {
		return "", nil, fmt.Errorf("invalid object header: %s", header)
	}

	content := raw[nullIdx+1:]
	return objType, content, nil
}
