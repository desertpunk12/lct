-- name: InsertLead :one
INSERT INTO leads (
    first_name, last_name, email, finance_amount, salary, state,
    vehicle_make, vehicle_model, vehicle_year, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListLeads :many
SELECT * FROM leads ORDER BY id;
