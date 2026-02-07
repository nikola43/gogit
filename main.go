package main

import (
	"fmt"
	"os"

	"gogit/cmd"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) < 2 {
		usage()
		return 1
	}

	var err error
	switch args[1] {
	case "init":
		err = cmd.Init()
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: gogit add <path>...")
			return 1
		}
		err = cmd.Add(args[2:])
	case "status":
		err = cmd.Status()
	case "commit":
		msg := ""
		for i := 2; i < len(args); i++ {
			if args[i] == "-m" && i+1 < len(args) {
				msg = args[i+1]
				break
			}
		}
		if msg == "" {
			fmt.Fprintln(os.Stderr, "usage: gogit commit -m \"message\"")
			return 1
		}
		err = cmd.Commit(msg)
	case "log":
		err = cmd.Log()
	case "diff":
		err = cmd.Diff()
	case "branch":
		name := ""
		if len(args) >= 3 {
			name = args[2]
		}
		err = cmd.Branch(name)
	case "checkout":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: gogit checkout <branch>")
			return 1
		}
		err = cmd.Checkout(args[2])
	case "merge":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: gogit merge <branch>")
			return 1
		}
		err = cmd.Merge(args[2])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[1])
		usage()
		return 1
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gogit <command> [<args>]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init       Create a new repository")
	fmt.Fprintln(os.Stderr, "  add        Add files to staging area")
	fmt.Fprintln(os.Stderr, "  status     Show working tree status")
	fmt.Fprintln(os.Stderr, "  commit     Record changes to repository")
	fmt.Fprintln(os.Stderr, "  log        Show commit history")
	fmt.Fprintln(os.Stderr, "  diff       Show changes in working tree")
	fmt.Fprintln(os.Stderr, "  branch     List or create branches")
	fmt.Fprintln(os.Stderr, "  checkout   Switch branches")
	fmt.Fprintln(os.Stderr, "  merge      Merge a branch")
}
