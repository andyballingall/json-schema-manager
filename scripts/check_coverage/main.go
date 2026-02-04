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

	output, err := runCoverTool(coverageFile)
	if err != nil {
		fmt.Printf("❌ Error running go tool cover: %v\n", err)
		os.Exit(1)
	}

	failures, totalCoverage := parseCoverageOutput(output)

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

func runCoverTool(coverageFile string) ([]byte, error) {
	cmd := exec.Command("go", "tool", "cover", "-func", coverageFile)
	return cmd.Output()
}

func parseCoverageOutput(output []byte) (failures []string, totalCoverage string) {
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Function-level exclusions (Package:Function)
	exclusions := map[string]float64{
		// filepath.Abs() error path is unreachable on Darwin
		"github.com/andyballingall/json-schema-manager/internal/fs/path_resolver.go:27": 85.0,
		// os.IsNotExist(err) false path is hard to trigger reliably
		"github.com/andyballingall/json-schema-manager/internal/schema/registry.go:139": 90.0,
		// break Loop branch inside select is race-prone item to test
		"github.com/andyballingall/json-schema-manager/internal/schema/tester.go:346": 94.0,
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "total:") {
			totalCoverage = line
			continue
		}

		if shouldSkipLine(line) {
			continue
		}

		if isLineExcluded(line, exclusions) {
			continue
		}

		if !strings.Contains(line, "100.0%") {
			failures = append(failures, line)
		}
	}

	return failures, totalCoverage
}

func shouldSkipLine(line string) bool {
	if !strings.Contains(line, ":") {
		return true
	}
	if strings.Contains(line, "/scripts/") {
		return true
	}
	if strings.Contains(line, "main.go") && strings.Contains(line, "main") {
		return true
	}
	return false
}

func isLineExcluded(line string, exclusions map[string]float64) bool {
	for pattern, threshold := range exclusions {
		if !strings.Contains(line, pattern) {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		percentageStr := strings.TrimSuffix(parts[len(parts)-1], "%")
		var percentage float64
		if _, err := fmt.Sscanf(percentageStr, "%f", &percentage); err == nil {
			if percentage >= threshold {
				return true
			}
		}
	}
	return false
}
