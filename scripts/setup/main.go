// Package main provides a script to set up the development environment.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var workflowFlag = flag.String("workflow", "local", "Workflow: local, ci, or coverage")

func main() {
	flag.Parse()

	workflow := *workflowFlag
	tools := getToolsForWorkflow(workflow)

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

	if workflow == "local" {
		_, _ = fmt.Println("🚀 Installing lefthook hooks...")
		if err := runCommand("lefthook", "install"); err != nil {
			_, _ = fmt.Printf("❌ Failed to install lefthook hooks: %v\n", err)
		} else {
			_, _ = fmt.Println("✅ Lefthook hooks installed!")
		}
	}
}

func getToolsForWorkflow(workflow string) map[string]string {
	allTools := map[string]string{
		"lefthook":      "github.com/evilmartians/lefthook@v1.7.1",
		"golangci-lint": "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0",
		"goreleaser":    "github.com/goreleaser/goreleaser/v2@v2.3.0",
		"staticcheck":   "honnef.co/go/tools/cmd/staticcheck@2023.1.7",
		"gotestsum":     "gotest.tools/gotestsum@v1.12.0",
		"gofumpt":       "mvdan.cc/gofumpt@v0.7.0",
	}

	switch workflow {
	case "ci":
		return map[string]string{
			"golangci-lint": allTools["golangci-lint"],
			"staticcheck":   allTools["staticcheck"],
			"gotestsum":     allTools["gotestsum"],
			"gofumpt":       allTools["gofumpt"],
		}
	case "coverage":
		return map[string]string{
			"gotestsum": allTools["gotestsum"],
		}
	default:
		return allTools
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
