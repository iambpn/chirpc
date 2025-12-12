build:
	go build -o build/example cmd/example/main.go

clean:
	go clean && rm -f build/

run-server:
	go run cmd/example/server/main.go

test:
	go test ./... --cover --coverprofile=coverage.out

test-v:
	go test ./... --cover -v

html-coverage:
	go tool cover -html=coverage.out -o coverage.html

.PHONY: run clean build test