build:
	go build -o build/example ./cmd/example/server

clean:
	go clean && rm -rf build/

run-server:
	go run cmd/example/server/main.go

test:
	go test ./... --cover --coverprofile=coverage.out

test-v:
	go test ./... --cover -v

html-coverage:
	go tool cover -html=coverage.out -o coverage.html

.PHONY: build clean run-server test test-v html-coverage