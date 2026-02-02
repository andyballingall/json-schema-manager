package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)

	// Local lazy instance ensures t.Parallel() safety
	lazy := &LazyManager{}

	rootCmd := NewRootCmd(lazy, logLevel, stderr)
	rootCmd.SetArgs(args[1:]) // Skip the program name
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		// Print error to stderr for script tests and CLI users (SilenceErrors is set)
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return err
	}

	return nil
}
