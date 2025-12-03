build:
	go build -o build/example cmd/example/main.go

clean:
	go clean && rm -f build/

run:
	go run cmd/example/main.go

test:
	go test ./... --cover

test-v:
	go test ./... --cover -v

.PHONY: run clean build test