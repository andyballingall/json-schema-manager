package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	args := os.Args[1:]

	// Separate custom script flags from go test flags
	var testArgs []string
	checkCoverage := false
	showSummary := false
	openBrowser := false
	generateBadge := false
	coverageFile := ""

	for _, arg := range args {
		switch {
		case arg == "--check-coverage":
			checkCoverage = true
		case arg == "--summary":
			showSummary = true
		case arg == "--browser":
			openBrowser = true
		case arg == "--badge":
			generateBadge = true
		case strings.HasPrefix(arg, "-coverprofile="):
			coverageFile = strings.TrimPrefix(arg, "-coverprofile=")
			testArgs = append(testArgs, arg)
		default:
			testArgs = append(testArgs, arg)
		}
	}

	// Any coverage-related flag triggers the coverage setup
	isCoverageRun := checkCoverage || showSummary || openBrowser || generateBadge

	if isCoverageRun {
		if coverageFile == "" {
			coverageFile = "coverage.out"
			testArgs = append(testArgs, "-coverprofile="+coverageFile)
		}
		// Ensure we are covering the internal packages
		hasCoverPkg := false
		for _, arg := range testArgs {
			if strings.HasPrefix(arg, "-coverpkg") {
				hasCoverPkg = true
				break
			}
		}
		if !hasCoverPkg {
			testArgs = append(testArgs, "-coverpkg=./internal/...")
		}
	}

	// Run tests
	_, err := exec.LookPath("gotestsum")
	if err == nil && !isCoverageRun {
		runCommand("gotestsum", append([]string{"--"}, testArgs...))
	} else {
		runCommand("go", append([]string{"test"}, testArgs...))
	}

	// Post-test actions
	switch {
	case checkCoverage:
		checkCoverageThresholds(coverageFile)
	case showSummary:
		runCommand("go", []string{"tool", "cover", "-func", coverageFile})
	case openBrowser:
		runCommand("go", []string{"tool", "cover", "-html", coverageFile})
	case generateBadge:
		generateCoverageBadge(coverageFile)
	}
}

func runCommand(name string, args []string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Command failed: %v\n", err)
		os.Exit(1)
	}
}

func checkCoverageThresholds(coverageFile string) {
	cmd := exec.Command("go", "tool", "cover", "-func", coverageFile)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Error running go tool cover: %v\n", err)
		os.Exit(1)
	}

	failures, totalLine := parseCoverageOutput(output)

	if len(failures) > 0 {
		fmt.Println("\n❌ Coverage check failed! The following functions have less than 100% coverage:")
		for _, f := range failures {
			fmt.Printf("  %s\n", f)
		}
		os.Exit(1)
	}

	if totalLine != "" {
		fmt.Printf("\n📊 %s\n", totalLine)
	}
	fmt.Printf("✅ Coverage check passed - 100%% coverage of internal packages achieved, " +
		"excluding approved exceptions\n")
}

func generateCoverageBadge(coverageFile string) {
	cmd := exec.Command("go", "tool", "cover", "-func", coverageFile)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Error running go tool cover: %v\n", err)
		os.Exit(1)
	}

	_, totalLine := parseCoverageOutput(output)
	if totalLine == "" {
		fmt.Println("❌ Could not find total coverage in output")
		os.Exit(1)
	}

	parts := strings.Fields(totalLine)
	percentageStr := parts[len(parts)-1] // e.g. "99.8%"

	var percentage float64
	if _, err := fmt.Sscanf(strings.TrimSuffix(percentageStr, "%"), "%f", &percentage); err != nil {
		fmt.Printf("❌ Error parsing coverage percentage: %v\n", err)
		os.Exit(1)
	}

	color := "#e05d44" // red
	switch {
	case percentage >= 100:
		color = "#4c1" // green
	case percentage >= 90:
		color = "#a4a61d" // yellowgreen
	case percentage >= 80:
		color = "#dfb317" // yellow
	case percentage >= 70:
		color = "#fe7d37" // orange
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="104" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a"><rect width="104" height="20" rx="3" fill="#fff"/></mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h67v20H0z"/>
    <path fill="%s" d="M67 0h37v20H67z"/>
    <path fill="url(#b)" d="M0 0h104v20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="33.5" y="15" fill="#010101" fill-opacity=".3">coverage</text>
    <text x="33.5" y="14">coverage</text>
    <text x="84.5" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="84.5" y="14">%s</text>
  </g>
</svg>`, color, percentageStr, percentageStr)

	err = os.WriteFile("coverage.svg", []byte(svg), 0o600)
	if err != nil {
		fmt.Printf("❌ Error writing coverage.svg: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Coverage badge generated: coverage.svg (%s)\n", percentageStr)
}

func parseCoverageOutput(output []byte) (failures []string, totalLine string) {
	// Function-level exclusions (Package:Function)
	exclusions := map[string]float64{
		// filepath.Abs() error path is unreachable on Darwin
		"github.com/andyballingall/json-schema-manager/internal/fs/path_resolver.go:27": 85.0,
		// os.IsNotExist(err) false path is hard to trigger reliably
		"github.com/andyballingall/json-schema-manager/internal/schema/registry.go:131": 90.0,
		// break Loop branch inside select is race-prone item to test
		"github.com/andyballingall/json-schema-manager/internal/schema/tester.go:346": 94.0,
		// singleflight double-check cache path is non-deterministic due to goroutine scheduling
		"github.com/andyballingall/json-schema-manager/internal/schema/registry.go:151": 90.0,
		// WatchValidation callback execution depends on file system event timing
		"github.com/andyballingall/json-schema-manager/internal/app/manager.go:193": 97.0,
		// TextReporter.Write coverage depends on WatchValidation callback timing
		"github.com/andyballingall/json-schema-manager/internal/report/text.go:38": 98.0,
		// BuildAll parallel error handling is hard to trigger reliably
		"github.com/andyballingall/json-schema-manager/internal/schema/dist.go:65": 97.0,
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "total:") {
			totalLine = line
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
	return failures, totalLine
}

func shouldSkipLine(line string) bool {
	if !strings.Contains(line, ":") {
		return true
	}
	if strings.Contains(line, "/scripts/") {
		return true
	}
	// Skip main functions in main.go as they are entry points
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
