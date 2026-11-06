package lead

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func pointer[T any](value T) *T { return &value }

func validInput() Input {
	return Input{
		FirstName: "Zoë", LastName: "Martin", Email: "zoe.martin@example.com",
		FinanceAmount: 38500.5, State: "QLD",
	}
}

func TestNormalize(t *testing.T) {
	input := Input{
		FirstName: " Zoë ", LastName: " Smith, Jr\t", Email: " ZOE@Example.COM ",
		FinanceAmount: 100, State: " victoria ",
		Vehicle: &Vehicle{Make: " Kia ", Model: pointer(" EV6 ")},
	}
	input.Normalize()
	if input.FirstName != "Zoë" || input.LastName != "Smith, Jr" || input.Email != "zoe@example.com" || input.State != "VIC" {
		t.Fatalf("unexpected normalization: %+v", input)
	}
	if input.Vehicle.Make != "Kia" || *input.Vehicle.Model != "EV6" {
		t.Fatalf("unexpected vehicle normalization: %+v", input.Vehicle)
	}
	if fields := input.Validate(); len(fields) != 0 {
		t.Fatal(fields)
	}

	input.Vehicle = &Vehicle{Make: " ", Model: pointer("\t")}
	input.Normalize()
	if input.Vehicle != nil {
		t.Fatal("an entirely empty vehicle should normalize to nil")
	}
	input.Vehicle = &Vehicle{Make: "Kia", Model: pointer(" ")}
	input.Normalize()
	if input.Vehicle == nil || input.Vehicle.Model != nil {
		t.Fatal("a make-only vehicle should retain the make and a nil model")
	}
}

func TestNormalizeStates(t *testing.T) {
	for name, code := range states {
		t.Run(name, func(t *testing.T) {
			input := validInput()
			input.State = " " + strings.ToLower(name) + " "
			input.Normalize()
			if input.State != code {
				t.Fatalf("got %q, want %q", input.State, code)
			}
			if fields := input.Validate(); len(fields) > 0 {
				t.Fatal(fields)
			}
		})
	}
}

func TestLeadJSONContract(t *testing.T) {
	createdAt := time.Date(2026, 11, 3, 10, 15, 0, 0, time.FixedZone("AEDT", 11*60*60))
	value := Lead{ID: 12, Input: validInput(), CreatedAt: createdAt}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":12,"first_name":"Zoë","last_name":"Martin","email":"zoe.martin@example.com","finance_amount":38500.5,"salary":null,"state":"QLD","vehicle":null,"created_at":"2026-11-03T10:15:00+11:00"}`
	if string(data) != want {
		t.Fatalf("got %s, want %s", data, want)
	}
	var decoded Lead
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.CreatedAt.Equal(createdAt) || !reflect.DeepEqual(decoded.Input, value.Input) {
		t.Fatalf("JSON round trip changed the lead: %+v", decoded)
	}
}

func TestInputJSONContract(t *testing.T) {
	payload := `{"first_name":"Zoë","last_name":"Martin","email":"zoe.martin@example.com","finance_amount":38500.5,"salary":null,"state":"QLD","vehicle":{"make":"Kia","model":"EV6","year":null}}`
	var input Input
	if err := json.Unmarshal([]byte(payload), &input); err != nil {
		t.Fatal(err)
	}
	input.Normalize()
	if fields := input.Validate(); len(fields) > 0 {
		t.Fatal(fields)
	}
	if input.Salary != nil || input.Vehicle == nil || input.Vehicle.Year != nil || *input.Vehicle.Model != "EV6" {
		t.Fatalf("unexpected optional fields: %+v", input)
	}
	data, err := json.Marshal(input)
	if err != nil || string(data) != payload {
		t.Fatalf("request round trip: %s, error: %v", data, err)
	}
}
