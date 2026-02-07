package cmd

import (
	"fmt"

	"gogit/refs"
	"gogit/repo"
)

func Branch(name string) error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	if name == "" {
		return listBranches(root)
	}

	return createBranch(root, name)
}

func listBranches(root string) error {
	branches, err := refs.ListBranches(root)
	if err != nil {
		return err
	}

	current, err := refs.CurrentBranch(root)
	if err != nil {
		return err
	}

	for _, b := range branches {
		if b == current {
			fmt.Printf("* %s\n", b)
		} else {
			fmt.Printf("  %s\n", b)
		}
	}
	return nil
}

func createBranch(root, name string) error {
	// Check if branch already exists
	existing, err := refs.ReadRef(root, refs.BranchRef(name))
	if err != nil {
		return err
	}
	if existing != "" {
		return fmt.Errorf("branch '%s' already exists", name)
	}

	// Get current HEAD commit
	hash, err := refs.ResolveHead(root)
	if err != nil {
		return err
	}
	if hash == "" {
		return fmt.Errorf("cannot create branch: no commits yet")
	}

	if err := refs.WriteRef(root, refs.BranchRef(name), hash); err != nil {
		return err
	}

	fmt.Printf("Created branch '%s' at %s\n", name, hash[:7])
	return nil
}
