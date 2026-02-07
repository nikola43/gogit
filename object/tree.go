package object

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"gogit/index"
)

// TreeEntry represents a single entry in a tree object.
type TreeEntry struct {
	Mode string
	Name string
	Hash string // 40-char hex
}

// WriteTree writes a tree object and returns its hash.
func WriteTree(root string, entries []TreeEntry) (string, error) {
	sort.Slice(entries, func(i, j int) bool {
		// Directories sort with trailing slash in git
		nameI := entries[i].Name
		nameJ := entries[j].Name
		if entries[i].Mode == "40000" {
			nameI += "/"
		}
		if entries[j].Mode == "40000" {
			nameJ += "/"
		}
		return nameI < nameJ
	})

	var buf bytes.Buffer
	for _, e := range entries {
		fmt.Fprintf(&buf, "%s %s\x00", e.Mode, e.Name)
		hashBytes, err := hex.DecodeString(e.Hash)
		if err != nil {
			return "", err
		}
		buf.Write(hashBytes)
	}

	return WriteObject(root, "tree", buf.Bytes())
}

// ReadTree reads a tree object and returns its entries.
func ReadTree(root, hash string) ([]TreeEntry, error) {
	_, content, err := ReadObject(root, hash)
	if err != nil {
		return nil, err
	}
	return ParseTree(content)
}

// ParseTree parses tree object content into entries.
func ParseTree(data []byte) ([]TreeEntry, error) {
	var entries []TreeEntry
	for len(data) > 0 {
		// Find the null byte separating "mode name" from hash
		nullIdx := bytes.IndexByte(data, 0)
		if nullIdx < 0 {
			return nil, fmt.Errorf("invalid tree entry")
		}

		header := string(data[:nullIdx])
		spaceIdx := strings.IndexByte(header, ' ')
		if spaceIdx < 0 {
			return nil, fmt.Errorf("invalid tree entry header: %s", header)
		}

		mode := header[:spaceIdx]
		name := header[spaceIdx+1:]

		if len(data) < nullIdx+1+20 {
			return nil, fmt.Errorf("tree entry too short")
		}
		hash := hex.EncodeToString(data[nullIdx+1 : nullIdx+21])

		entries = append(entries, TreeEntry{Mode: mode, Name: name, Hash: hash})
		data = data[nullIdx+21:]
	}
	return entries, nil
}

// BuildTreeFromIndex builds a tree hierarchy from index entries and writes
// all tree objects to the store. Returns the root tree hash.
func BuildTreeFromIndex(root string, idx *index.Index) (string, error) {
	// Group entries by directory
	type dirEntry struct {
		name    string
		mode    string
		hash    string
		isTree  bool
		entries map[string]*dirEntry
	}

	rootDir := &dirEntry{entries: make(map[string]*dirEntry)}

	for _, e := range idx.Entries {
		parts := strings.Split(e.Path, "/")
		cur := rootDir
		for i, part := range parts {
			if i == len(parts)-1 {
				// Leaf blob
				cur.entries[part] = &dirEntry{
					name: part,
					mode: fmt.Sprintf("%o", e.Mode),
					hash: e.Hash,
				}
			} else {
				// Intermediate directory
				if _, ok := cur.entries[part]; !ok {
					cur.entries[part] = &dirEntry{
						name:    part,
						isTree:  true,
						entries: make(map[string]*dirEntry),
					}
				}
				cur = cur.entries[part]
			}
		}
	}

	// Recursively write trees
	var writeDir func(d *dirEntry) (string, error)
	writeDir = func(d *dirEntry) (string, error) {
		var treeEntries []TreeEntry
		for _, child := range d.entries {
			if child.isTree {
				childHash, err := writeDir(child)
				if err != nil {
					return "", err
				}
				treeEntries = append(treeEntries, TreeEntry{
					Mode: "40000",
					Name: child.name,
					Hash: childHash,
				})
			} else {
				treeEntries = append(treeEntries, TreeEntry{
					Mode: child.mode,
					Name: child.name,
					Hash: child.hash,
				})
			}
		}
		return WriteTree(root, treeEntries)
	}

	return writeDir(rootDir)
}

// FlattenTree recursively flattens a tree into a map of path→hash.
func FlattenTree(root, treeHash, prefix string) (map[string]string, error) {
	entries, err := ReadTree(root, treeHash)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, e := range entries {
		fullPath := e.Name
		if prefix != "" {
			fullPath = path.Join(prefix, e.Name)
		}
		if e.Mode == "40000" {
			sub, err := FlattenTree(root, e.Hash, fullPath)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				result[k] = v
			}
		} else {
			result[fullPath] = e.Hash
		}
	}
	return result, nil
}
