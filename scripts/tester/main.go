package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	args := os.Args[1:]

	// Detect if coverage is requested.
	// gotestsum with multiple packages can overwrite the coverage profile.
	hasCover := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "-cover") {
			hasCover = true
			break
		}
	}

	// Check if gotestsum is available and we're not doing coverage
	_, err := exec.LookPath("gotestsum")
	if err == nil && !hasCover {
		runTest("gotestsum", append([]string{"--"}, args...))
		return
	}

	// Fallback to go test
	runTest("go", append([]string{"test"}, args...))
}

func runTest(name string, args []string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Tests failed: %v\n", err)
		os.Exit(1)
	}
}
