package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/andyballingall/json-schema-manager/internal/app"
)

func main() {
	// Create context that cancels on SIGINT (Ctrl+C) or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args, os.Stdout, os.Stderr); err != nil {
		//nolint:gocritic // os.Exit is intentional
		os.Exit(1)
	}
}
