package object

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"
)

// Commit represents a parsed commit object.
type Commit struct {
	TreeHash  string
	Parents   []string
	Author    string
	Committer string
	Message   string
}

// WriteCommit creates a commit object and returns its hash.
func WriteCommit(root, treeHash string, parents []string, message string) (string, error) {
	author := formatAuthor()
	timestamp := formatTimestamp()

	var buf strings.Builder
	fmt.Fprintf(&buf, "tree %s\n", treeHash)
	for _, p := range parents {
		fmt.Fprintf(&buf, "parent %s\n", p)
	}
	fmt.Fprintf(&buf, "author %s %s\n", author, timestamp)
	fmt.Fprintf(&buf, "committer %s %s\n", author, timestamp)
	fmt.Fprintf(&buf, "\n%s\n", message)

	return WriteObject(root, "commit", []byte(buf.String()))
}

// ReadCommit reads and parses a commit object.
func ReadCommit(root, hash string) (*Commit, error) {
	_, content, err := ReadObject(root, hash)
	if err != nil {
		return nil, err
	}
	return ParseCommit(content)
}

// ParseCommit parses commit content into a Commit struct.
func ParseCommit(data []byte) (*Commit, error) {
	c := &Commit{}
	text := string(data)

	// Split headers from message at first blank line
	parts := strings.SplitN(text, "\n\n", 2)
	if len(parts) == 2 {
		c.Message = strings.TrimSpace(parts[1])
	}

	for _, line := range strings.Split(parts[0], "\n") {
		if strings.HasPrefix(line, "tree ") {
			c.TreeHash = strings.TrimPrefix(line, "tree ")
		} else if strings.HasPrefix(line, "parent ") {
			c.Parents = append(c.Parents, strings.TrimPrefix(line, "parent "))
		} else if strings.HasPrefix(line, "author ") {
			c.Author = strings.TrimPrefix(line, "author ")
		} else if strings.HasPrefix(line, "committer ") {
			c.Committer = strings.TrimPrefix(line, "committer ")
		}
	}

	return c, nil
}

// userLookup is a variable wrapping user.Current so tests can override it.
var userLookup = user.Current

func formatAuthor() string {
	name := os.Getenv("GOGIT_AUTHOR_NAME")
	if name == "" {
		if u, err := userLookup(); err == nil {
			name = u.Username
		} else {
			name = "Unknown"
		}
	}
	email := os.Getenv("GOGIT_AUTHOR_EMAIL")
	if email == "" {
		email = name + "@localhost"
	}
	return fmt.Sprintf("%s <%s>", name, email)
}

func formatTimestamp() string {
	now := time.Now()
	_, offset := now.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	return fmt.Sprintf("%d %s%02d%02d", now.Unix(), sign, hours, minutes)
}
