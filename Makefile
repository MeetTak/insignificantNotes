# notesync — common tasks. Run `make help` for the list.
BIN         := bin/notesync
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: help build test vet check scan export site serve publish dry-run hooks install clean

help: ## list targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

build: ## compile the notesync binary into bin/
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN) ./cmd/notesync

test: ## run unit tests + the content gate (secrets, excluded folders, links)
	go test ./...

vet: ## static checks
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

check: vet test build ## everything CI runs, locally
	./$(BIN) scan

scan: build ## secret-scan the working tree
	./$(BIN) scan

export: build ## pull notes from Apple Notes into content/
	./$(BIN) export

site: build ## render content/ into site/
	./$(BIN) build

serve: build ## preview at http://localhost:8080 with live reload
	./$(BIN) serve

dry-run: build ## export (no writes) + scan + build, no git
	./$(BIN) publish --dry-run

publish: build ## export → scan → build → commit → push
	./$(BIN) publish

hooks: build ## install git pre-commit / pre-push hooks
	./$(BIN) init

install: build ## copy the binary onto your PATH ($(INSTALL_DIR))
	mkdir -p $(INSTALL_DIR) && install -m 0755 $(BIN) $(INSTALL_DIR)/notesync
	@echo "installed $(INSTALL_DIR)/notesync"

clean: ## remove build output (bin/ and site/ only; never content/ or Apple Notes)
	rm -rf bin site
