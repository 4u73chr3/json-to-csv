package services

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
)

var ErrNoResults = errors.New("API response contains no results")

func JSONToCSV(input []byte) ([]byte, error) {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode JSON: multiple JSON values are not allowed")
		}
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	rows, err := extractRows(root)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNoResults
	}

	columnSet := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			columnSet[key] = struct{}{}
		}
	}
	columns := make([]string, 0, len(columnSet))
	for key := range columnSet {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	if len(columns) == 0 {
		return nil, ErrNoResults
	}

	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write(columns); err != nil {
		return nil, fmt.Errorf("write CSV header: %w", err)
	}
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, column := range columns {
			record[i], err = csvValue(row[column])
			if err != nil {
				return nil, fmt.Errorf("format field %q: %w", column, err)
			}
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("write CSV row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush CSV: %w", err)
	}
	return output.Bytes(), nil
}

func extractRows(root any) ([]map[string]any, error) {
	switch value := root.(type) {
	case []any:
		return rowsFromArray(value)
	case map[string]any:
		for _, key := range []string{"data", "results", "items"} {
			if nested, ok := value[key]; ok {
				if array, ok := nested.([]any); ok {
					return rowsFromArray(array)
				}
			}
		}
		return []map[string]any{value}, nil
	default:
		return nil, fmt.Errorf("JSON root must be an array or object")
	}
}

func rowsFromArray(values []any) ([]map[string]any, error) {
	rows := make([]map[string]any, 0, len(values))
	for index, value := range values {
		row, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("JSON result %d must be an object", index)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func csvValue(value any) (string, error) {
	switch value := value.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	case json.Number:
		return value.String(), nil
	case bool:
		return fmt.Sprint(value), nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
}
