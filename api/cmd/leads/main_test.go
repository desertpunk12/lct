package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"leadcapture/api/internal/db"
)

func TestImportCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	data := "first_name,last_name,email,finance_amount,state\n" +
		"Zoë,Martin,ZOE@example.com,100,victoria\n" +
		"Different,Person,zoe@EXAMPLE.COM,200,NSW\n" +
		"A,B,,100,NSW\n"
	if err := os.WriteFile("input.csv", []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"import", "input.csv"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Imported 1 leads; skipped 2 rows.\n" {
		t.Fatalf("unexpected summary: %q", got)
	}
	wantErrors := "line 3: email: duplicate email\nline 4: email: is required\n"
	if got := stderr.String(); got != wantErrors {
		t.Fatalf("got %q, want %q", got, wantErrors)
	}

	store, err := db.Open(context.Background(), "leads.db")
	if err != nil {
		t.Fatal(err)
	}
	leads, err := store.ListLeads(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if len(leads) != 1 || leads[0].FirstName != "Zoë" || leads[0].FinanceAmount != 100 || leads[0].State != "VIC" {
		t.Fatalf("duplicate overwrote original: %+v", leads)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run(context.Background(), []string{"import", "input.csv"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Imported 0 leads; skipped 3 rows.\n" || strings.Count(stderr.String(), "duplicate email") != 2 {
		t.Fatalf("reimport changed data: stdout %q, stderr %q", &stdout, &stderr)
	}
}

func TestCommandArguments(t *testing.T) {
	cases := [][]string{
		nil,
		{"import"},
		{"import", "one.csv", "two.csv"},
		{"unknown", "file.csv"},
		{"serve", "--port", "9090"},
	}
	for _, args := range cases {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil || err.Error() != "usage: leads import <file> | leads serve" {
			t.Errorf("args %v: got %v", args, err)
		}
	}
}

func TestMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := run(context.Background(), []string{"import", "missing.csv"}, io.Discard, io.Discard); err == nil || !strings.Contains(err.Error(), "open CSV:") {
		t.Fatalf("expected file error, got %v", err)
	}
	if _, err := os.Stat("leads.db"); !os.IsNotExist(err) {
		t.Fatalf("should not create a database for a missing file, got %v", err)
	}
}

func TestMalformedFileIsFatal(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("input.csv", []byte("email\na@example.com\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := run(context.Background(), []string{"import", "input.csv"}, &stdout, io.Discard); err == nil || !strings.Contains(err.Error(), "missing required CSV header") {
		t.Fatalf("expected header error, got %v", err)
	}
	if stdout.String() != "Imported 0 leads; skipped 0 rows.\n" {
		t.Fatalf("unexpected summary: %q", &stdout)
	}
}
