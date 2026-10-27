.PHONY: build test race vet fmt install clean

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

install:
	go install ./cmd/tnow

clean:
	rm -f tnow