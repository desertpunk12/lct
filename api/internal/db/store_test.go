package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"leadcapture/api/internal/db"
	"leadcapture/api/internal/lead"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func openStore(t *testing.T, path string) *db.Store {
	t.Helper()
	store, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	return store
}

func input() lead.Input {
	return lead.Input{
		FirstName: " Zoë ", LastName: "Martin", Email: " ZOE@Example.COM ",
		FinanceAmount: 38500.5, State: " Queensland ",
	}
}

func TestEmptyListIsArray(t *testing.T) {
	store := openStore(t, ":memory:")
	leads, err := store.ListLeads(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(leads)
	if err != nil || string(data) != "[]" {
		t.Fatalf("got %s, error %v; want []", data, err)
	}
}

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "leads.db")
	store := openStore(t, path)
	first, err := store.CreateLead(ctx, input())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 || first.CreatedAt.IsZero() || first.CreatedAt.Location() != time.UTC {
		t.Fatalf("invalid generated fields: %+v", first)
	}
	if first.Email != "zoe@example.com" || first.FirstName != "Zoë" || first.State != "QLD" || first.Salary != nil || first.Vehicle != nil {
		t.Fatalf("unexpected normalization/nulls: %+v", first)
	}

	salary := 100000.5
	year := 1990
	secondInput := input()
	secondInput.Email = "second@example.com"
	secondInput.Salary = &salary
	secondInput.Vehicle = &lead.Vehicle{Make: "Kia", Year: &year}
	second, err := store.CreateLead(ctx, secondInput)
	if err != nil {
		t.Fatal(err)
	}
	model := "EV6"
	thirdInput := input()
	thirdInput.Email = "third@example.com"
	thirdInput.Vehicle = &lead.Vehicle{Make: "Kia", Model: &model}
	third, err := store.CreateLead(ctx, thirdInput)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openStore(t, path)
	got, err := reopened.ListLeads(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []lead.Lead{first, second, third}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("database round trip changed leads: got %+v, want %+v", got, want)
	}
	if got[1].Vehicle.Model != nil || got[2].Vehicle.Year != nil {
		t.Fatal("partial vehicle nulls were not preserved")
	}
}

func TestDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, ":memory:")
	original, err := store.CreateLead(ctx, input())
	if err != nil {
		t.Fatal(err)
	}
	duplicate := input()
	duplicate.Email = "zoe@EXAMPLE.com"
	duplicate.FirstName = "Must not overwrite"
	_, err = store.CreateLead(ctx, duplicate)
	if !errors.Is(err, db.ErrDuplicateEmail) {
		t.Fatalf("expected duplicate email error, got %v", err)
	}
	leads, err := store.ListLeads(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(leads) != 1 || !reflect.DeepEqual(leads[0], original) {
		t.Fatalf("duplicate changed persisted leads: %+v", leads)
	}
}

func TestSQLiteEnforcesCaseInsensitiveEmailUniqueness(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "leads.db")
	store := openStore(t, path)
	if _, err := store.CreateLead(ctx, input()); err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, err = conn.ExecContext(ctx, `
		INSERT INTO leads (first_name, last_name, email, finance_amount, state, created_at)
		VALUES ('Other', 'Person', 'ZOE@EXAMPLE.COM', 1, 'NSW', ?)`, time.Now().UTC())
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		t.Fatalf("expected SQLite uniqueness constraint, got %v", err)
	}
}

func TestCreationValidatesBeforeInserting(t *testing.T) {
	store := openStore(t, ":memory:")
	invalid := input()
	invalid.FinanceAmount = 0
	_, err := store.CreateLead(context.Background(), invalid)
	var fields lead.FieldErrors
	if !errors.As(err, &fields) || fields["finance_amount"] != "must be greater than 0" {
		t.Fatalf("expected field error, got %v", err)
	}
	leads, err := store.ListLeads(context.Background())
	if err != nil || len(leads) != 0 {
		t.Fatalf("invalid lead was persisted: %+v, error %v", leads, err)
	}
}

func TestDatabaseErrorsAreNotValidationErrors(t *testing.T) {
	store := openStore(t, ":memory:")
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := store.CreateLead(context.Background(), input())
	var fields lead.FieldErrors
	if err == nil || errors.As(err, &fields) {
		t.Fatalf("expected operational error, got %v", err)
	}
}

func TestOpenFailure(t *testing.T) {
	_, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "missing", "leads.db"))
	if err == nil {
		t.Fatal("expected database initialization to fail")
	}
}
