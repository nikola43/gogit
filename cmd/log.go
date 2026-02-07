package cmd

import (
	"fmt"

	"gogit/object"
	"gogit/refs"
	"gogit/repo"
)

func Log() error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	hash, err := refs.ResolveHead(root)
	if err != nil {
		return err
	}
	if hash == "" {
		fmt.Println("No commits yet")
		return nil
	}

	for hash != "" {
		commit, err := object.ReadCommit(root, hash)
		if err != nil {
			return err
		}

		fmt.Printf("commit %s\n", hash)
		fmt.Printf("Author: %s\n", commit.Author)
		fmt.Println()
		fmt.Printf("    %s\n", commit.Message)
		fmt.Println()

		if len(commit.Parents) > 0 {
			hash = commit.Parents[0]
		} else {
			hash = ""
		}
	}

	return nil
}
