.PHONY: build test race vet fmt lint lint-fix install clean

build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

install:
	go install ./cmd/tnow

clean:
	rm -f tnow