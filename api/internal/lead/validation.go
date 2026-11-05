package lead

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// FieldErrors uses the same field names as the API's errors object.
type FieldErrors map[string]string

func (e FieldErrors) Error() string {
	fields := make([]string, 0, len(e))
	for field := range e {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	messages := make([]string, 0, len(fields))
	for _, field := range fields {
		messages = append(messages, field+": "+e[field])
	}
	return strings.Join(messages, "; ")
}

// Validate expects normalized input. Persistence normalizes and validates every
// creation so the importer and future HTTP handler cannot apply different rules.
func (in Input) Validate() FieldErrors {
	errors := FieldErrors{}
	for field, value := range map[string]string{
		"first_name": in.FirstName,
		"last_name":  in.LastName,
		"email":      in.Email,
		"state":      in.State,
	} {
		if value == "" {
			errors[field] = "is required"
		}
	}

	if !positiveFinite(in.FinanceAmount) {
		errors["finance_amount"] = "must be greater than 0"
	}
	if in.Salary != nil && !positiveFinite(*in.Salary) {
		errors["salary"] = "must be greater than 0"
	}
	if _, valid := states[in.State]; in.State != "" && !valid {
		errors["state"] = "must be a valid Australian state or territory"
	}

	if in.Vehicle != nil {
		if in.Vehicle.Make == "" && (in.Vehicle.Model != nil || in.Vehicle.Year != nil) {
			errors["vehicle.make"] = "is required when a model or year is given"
		}
		nextYear := time.Now().Year() + 1
		if in.Vehicle.Year != nil && (*in.Vehicle.Year < 1990 || *in.Vehicle.Year > nextYear) {
			errors["vehicle.year"] = fmt.Sprintf("must be between 1990 and %d", nextYear)
		}
	}
	return errors
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsInf(value, 0) && !math.IsNaN(value)
}
