package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"time"

	"leadcapture/api/internal/csvimport"
	"leadcapture/api/internal/db"
	"leadcapture/api/internal/httpapi"
)

//go:embed all:dist
var dist embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) (err error) {
	switch {
	case len(args) == 2 && args[0] == "import":
		return runImport(ctx, args[1], stdout, stderr)
	case len(args) == 1 && args[0] == "serve":
		return runServe(ctx)
	default:
		return errors.New("usage: leads import <file> | leads serve")
	}
}

func runImport(ctx context.Context, path string, stdout, stderr io.Writer) (err error) {
	file, err := os.Open(path)
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

func runServe(ctx context.Context) (err error) {
	store, err := db.Open(ctx, "leads.db")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.Close()) }()

	frontend, err := fs.Sub(dist, "dist")
	if err != nil {
		return err
	}

	server := &http.Server{Addr: ":8080", Handler: httpapi.New(store, frontend)}
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
