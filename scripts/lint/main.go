package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	_, err := exec.LookPath("golangci-lint")
	if err != nil {
		fmt.Println("golangci-lint not found. Install it with 'make setup' (macOS/Linux)" +
			" or '.\\win-make.ps1 setup' (Windows)")
		os.Exit(1)
	}

	fmt.Println("Linting with golangci-lint...")
	cmd := exec.Command("golangci-lint", "run")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		fmt.Printf("❌ Linting failed: %v\n", err)
		os.Exit(1)
	}
}
