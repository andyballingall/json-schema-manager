package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	coverageFile := "coverage.out"
	if len(os.Args) > 1 {
		coverageFile = os.Args[1]
	}

	cmd := exec.Command("go", "tool", "cover", "-func", coverageFile)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Error running go tool cover: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var failures []string
	var totalCoverage string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// The last line is the total coverage
		if strings.HasPrefix(line, "total:") {
			totalCoverage = line
			continue
		}

		// Skip the header or other non-function lines
		if !strings.Contains(line, ":") {
			continue
		}

		// Exclude scripts directory
		if strings.Contains(line, "/scripts/") {
			continue
		}

		// Check if it's the main function in main.go
		if strings.Contains(line, "main.go") && strings.Contains(line, "main") {
			continue
		}

		// Logic: If it doesn't contain "100.0%", it's a failure
		if !strings.Contains(line, "100.0%") {
			failures = append(failures, line)
		}
	}

	if len(failures) > 0 {
		fmt.Println("❌ Coverage check failed! The following functions have less than 100% coverage:")
		for _, f := range failures {
			fmt.Printf("  %s\n", f)
		}
		os.Exit(1)
	}

	fmt.Printf("✅ All non-main functions have 100%% coverage!\n")
	if totalCoverage != "" {
		fmt.Printf("📊 %s\n", totalCoverage)
	}
}
