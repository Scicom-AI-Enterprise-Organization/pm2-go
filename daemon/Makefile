SHELL := /bin/bash
BINARY := pm2-go
PREFIX ?= $(HOME)/.local
BIN_DIR := $(PREFIX)/bin
GO ?= $(HOME)/go-toolchain/go/bin/go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: all build install test test-race test-cover lint tidy clean run

all: build

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/pm2-go

install: build
	@mkdir -p $(BIN_DIR)
	install -m 0755 bin/$(BINARY) $(BIN_DIR)/$(BINARY)
	@echo "Installed to $(BIN_DIR)/$(BINARY)"

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-cover:
	$(GO) test -cover ./...

lint:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin/

run: build
	./bin/$(BINARY) $(ARGS)
