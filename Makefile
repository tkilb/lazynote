BINARY  := lazynote
INSTALL_DIR := $(HOME)/bin
CMD_PKG := ./cmd/lazynote
VERSION := $(shell cat VERSION)
LDFLAGS := -X github.com/tkilb/lazynote/internal/version.Version=$(VERSION)

.PHONY: build install test clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_PKG)

## install builds the binary and copies it to $(INSTALL_DIR) (must be on PATH).
install: build
	mkdir -p $(INSTALL_DIR)
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(INSTALL_DIR)/$(BINARY)"

test:
	go test ./...

clean:
	rm -f $(BINARY)
