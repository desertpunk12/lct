package csvimport

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"leadcapture/api/internal/db"
	"leadcapture/api/internal/lead"
)

type Diagnostic struct {
	Line int
	Err  error
}

type Result struct {
	Imported int
	Skipped  []Diagnostic
}

// Import commits valid rows independently. Bad row data and duplicate emails
// are diagnostics; structural CSV, I/O, and database failures stop the import.
// Results include any rows committed before a fatal error.
func Import(ctx context.Context, source io.Reader, store *db.Store) (Result, error) {
	result := Result{}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	reader := csv.NewReader(source)
	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return result, errors.New("CSV header is required")
	}
	if err != nil {
		return result, fmt.Errorf("read CSV header: %w", err)
	}
	columns, err := mapColumns(header)
	if err != nil {
		return result, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return result, nil
		}
		if err != nil {
			var parseErr *csv.ParseError
			if errors.Is(err, csv.ErrFieldCount) && errors.As(err, &parseErr) {
				result.Skipped = append(result.Skipped, Diagnostic{
					Line: parseErr.StartLine,
					Err:  fmt.Errorf("expected %d fields, got %d", len(header), len(record)),
				})
				continue
			}
			// Broken quoting can consume subsequent lines, so don't guess where
			// the next record starts or silently discard the remaining file.
			return result, fmt.Errorf("read CSV: %w", err)
		}

		line, _ := reader.FieldPos(0)
		input, fields := parseRow(record, columns)
		if len(fields) > 0 {
			result.Skipped = append(result.Skipped, Diagnostic{Line: line, Err: fields})
			continue
		}
		if _, err := store.CreateLead(ctx, input); err != nil {
			var fields lead.FieldErrors
			if errors.As(err, &fields) {
				result.Skipped = append(result.Skipped, Diagnostic{Line: line, Err: fields})
				continue
			}
			return result, fmt.Errorf("line %d: %w", line, err)
		}
		result.Imported++
	}
}

func mapColumns(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	for index, name := range header {
		if index == 0 {
			name = strings.TrimPrefix(name, "\uFEFF")
		}
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			return nil, fmt.Errorf("CSV header column %d is empty", index+1)
		}
		if _, exists := columns[name]; exists {
			return nil, fmt.Errorf("duplicate CSV header %q", name)
		}
		columns[name] = index
	}
	for _, name := range []string{"first_name", "last_name", "email", "finance_amount", "state"} {
		if _, exists := columns[name]; !exists {
			return nil, fmt.Errorf("missing required CSV header %q", name)
		}
	}
	return columns, nil
}

func parseRow(record []string, columns map[string]int) (lead.Input, lead.FieldErrors) {
	value := func(name string) string {
		if index, exists := columns[name]; exists {
			return strings.TrimSpace(record[index])
		}
		return ""
	}
	input := lead.Input{
		FirstName: value("first_name"),
		LastName:  value("last_name"),
		Email:     value("email"),
		State:     value("state"),
		Vehicle:   &lead.Vehicle{Make: value("vehicle_make")},
	}
	parseErrors := lead.FieldErrors{}
	if raw := value("finance_amount"); raw == "" {
		parseErrors["finance_amount"] = "is required"
	} else if amount, err := parseAmount(raw); err != nil {
		parseErrors["finance_amount"] = "must be a number"
	} else {
		input.FinanceAmount = amount
	}
	if raw := value("salary"); raw != "" {
		if amount, err := parseAmount(raw); err != nil {
			parseErrors["salary"] = "must be a number"
		} else {
			input.Salary = &amount
		}
	}
	if model := value("vehicle_model"); model != "" {
		input.Vehicle.Model = &model
	}
	if raw := value("vehicle_year"); raw != "" {
		if year, err := strconv.Atoi(raw); err != nil {
			parseErrors["vehicle.year"] = "must be a whole number"
			if input.Vehicle.Make == "" {
				parseErrors["vehicle.make"] = "is required when a model or year is given"
			}
		} else {
			input.Vehicle.Year = &year
		}
	}

	input.Normalize()
	fields := input.Validate()
	for field, message := range parseErrors {
		fields[field] = message
	}
	return input, fields
}

var groupedAmount = regexp.MustCompile(`^[+-]?[0-9]{1,3}(,[0-9]{3})+(\.[0-9]+)?$`)

func parseAmount(raw string) (float64, error) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "$"))
	if strings.Contains(raw, ",") && !groupedAmount.MatchString(raw) {
		return 0, errors.New("invalid thousands grouping")
	}
	return strconv.ParseFloat(strings.ReplaceAll(raw, ",", ""), 64)
}
