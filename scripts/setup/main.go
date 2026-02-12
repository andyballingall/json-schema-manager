// Package main provides a script to set up the development environment.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	tools := map[string]string{
		"lefthook":      "github.com/evilmartians/lefthook@latest",
		"golangci-lint": "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0",
		"goreleaser":    "github.com/goreleaser/goreleaser/v2@latest",
		"staticcheck":   "honnef.co/go/tools/cmd/staticcheck@latest",
		"gotestsum":     "gotest.tools/gotestsum@latest",
		"gofumpt":       "mvdan.cc/gofumpt@latest",
	}

	for tool, path := range tools {
		if !isToolInstalled(tool) {
			_, _ = fmt.Printf("📦 Installing %s...\n", tool)
			if err := installTool(path); err != nil {
				_, _ = fmt.Printf("❌ Failed to install %s: %v\n", tool, err)
			} else {
				_, _ = fmt.Printf("✅ Installed %s\n", tool)
			}
		} else {
			_, _ = fmt.Printf("✅ %s is already installed\n", tool)
		}
	}

	_, _ = fmt.Println("🚀 Installing lefthook hooks...")
	if err := runCommand("lefthook", "install"); err != nil {
		_, _ = fmt.Printf("❌ Failed to install lefthook hooks: %v\n", err)
	} else {
		_, _ = fmt.Println("✅ Lefthook hooks installed!")
	}
}

func isToolInstalled(name string) bool {
	_, err := exec.LookPath(name)
	if err == nil {
		return true
	}

	// Also check GOPATH/bin
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		home, _ := os.UserHomeDir()
		goPath = filepath.Join(home, "go")
	}
	binName := name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	_, err = os.Stat(filepath.Join(goPath, "bin", binName))
	return err == nil
}

func installTool(path string) error {
	return runCommand("go", "install", path)
}

func runCommand(name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		// Try to find it in GOPATH/bin
		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			home, _ := os.UserHomeDir()
			goPath = filepath.Join(home, "go")
		}
		binName := name
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		fullPath := filepath.Join(goPath, "bin", binName)
		if _, statErr := os.Stat(fullPath); statErr == nil {
			path = fullPath
		} else {
			return fmt.Errorf("%s not found in PATH or %s", name, fullPath)
		}
	}

	cmd := exec.CommandContext(context.Background(), path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
