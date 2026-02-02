package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	binaryName := "jsm"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	// Get version
	versionCmd := exec.Command("go", "run", "scripts/version/main.go")
	versionOut, _ := versionCmd.Output()
	version := string(versionOut)

	ldflags := fmt.Sprintf("-X github.com/andyballingall/json-schema-manager/internal/app.Version=%s", version)

	// Ensure bin directory exists
	if err := os.MkdirAll("bin", 0o755); err != nil {
		fmt.Printf("❌ Failed to create bin directory: %v\n", err)
		os.Exit(1)
	}

	outputPath := filepath.Join("bin", binaryName)
	fmt.Printf("Building %s...\n", version)

	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", outputPath, "cmd/jsm/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Build complete: %s\n", outputPath)
}
