// Package main provides a script to get the current version from git.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.CommandContext(context.Background(), "git", "describe", "--tags", "--always", "--dirty")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Print("dev")
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
}
