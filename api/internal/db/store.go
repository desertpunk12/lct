package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"leadcapture/api/internal/lead"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

//go:embed schema.sql
var schema string

type Store struct {
	conn    *sql.DB
	queries *Queries
}

func Open(ctx context.Context, path string) (*Store, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// SQLite has one writer; one connection also keeps :memory: databases intact.
	conn.SetMaxOpenConns(1)
	if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout = 5000;\n"+schema); err != nil {
		return nil, errors.Join(fmt.Errorf("initialize database: %w", err), conn.Close())
	}
	return &Store{conn: conn, queries: New(conn)}, nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) CreateLead(ctx context.Context, input lead.Input) (lead.Lead, error) {
	input.Normalize()
	if fields := input.Validate(); len(fields) > 0 {
		return lead.Lead{}, fields
	}

	params := InsertLeadParams{
		FirstName:     input.FirstName,
		LastName:      input.LastName,
		Email:         input.Email,
		FinanceAmount: input.FinanceAmount,
		State:         input.State,
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
	}
	if input.Salary != nil {
		params.Salary = sql.NullFloat64{Float64: *input.Salary, Valid: true}
	}
	if input.Vehicle != nil {
		params.VehicleMake = sql.NullString{String: input.Vehicle.Make, Valid: true}
		if input.Vehicle.Model != nil {
			params.VehicleModel = sql.NullString{String: *input.Vehicle.Model, Valid: true}
		}
		if input.Vehicle.Year != nil {
			params.VehicleYear = sql.NullInt64{Int64: int64(*input.Vehicle.Year), Valid: true}
		}
	}

	row, err := s.queries.InsertLead(ctx, params)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return lead.Lead{}, lead.FieldErrors{"email": "duplicate email"}
		}
		return lead.Lead{}, fmt.Errorf("insert lead: %w", err)
	}
	return toLead(row), nil
}

func (s *Store) ListLeads(ctx context.Context) ([]lead.Lead, error) {
	rows, err := s.queries.ListLeads(ctx)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	leads := make([]lead.Lead, 0, len(rows))
	for _, row := range rows {
		leads = append(leads, toLead(row))
	}
	return leads, nil
}

func toLead(row Lead) lead.Lead {
	result := lead.Lead{
		ID: row.ID,
		Input: lead.Input{
			FirstName:     row.FirstName,
			LastName:      row.LastName,
			Email:         row.Email,
			FinanceAmount: row.FinanceAmount,
			State:         row.State,
		},
		CreatedAt: row.CreatedAt,
	}
	if row.Salary.Valid {
		result.Salary = &row.Salary.Float64
	}
	if row.VehicleMake.Valid {
		result.Vehicle = &lead.Vehicle{Make: row.VehicleMake.String}
		if row.VehicleModel.Valid {
			result.Vehicle.Model = &row.VehicleModel.String
		}
		if row.VehicleYear.Valid {
			year := int(row.VehicleYear.Int64)
			result.Vehicle.Year = &year
		}
	}
	return result
}
