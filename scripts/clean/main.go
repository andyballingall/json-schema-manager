package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Directories to clean
	dirs := []string{"bin", "dist"}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("❌ Failed to remove dir %s: %v\n", dir, err)
		} else {
			fmt.Printf("✅ Removed dir %s\n", dir)
		}
	}

	// Individual files to clean
	files := []string{".jsm.log"}
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			fmt.Printf("❌ Failed to remove file %s: %v\n", file, err)
		} else if err == nil {
			fmt.Printf("✅ Removed file %s\n", file)
		}
	}

	// Patterns to clean
	patterns := []string{"coverage*", "*.out", "*.test", "*.coverprofile", "profile.cov"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Printf("❌ Failed to glob pattern %s: %v\n", pattern, err)
			continue
		}
		for _, match := range matches {
			if rErr := os.Remove(match); rErr != nil {
				fmt.Printf("❌ Failed to remove matched file %s: %v\n", match, rErr)
			} else {
				fmt.Printf("✅ Removed matched file %s\n", match)
			}
		}
	}
}
