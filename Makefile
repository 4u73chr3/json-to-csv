.PHONY: run test vet build clean

BINARY := json-to-csv

run:
	go run .

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)
