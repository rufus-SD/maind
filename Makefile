VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/rufus-SD/maind/internal/cli.Version=$(VERSION)"
BINARY  := bin/maind

.PHONY: build install clean test

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

clean:
	rm -rf bin/

test:
	go test ./... -v

lint:
	go vet ./...
