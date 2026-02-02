package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	_, err := exec.LookPath("gofumpt")
	if err != nil {
		fmt.Println("gofumpt not found. Install it with 'make setup' (macOS/Linux) or '.\\win-make.ps1 setup' (Windows)")
		os.Exit(1)
	}

	fmt.Println("Formatting with gofumpt...")
	cmd := exec.Command("gofumpt", "-l", "-w", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		fmt.Printf("❌ Formatting failed: %v\n", err)
		os.Exit(1)
	}
}
