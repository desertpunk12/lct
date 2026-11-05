package lead

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	nextYear := time.Now().Year() + 1
	yearError := fmt.Sprintf("must be between 1990 and %d", nextYear)
	cases := []struct {
		name   string
		change func(*Input)
		want   FieldErrors
	}{
		{"minimal", func(in *Input) {}, nil},
		{"first name required", func(in *Input) { in.FirstName = " " }, FieldErrors{"first_name": "is required"}},
		{"last name required", func(in *Input) { in.LastName = "\t" }, FieldErrors{"last_name": "is required"}},
		{"email required", func(in *Input) { in.Email = " " }, FieldErrors{"email": "is required"}},
		{"state required", func(in *Input) { in.State = " " }, FieldErrors{"state": "is required"}},
		{"unknown state", func(in *Input) { in.State = "XX" }, FieldErrors{"state": "must be a valid Australian state or territory"}},
		{"zero finance", func(in *Input) { in.FinanceAmount = 0 }, FieldErrors{"finance_amount": "must be greater than 0"}},
		{"negative finance", func(in *Input) { in.FinanceAmount = -1 }, FieldErrors{"finance_amount": "must be greater than 0"}},
		{"NaN finance", func(in *Input) { in.FinanceAmount = math.NaN() }, FieldErrors{"finance_amount": "must be greater than 0"}},
		{"infinite finance", func(in *Input) { in.FinanceAmount = math.Inf(1) }, FieldErrors{"finance_amount": "must be greater than 0"}},
		{"positive salary", func(in *Input) { in.Salary = pointer(90000.5) }, nil},
		{"zero salary", func(in *Input) { in.Salary = pointer(0.0) }, FieldErrors{"salary": "must be greater than 0"}},
		{"negative salary", func(in *Input) { in.Salary = pointer(-1.0) }, FieldErrors{"salary": "must be greater than 0"}},
		{"NaN salary", func(in *Input) { in.Salary = pointer(math.NaN()) }, FieldErrors{"salary": "must be greater than 0"}},
		{"infinite salary", func(in *Input) { in.Salary = pointer(math.Inf(1)) }, FieldErrors{"salary": "must be greater than 0"}},
		{"empty vehicle", func(in *Input) { in.Vehicle = &Vehicle{} }, nil},
		{"make only", func(in *Input) { in.Vehicle = &Vehicle{Make: "Mazda"} }, nil},
		{"partial vehicle", func(in *Input) { in.Vehicle = &Vehicle{Make: "Kia", Model: pointer("EV6")} }, nil},
		{"model without make", func(in *Input) { in.Vehicle = &Vehicle{Model: pointer("Corolla")} }, FieldErrors{"vehicle.make": "is required when a model or year is given"}},
		{"year without make", func(in *Input) { in.Vehicle = &Vehicle{Year: pointer(2020)} }, FieldErrors{"vehicle.make": "is required when a model or year is given"}},
		{"minimum year", func(in *Input) { in.Vehicle = &Vehicle{Make: "Toyota", Year: pointer(1990)} }, nil},
		{"next year", func(in *Input) { in.Vehicle = &Vehicle{Make: "Toyota", Year: &nextYear} }, nil},
		{"too old", func(in *Input) { in.Vehicle = &Vehicle{Make: "Toyota", Year: pointer(1989)} }, FieldErrors{"vehicle.year": yearError}},
		{"too new", func(in *Input) { in.Vehicle = &Vehicle{Make: "Toyota", Year: pointer(nextYear + 1)} }, FieldErrors{"vehicle.year": yearError}},
		{"zero year is not absent", func(in *Input) { in.Vehicle = &Vehicle{Make: "Toyota", Year: pointer(0)} }, FieldErrors{"vehicle.year": yearError}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			input := validInput()
			test.change(&input)
			input.Normalize()
			got := input.Validate()
			if len(got) != len(test.want) || (len(got) > 0 && !reflect.DeepEqual(got, test.want)) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateReportsAllFields(t *testing.T) {
	got := (Input{}).Validate()
	want := FieldErrors{
		"first_name": "is required", "last_name": "is required", "email": "is required",
		"finance_amount": "must be greater than 0", "state": "is required",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFieldErrorsAreDeterministic(t *testing.T) {
	fields := FieldErrors{"state": "invalid", "email": "duplicate email"}
	if got := fields.Error(); got != "email: duplicate email; state: invalid" {
		t.Fatal(got)
	}
}
