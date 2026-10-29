BINARY  := lazynote
CMD_PKG := ./cmd/lazynote
VERSION := $(shell cat VERSION)
LDFLAGS := -X github.com/tkilb/lazynote/internal/version.Version=$(VERSION)

.PHONY: build test clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_PKG)

test:
	go test ./...

clean:
	rm -f $(BINARY)
