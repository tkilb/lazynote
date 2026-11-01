BINARY  := lazynote
INSTALL_DIR := $(HOME)/bin
CMD_PKG := ./cmd/lazynote
VERSION := $(shell cat VERSION)
LDFLAGS := -X github.com/tkilb/lazynote/internal/version.Version=$(VERSION)

.PHONY: build install test clean release minor major patch

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

## release bumps VERSION and cuts a release. Defaults to a patch bump;
## pass `minor`/`major` as an extra goal to bump that part instead, e.g.
## `make release minor`. Each commits just the VERSION file, tags the
## commit v<version>, and pushes both the branch and tag to origin, which
## triggers .github/workflows/release.yml. It asks first. See
## scripts/release.sh.
BUMP_TYPE := patch
ifneq (,$(filter minor,$(MAKECMDGOALS)))
BUMP_TYPE := minor
endif
ifneq (,$(filter major,$(MAKECMDGOALS)))
BUMP_TYPE := major
endif

release:
	@./scripts/release.sh $(BUMP_TYPE)

# no-op targets so `make release minor` / `make release major` /
# `make release patch` do not fail with "no rule to make target"
minor major patch:
	@:
