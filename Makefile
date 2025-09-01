.DEFAULT_GOAL := build

.PHONY: fmt vet build clean staticcheck tidy test test-cover run

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build

clean:
	go clean

staticcheck: fmt vet
	staticcheck ./...

tidy:
	go mod tidy

test:
	go test -v ./...

test-cover:
	go test -v -cover ./...

run:
	go run main.go