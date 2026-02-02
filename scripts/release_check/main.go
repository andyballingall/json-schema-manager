package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	_, err := exec.LookPath("goreleaser")
	if err != nil {
		fmt.Println("goreleaser not found. Install it with 'make setup' (macOS/Linux) or '.\\win-make.ps1 setup' (Windows)")
		os.Exit(1)
	}

	fmt.Println("Checking GoReleaser configuration...")
	cmd := exec.Command("goreleaser", "check")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		fmt.Printf("❌ Release check failed: %v\n", err)
		os.Exit(1)
	}
}
