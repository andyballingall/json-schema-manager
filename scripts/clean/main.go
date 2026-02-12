// Package main provides a script to clean up build and test artefacts.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	cleanDirs([]string{"bin", "dist"})
	cleanFiles([]string{".jsm.log"})
	cleanPatterns([]string{"coverage*", "*.out", "*.test", "*.coverprofile", "profile.cov"})
}

func cleanDirs(dirs []string) {
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			_, _ = fmt.Printf("❌ Failed to remove dir %s: %v\n", dir, err)
		} else {
			_, _ = fmt.Printf("✅ Removed dir %s\n", dir)
		}
	}
}

func cleanFiles(files []string) {
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Printf("❌ Failed to remove file %s: %v\n", file, err)
		} else if err == nil {
			_, _ = fmt.Printf("✅ Removed file %s\n", file)
		}
	}
}

func cleanPatterns(patterns []string) {
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			_, _ = fmt.Printf("❌ Failed to glob pattern %s: %v\n", pattern, err)
			continue
		}
		for _, match := range matches {
			if rErr := os.Remove(match); rErr != nil {
				_, _ = fmt.Printf("❌ Failed to remove matched file %s: %v\n", match, rErr)
			} else {
				_, _ = fmt.Printf("✅ Removed matched file %s\n", match)
			}
		}
	}
}
