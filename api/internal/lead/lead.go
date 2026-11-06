package lead

import (
	"strings"
	"time"
)

// Input is the shared creation payload for CSV imports and POST /api/leads.
type Input struct {
	FirstName     string   `json:"first_name"`
	LastName      string   `json:"last_name"`
	Email         string   `json:"email"`
	FinanceAmount float64  `json:"finance_amount"`
	Salary        *float64 `json:"salary"`
	State         string   `json:"state"`
	Vehicle       *Vehicle `json:"vehicle"`
}

type Vehicle struct {
	Make  string  `json:"make"`
	Model *string `json:"model"`
	Year  *int    `json:"year"`
}

type Lead struct {
	ID int64 `json:"id"`
	Input
	CreatedAt time.Time `json:"created_at"`
}

var states = map[string]string{
	"ACT":                          "ACT",
	"AUSTRALIAN CAPITAL TERRITORY": "ACT",
	"NSW":                          "NSW",
	"NEW SOUTH WALES":              "NSW",
	"NT":                           "NT",
	"NORTHERN TERRITORY":           "NT",
	"QLD":                          "QLD",
	"QUEENSLAND":                   "QLD",
	"SA":                           "SA",
	"SOUTH AUSTRALIA":              "SA",
	"TAS":                          "TAS",
	"TASMANIA":                     "TAS",
	"VIC":                          "VIC",
	"VICTORIA":                     "VIC",
	"WA":                           "WA",
	"WESTERN AUSTRALIA":            "WA",
}

func (in *Input) Normalize() {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.State = strings.ToUpper(strings.TrimSpace(in.State))
	if code, ok := states[in.State]; ok {
		in.State = code
	}

	if in.Vehicle != nil {
		vehicle := *in.Vehicle
		vehicle.Make = strings.TrimSpace(vehicle.Make)
		if vehicle.Model != nil {
			model := strings.TrimSpace(*vehicle.Model)
			vehicle.Model = nil
			if model != "" {
				vehicle.Model = &model
			}
		}
		in.Vehicle = &vehicle
		if vehicle.Make == "" && vehicle.Model == nil && vehicle.Year == nil {
			in.Vehicle = nil
		}
	}
}
