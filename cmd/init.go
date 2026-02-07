package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gogit/repo"
)

func Init() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	return initAt(wd)
}

func initAt(wd string) error {
	gogitDir := filepath.Join(wd, repo.GogitDir)
	if _, err := os.Stat(gogitDir); err == nil {
		return fmt.Errorf("already a gogit repository: %s", gogitDir)
	}

	dirs := []string{
		gogitDir,
		filepath.Join(gogitDir, "objects"),
		filepath.Join(gogitDir, "refs", "heads"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	headContent := "ref: refs/heads/main\n"
	if err := os.WriteFile(filepath.Join(gogitDir, "HEAD"), []byte(headContent), 0644); err != nil {
		return err
	}

	fmt.Printf("Initialized empty gogit repository in %s\n", gogitDir)
	return nil
}
