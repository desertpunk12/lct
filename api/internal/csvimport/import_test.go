package csvimport_test

import (
	"context"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"leadcapture/api/internal/csvimport"
	"leadcapture/api/internal/db"
	"leadcapture/api/internal/lead"
)

const header = "first_name,last_name,email,finance_amount,salary,state,vehicle_make,vehicle_model,vehicle_year\n"
const requiredHeader = "first_name,last_name,email,finance_amount,state\n"

func openStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.Open(context.Background(), ":memory:")
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

func TestSampleFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "leads.csv"))
	if err != nil {
		t.Fatal(err)
	}
	store := openStore(t)
	result, err := csvimport.Import(context.Background(), strings.NewReader(string(data)), store)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 36 || len(result.Skipped) != 4 {
		t.Fatalf("unexpected result: %+v", result)
	}
	wantLines := []int{25, 29, 33, 37}
	wantErrors := []string{
		"state: must be a valid Australian state or territory",
		"email: is required",
		"vehicle.make: is required when a model or year is given",
		"email: duplicate email",
	}
	for i, diagnostic := range result.Skipped {
		if diagnostic.Line != wantLines[i] || diagnostic.Err.Error() != wantErrors[i] {
			t.Errorf("diagnostic %d: got %+v, want line %d: %s", i, diagnostic, wantLines[i], wantErrors[i])
		}
	}
	leads, err := store.ListLeads(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byEmail := make(map[string]lead.Lead, len(leads))
	for _, item := range leads {
		byEmail[item.Email] = item
	}
	liam := byEmail["liam.obrien@example.com"]
	if liam.FinanceAmount != 45000 || liam.State != "VIC" {
		t.Fatalf("currency/lowercase state not normalized: %+v", liam)
	}
	daniel := byEmail["daniel.smith@example.com"]
	if daniel.LastName != "Smith, Jr" || daniel.State != "VIC" || daniel.Vehicle.Year != nil {
		t.Fatalf("quoted name/full state/optional year not preserved: %+v", daniel)
	}
	zoe := byEmail["zoe.martin@example.com"]
	if zoe.FirstName != "Zoë" || zoe.FinanceAmount != 38500.5 || zoe.Salary != nil || zoe.Vehicle != nil {
		t.Fatalf("unicode/decimals/nulls not preserved: %+v", zoe)
	}
	if byEmail["aisha.khan@example.com"].Vehicle.Model != nil {
		t.Fatal("missing model should be nil")
	}

	again, err := csvimport.Import(context.Background(), strings.NewReader(string(data)), store)
	if err != nil || again.Imported != 0 || len(again.Skipped) != 40 {
		t.Fatalf("reimport must not insert duplicates: %+v, error %v", again, err)
	}
	if again.Skipped[0].Err.Error() != "email: duplicate email" {
		t.Fatalf("existing email was not reported as duplicate: %+v", again.Skipped[0])
	}
}

func TestHeaderMapping(t *testing.T) {
	// Optional columns may be absent; unrelated columns don't change the mapping.
	data := "\uFEFFEMAIL, State ,finance_amount,last_name,first_name,partner_id\r\n" +
		" ZOE@Example.com , victoria ,\"$45,000.50\", Martin , Zoë ,123\r\n"
	store := openStore(t)
	result, err := csvimport.Import(context.Background(), strings.NewReader(data), store)
	if err != nil || result.Imported != 1 || len(result.Skipped) != 0 {
		t.Fatalf("unexpected result: %+v, error %v", result, err)
	}
	leads, err := store.ListLeads(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := leads[0]
	if got.Email != "zoe@example.com" || got.State != "VIC" || got.FinanceAmount != 45000.5 || got.Vehicle != nil || got.Salary != nil {
		t.Fatalf("header mapping changed values: %+v", got)
	}
}

func TestPhysicalLineNumbersAndFieldCounts(t *testing.T) {
	data := requiredHeader + "\n" +
		"\"Ana\nMaria\",Smith,,1,NSW\n" +
		"too,few\n" +
		"A,B,good@example.com,1,NSW\n" +
		"A,B,extra@example.com,1,NSW,extra\n"
	result, err := csvimport.Import(context.Background(), strings.NewReader(data), openStore(t))
	if err != nil || result.Imported != 1 || len(result.Skipped) != 3 {
		t.Fatalf("unexpected result: %+v, error %v", result, err)
	}
	wantLines := []int{3, 5, 7}
	wantErrors := []string{"email: is required", "expected 5 fields, got 2", "expected 5 fields, got 6"}
	for i, diagnostic := range result.Skipped {
		if diagnostic.Line != wantLines[i] || diagnostic.Err.Error() != wantErrors[i] {
			t.Errorf("diagnostic %d: got %+v, want line %d: %s", i, diagnostic, wantLines[i], wantErrors[i])
		}
	}
}

func TestInvalidHeaders(t *testing.T) {
	cases := []struct{ name, csv, message string }{
		{"empty", "", "CSV header is required"},
		{"missing required", "first_name,last_name,email,state\n", "missing required CSV header \"finance_amount\""},
		{"duplicate", requiredHeader[:len(requiredHeader)-1] + ",EMAIL\n", "duplicate CSV header \"email\""},
		{"empty column", requiredHeader[:len(requiredHeader)-1] + ",\n", "CSV header column 6 is empty"},
		{"broken quoting", "\"first_name,last_name,email,finance_amount,state\n", "read CSV header:"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := csvimport.Import(context.Background(), strings.NewReader(test.csv), openStore(t))
			if err == nil || !strings.Contains(err.Error(), test.message) || result.Imported != 0 {
				t.Fatalf("got %+v, error %v; want %s", result, err, test.message)
			}
		})
	}
}

