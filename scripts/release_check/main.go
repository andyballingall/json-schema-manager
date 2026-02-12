// Package main provides a script to check the GoReleaser configuration.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	_, err := exec.LookPath("goreleaser")
	if err != nil {
		_, _ = fmt.Println(
			"goreleaser not found. Install it with 'make setup' (macOS/Linux) or '.\\win-make.ps1 setup' (Windows)",
		)
		os.Exit(1)
	}

	_, _ = fmt.Println("Checking GoReleaser configuration...")
	cmd := exec.CommandContext(context.Background(), "goreleaser", "check")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		_, _ = fmt.Printf("❌ Release check failed: %v\n", err)
		os.Exit(1)
	}
}
