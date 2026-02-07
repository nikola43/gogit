package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gogit/index"
	"gogit/object"
	"gogit/repo"
)

func Diff() error {
	root, err := repo.Find()
	if err != nil {
		return err
	}

	idx, err := index.ReadIndex(root)
	if err != nil {
		return err
	}

	for _, e := range idx.Entries {
		absPath := filepath.Join(root, e.Path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				// File deleted — show full removal
				oldContent, err := object.ReadBlob(root, e.Hash)
				if err != nil {
					continue
				}
				printUnifiedDiff(e.Path, strings.Split(string(oldContent), "\n"), nil)
			}
			continue
		}

		currentHash := object.HashBlob(content)
		if currentHash == e.Hash {
			continue
		}

		oldContent, err := object.ReadBlob(root, e.Hash)
		if err != nil {
			continue
		}

		oldLines := strings.Split(string(oldContent), "\n")
		newLines := strings.Split(string(content), "\n")
		printUnifiedDiff(e.Path, oldLines, newLines)
	}

	return nil
}

func printUnifiedDiff(path string, oldLines, newLines []string) {
	fmt.Printf("--- a/%s\n", path)
	fmt.Printf("+++ b/%s\n", path)

	if newLines == nil {
		// File deleted
		fmt.Printf("@@ -1,%d +0,0 @@\n", len(oldLines))
		for _, line := range oldLines {
			fmt.Printf("-%s\n", line)
		}
		return
	}

	// LCS-based diff
	lcs := computeLCS(oldLines, newLines)
	hunks := buildHunks(oldLines, newLines, lcs)

	for _, hunk := range hunks {
		fmt.Println(hunk)
	}
}

// computeLCS computes the longest common subsequence table.
func computeLCS(a, b []string) [][]int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp
}

// buildHunks generates unified diff hunks from the LCS table.
func buildHunks(oldLines, newLines []string, dp [][]int) []string {
	// Backtrack to find the diff operations
	type diffLine struct {
		op   byte // ' ', '+', '-'
		text string
	}

	var diff []diffLine
	i, j := len(oldLines), len(newLines)
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			diff = append(diff, diffLine{' ', oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			diff = append(diff, diffLine{'+', newLines[j-1]})
			j--
		} else {
			diff = append(diff, diffLine{'-', oldLines[i-1]})
			i--
		}
	}

	// Reverse the diff (we built it backwards)
	for l, r := 0, len(diff)-1; l < r; l, r = l+1, r-1 {
		diff[l], diff[r] = diff[r], diff[l]
	}

	// Build hunks with context
	const contextLines = 3
	var result []string
	var hunkLines []string
	hunkOldStart, hunkNewStart := 0, 0
	hunkOldCount, hunkNewCount := 0, 0
	oldLine, newLine := 1, 1
	lastChange := -1

	flushHunk := func() {
		if len(hunkLines) > 0 {
			header := fmt.Sprintf("@@ -%d,%d +%d,%d @@",
				hunkOldStart, hunkOldCount, hunkNewStart, hunkNewCount)
			result = append(result, header)
			result = append(result, hunkLines...)
			hunkLines = nil
			hunkOldCount = 0
			hunkNewCount = 0
		}
	}

	for idx, d := range diff {
		isChange := d.op != ' '

		if isChange {
			if lastChange < 0 || idx-lastChange > 2*contextLines {
				flushHunk()
				// Start new hunk with context
				start := idx - contextLines
				if start < 0 {
					start = 0
				}
				tmpOld, tmpNew := 1, 1
				for k := 0; k < start; k++ {
					if diff[k].op != '+' {
						tmpOld++
					}
					if diff[k].op != '-' {
						tmpNew++
					}
				}
				hunkOldStart = tmpOld
				hunkNewStart = tmpNew
				hunkOldCount = 0
				hunkNewCount = 0
				for k := start; k < idx; k++ {
					line := fmt.Sprintf(" %s", diff[k].text)
					hunkLines = append(hunkLines, line)
					hunkOldCount++
					hunkNewCount++
				}
			} else {
				// Add intervening context
				for k := lastChange + 1; k < idx; k++ {
					line := fmt.Sprintf(" %s", diff[k].text)
					hunkLines = append(hunkLines, line)
					hunkOldCount++
					hunkNewCount++
				}
			}
			lastChange = idx
		}

		if isChange {
			hunkLines = append(hunkLines, fmt.Sprintf("%c%s", d.op, d.text))
			if d.op == '-' {
				hunkOldCount++
			} else {
				hunkNewCount++
			}
		}

		if d.op != '+' {
			oldLine++
		}
		if d.op != '-' {
			newLine++
		}
	}

	// Add trailing context after last change
	if lastChange >= 0 {
		end := lastChange + contextLines + 1
		if end > len(diff) {
			end = len(diff)
		}
		for k := lastChange + 1; k < end; k++ {
			line := fmt.Sprintf(" %s", diff[k].text)
			hunkLines = append(hunkLines, line)
			hunkOldCount++
			hunkNewCount++
		}
		flushHunk()
	}

	_ = oldLine
	_ = newLine

	return result
}
