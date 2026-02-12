package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/andyballingall/json-schema-manager/internal/fsh"
)

// Run executes the application with the given arguments.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, envProvider fsh.EnvProvider) error {
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)

	// Local lazy instance ensures t.Parallel() safety
	lazy := &LazyManager{}

	if envProvider == nil {
		envProvider = fsh.NewEnvProvider()
	}

	rootCmd := NewRootCmd(lazy, logLevel, stdout, stderr, envProvider)
	rootCmd.SetArgs(args[1:]) // Skip the program name
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			_, _ = fmt.Fprintln(stderr, "Interrupted by user")
			return nil
		}
		// Print error to stderr for script tests and CLI users (SilenceErrors is set)
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return err
	}

	return nil
}
