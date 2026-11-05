package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"leadcapture/api/internal/csvimport"
	"leadcapture/api/internal/db"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) (err error) {
	if len(args) != 2 || args[0] != "import" {
		return errors.New("usage: leads import <file>")
	}
	file, err := os.Open(args[1])
	if err != nil {
		return fmt.Errorf("open CSV: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()

	store, err := db.Open(ctx, "leads.db")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.Close()) }()

	result, importErr := csvimport.Import(ctx, file, store)
	for _, diagnostic := range result.Skipped {
		if _, err := fmt.Fprintf(stderr, "line %d: %s\n", diagnostic.Line, diagnostic.Err); err != nil {
			return errors.Join(importErr, err)
		}
	}
	_, writeErr := fmt.Fprintf(stdout, "Imported %d leads; skipped %d rows.\n", result.Imported, len(result.Skipped))
	return errors.Join(importErr, writeErr)
}
