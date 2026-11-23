package services

import (
	"encoding/csv"
	"errors"
	"strings"
	"testing"
)

func TestJSONToCSV(t *testing.T) {
	input := `[{"name":"Ada","id":1,"tags":["go"]},{"id":2,"name":"Linus","active":true}]`
	data, err := JSONToCSV([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := []string{"active", "id", "name", "tags"}
	for i, value := range wantHeader {
		if rows[0][i] != value {
			t.Fatalf("header[%d] = %q, want %q", i, rows[0][i], value)
		}
	}
	if rows[1][3] != `["go"]` {
		t.Fatalf("nested value = %q", rows[1][3])
	}
	if rows[2][0] != "true" {
		t.Fatalf("boolean value = %q", rows[2][0])
	}
}

func TestJSONToCSVWrappedResults(t *testing.T) {
	data, err := JSONToCSV([]byte(`{"results":[{"id":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "id\n1\n" {
		t.Fatalf("CSV = %q", data)
	}
}

func TestJSONToCSVRejectsInvalidShape(t *testing.T) {
	_, err := JSONToCSV([]byte(`["not-an-object"]`))
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestJSONToCSVRejectsTrailingJSON(t *testing.T) {
	_, err := JSONToCSV([]byte(`[{"id":1}] trailing`))
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestJSONToCSVRejectsEmptyResults(t *testing.T) {
	_, err := JSONToCSV([]byte(`[]`))
	if !errors.Is(err, ErrNoResults) {
		t.Fatalf("error = %v, want ErrNoResults", err)
	}
}
