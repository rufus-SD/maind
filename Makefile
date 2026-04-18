VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/rufus-SD/maind/internal/cli.Version=$(VERSION)"
BINARY  := bin/maind
DIST    := dist

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: build install clean test lint release

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

clean:
	rm -rf bin/ $(DIST)/

test:
	go test ./... -v

lint:
	go vet ./...

release: clean
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		name=maind-$(VERSION)-$${os}-$${arch}; \
		echo "Building $${os}/$${arch}..."; \
		GOOS=$${os} GOARCH=$${arch} go build $(LDFLAGS) -o $(DIST)/$${name}/maind . ; \
		tar -czf $(DIST)/$${name}.tar.gz -C $(DIST) $${name}/ ; \
		rm -rf $(DIST)/$${name} ; \
	done
	@cd $(DIST) && shasum -a 256 *.tar.gz > checksums.txt
	@echo "Release artifacts in $(DIST)/"
