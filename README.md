# JSON to CSV

A small Go web application that accepts a parameter, fetches JSON, converts the
results to CSV, and sends the CSV to the browser as a download.

## Run

Requires Go 1.27 or later.

```bash
go run .
```

Open <http://localhost:8080>. By default the server uses mock JSON, so it can be
tested without an external dependency.

## Use a real API

Set `API_URL` to the endpoint that should receive the form value. The configured
parameter name is `q` by default.

```bash
API_URL="https://example.com/api/search" \
API_PARAMETER="q" \
API_TIMEOUT="10s" \
PORT="8080" \
go run .
```

The API client preserves query parameters already present in `API_URL`. The
response may be an array of objects, one object, or an object containing an array
under `data`, `results`, or `items`. Nested values are encoded as JSON in the CSV.
Columns are the sorted union of all object keys.

## Test

```bash
go test ./...
```
# json-to-csv
