# gogit

A simplified Git implementation written in Go. Implements core version control operations including repository management, staging, commits, branching, merging, and diffs.

## Features

- **Repository initialization** (`init`)
- **File staging** with directory traversal and executable detection (`add`)
- **Working tree status** showing staged, unstaged, and untracked files (`status`)
- **Commits** with author info, timestamps, and parent tracking (`commit`)
- **Commit history** traversal (`log`)
- **Unified diffs** using LCS algorithm (`diff`)
- **Branch** creation and listing (`branch`)
- **Checkout** with working tree updates and empty directory cleanup (`checkout`)
- **Merge** with fast-forward detection, file-level 3-way merge, and conflict reporting (`merge`)

## Build

```
go build -o gogit .
```

## Usage

```
gogit init                        # Initialize a new repository
gogit add <path>...               # Stage files
gogit status                      # Show working tree status
gogit commit -m "message"         # Create a commit
gogit log                         # Show commit history
gogit diff                        # Show unstaged changes
gogit branch [name]               # List or create branches
gogit checkout <branch>           # Switch branches
gogit merge <branch>              # Merge a branch
```

## Architecture

```
.gogit/
  HEAD            # Current branch reference or detached commit hash
  objects/        # Zlib-compressed objects (blobs, trees, commits)
  refs/heads/     # Branch references
  index           # Binary staging area with SHA-1 integrity check
```

### Packages

| Package  | Purpose |
|----------|---------|
| `cmd`    | CLI command implementations |
| `object` | Object storage (blob, tree, commit) with zlib compression |
| `index`  | Binary index (staging area) with SHA-1 checksums |
| `refs`   | HEAD, branch reference management |
| `repo`   | Repository discovery and path helpers |

### Object Format

Objects are stored as `type size\0content`, zlib-compressed, addressed by their SHA-1 hash. The first two hex characters of the hash form the subdirectory name.

### Index Format

Custom binary format: `GIDX` magic, version, entry count, entries (ctime, mtime, size, hash, mode, path) with 8-byte padding, followed by a SHA-1 checksum.

## Configuration

Author information is read from environment variables:

```
export GOGIT_AUTHOR_NAME="Your Name"
export GOGIT_AUTHOR_EMAIL="you@example.com"
```

Falls back to the system username if not set.

## Testing

```
go test ./...
```

99%+ test coverage across all packages.
