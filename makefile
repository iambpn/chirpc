build:
	go build -o build/example cmd/example/main.go

clean:
	go clean && rm -f build/

run:
	go run cmd/main.go

test:
	go test ./...

.PHONY: run clean build test