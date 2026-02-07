package index

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

	"gogit/repo"
)

const (
	indexMagic   = "GIDX"
	indexVersion = 1
)

// Entry represents a single index entry.
type Entry struct {
	Ctime uint32
	Mtime uint32
	Size  uint32
	Hash  string // 40-char hex SHA1
	Mode  uint32
	Path  string
}

// Index represents the staging area.
type Index struct {
	Entries []Entry
}

// ReadIndex reads the index file from disk.
func ReadIndex(root string) (*Index, error) {
	path := repo.IndexPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{}, nil
		}
		return nil, err
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("index file too short")
	}

	// Verify checksum
	storedSum := data[len(data)-20:]
	h := sha1.Sum(data[:len(data)-20])
	if !bytes.Equal(h[:], storedSum) {
		return nil, fmt.Errorf("index checksum mismatch")
	}

	r := bytes.NewReader(data)

	// Header
	magic := make([]byte, 4)
	r.Read(magic)
	if string(magic) != indexMagic {
		return nil, fmt.Errorf("invalid index magic: %s", magic)
	}

	var version, count uint32
	binary.Read(r, binary.BigEndian, &version)
	binary.Read(r, binary.BigEndian, &count)

	if version != indexVersion {
		return nil, fmt.Errorf("unsupported index version: %d", version)
	}

	idx := &Index{Entries: make([]Entry, 0, count)}
	for i := uint32(0); i < count; i++ {
		var e Entry
		binary.Read(r, binary.BigEndian, &e.Ctime)
		binary.Read(r, binary.BigEndian, &e.Mtime)
		binary.Read(r, binary.BigEndian, &e.Size)

		hashBytes := make([]byte, 20)
		r.Read(hashBytes)
		e.Hash = hex.EncodeToString(hashBytes)

		binary.Read(r, binary.BigEndian, &e.Mode)

		var pathLen uint16
		binary.Read(r, binary.BigEndian, &pathLen)
		pathBytes := make([]byte, pathLen)
		r.Read(pathBytes)
		e.Path = string(pathBytes)

		// Read padding to 8-byte boundary
		// Entry size so far: 4+4+4+20+4+2+pathLen = 38+pathLen
		entryLen := 38 + int(pathLen)
		padLen := (8 - (entryLen % 8)) % 8
		if padLen > 0 {
			pad := make([]byte, padLen)
			r.Read(pad)
		}

		idx.Entries = append(idx.Entries, e)
	}

	return idx, nil
}

// WriteIndex writes the index to disk.
func WriteIndex(root string, idx *Index) error {
	sort.Slice(idx.Entries, func(i, j int) bool {
		return idx.Entries[i].Path < idx.Entries[j].Path
	})

	var buf bytes.Buffer

	// Header
	buf.WriteString(indexMagic)
	binary.Write(&buf, binary.BigEndian, uint32(indexVersion))
	binary.Write(&buf, binary.BigEndian, uint32(len(idx.Entries)))

	for _, e := range idx.Entries {
		binary.Write(&buf, binary.BigEndian, e.Ctime)
		binary.Write(&buf, binary.BigEndian, e.Mtime)
		binary.Write(&buf, binary.BigEndian, e.Size)

		hashBytes, _ := hex.DecodeString(e.Hash)
		buf.Write(hashBytes)

		binary.Write(&buf, binary.BigEndian, e.Mode)
		binary.Write(&buf, binary.BigEndian, uint16(len(e.Path)))
		buf.WriteString(e.Path)

		// Pad to 8-byte boundary
		entryLen := 38 + len(e.Path)
		padLen := (8 - (entryLen % 8)) % 8
		for k := 0; k < padLen; k++ {
			buf.WriteByte(0)
		}
	}

	// Checksum
	h := sha1.Sum(buf.Bytes())
	buf.Write(h[:])

	return os.WriteFile(repo.IndexPath(root), buf.Bytes(), 0644)
}

// AddEntry adds or updates an entry in the index.
func (idx *Index) AddEntry(e Entry) {
	for i, existing := range idx.Entries {
		if existing.Path == e.Path {
			idx.Entries[i] = e
			return
		}
	}
	idx.Entries = append(idx.Entries, e)
}

// RemoveEntry removes an entry from the index by path.
func (idx *Index) RemoveEntry(path string) {
	for i, e := range idx.Entries {
		if e.Path == path {
			idx.Entries = append(idx.Entries[:i], idx.Entries[i+1:]...)
			return
		}
	}
}

// LookupEntry finds an entry by path.
func (idx *Index) LookupEntry(path string) *Entry {
	for i, e := range idx.Entries {
		if e.Path == path {
			return &idx.Entries[i]
		}
	}
	return nil
}
