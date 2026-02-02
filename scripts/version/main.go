package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Print("dev")
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
}