func TestHeaderOnly(t *testing.T) {
	result, err := csvimport.Import(context.Background(), strings.NewReader(header), openStore(t))
	if err != nil || result.Imported != 0 || len(result.Skipped) != 0 {
		t.Fatalf("unexpected result: %+v, error %v", result, err)
	}
}

func TestRowParsingErrors(t *testing.T) {
	cases := []struct {
		name   string
		column int
		value  string
		want   lead.FieldErrors
	}{
		{"required finance", 3, " ", lead.FieldErrors{"finance_amount": "is required"}},
		{"invalid finance", 3, "oops", lead.FieldErrors{"finance_amount": "must be a number"}},
		{"invalid grouping", 3, "$1,00", lead.FieldErrors{"finance_amount": "must be a number"}},
		{"zero finance", 3, "0", lead.FieldErrors{"finance_amount": "must be greater than 0"}},
		{"negative currency", 3, "$-5.00", lead.FieldErrors{"finance_amount": "must be greater than 0"}},
		{"NaN finance", 3, "NaN", lead.FieldErrors{"finance_amount": "must be greater than 0"}},
		{"infinite finance", 3, "+Inf", lead.FieldErrors{"finance_amount": "must be greater than 0"}},
		{"overflow finance", 3, "1e999", lead.FieldErrors{"finance_amount": "must be a number"}},
		{"invalid salary", 4, "many", lead.FieldErrors{"salary": "must be a number"}},
		{"zero salary", 4, "0", lead.FieldErrors{"salary": "must be greater than 0"}},
		{"negative salary", 4, "-1", lead.FieldErrors{"salary": "must be greater than 0"}},
		{"infinite salary", 4, "Infinity", lead.FieldErrors{"salary": "must be greater than 0"}},
		{"invalid year without make", 8, "2024.5", lead.FieldErrors{
			"vehicle.year": "must be a whole number", "vehicle.make": "is required when a model or year is given",
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			row := []string{"A", "B", "a@example.com", "1", "", "NSW", "", "", ""}
			row[test.column] = test.value
			var data strings.Builder
			data.WriteString(header)
			writer := csv.NewWriter(&data)
			if err := writer.Write(row); err != nil {
				t.Fatal(err)
			}
			writer.Flush()
			if err := writer.Error(); err != nil {
				t.Fatal(err)
			}
			result, err := csvimport.Import(context.Background(), strings.NewReader(data.String()), openStore(t))
			if err != nil || result.Imported != 0 || len(result.Skipped) != 1 {
				t.Fatalf("unexpected result: %+v, error %v", result, err)
			}
			var fields lead.FieldErrors
			if !errors.As(result.Skipped[0].Err, &fields) || !reflect.DeepEqual(fields, test.want) {
				t.Fatalf("got %v, want %v", result.Skipped[0].Err, test.want)
			}
		})
	}
}

func TestMalformedCSVStopsButPreservesImportedRows(t *testing.T) {
	store := openStore(t)
	data := requiredHeader + "A,B,good@example.com,1,NSW\n\"unclosed,B,bad@example.com,1,NSW\n"
	result, err := csvimport.Import(context.Background(), strings.NewReader(data), store)
	if err == nil || !strings.Contains(err.Error(), "read CSV:") || result.Imported != 1 || len(result.Skipped) != 0 {
		t.Fatalf("unexpected result: %+v, error %v", result, err)
	}
	leads, err := store.ListLeads(context.Background())
	if err != nil || len(leads) != 1 {
		t.Fatalf("earlier row should remain committed: %+v, error %v", leads, err)
	}
}

func TestDatabaseFailureIsFatal(t *testing.T) {
	store := openStore(t)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	result, err := csvimport.Import(context.Background(), strings.NewReader(requiredHeader+"A,B,a@example.com,1,NSW\n"), store)
	if err == nil || result.Imported != 0 || len(result.Skipped) != 0 || !strings.Contains(err.Error(), "line 2:") {
		t.Fatalf("database errors must not be skipped: %+v, error %v", result, err)
	}
}

func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := csvimport.Import(ctx, strings.NewReader(header), openStore(t))
	if !errors.Is(err, context.Canceled) || result.Imported != 0 {
		t.Fatalf("unexpected result: %+v, error %v", result, err)
	}
}
