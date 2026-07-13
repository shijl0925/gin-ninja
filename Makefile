GOBIN := $(shell go env GOBIN)
ifeq ($(strip $(GOBIN)),)
GOBIN := $(shell go env GOPATH)/bin
endif

CLI_NAME := gin-ninja-cli
CLI_BUILD_DIR := $(CURDIR)/bin
CLI_BUILD_PATH := $(CLI_BUILD_DIR)/$(CLI_NAME)
CLI_INSTALL_PATH := $(GOBIN)/$(CLI_NAME)
STATICCHECK := $(GOBIN)/staticcheck
STATICCHECK_VERSION ?= latest

.PHONY: build build-cli ci install-cli staticcheck staticcheck-install test test-browser validate

build-cli:
	mkdir -p $(CLI_BUILD_DIR)
	cd ./cmd/gin-ninja-cli && go build -o $(CLI_BUILD_PATH) .

build:
	go build ./...
	cd ./cmd/gin-ninja-cli && go build ./...

install-cli:
	mkdir -p $(GOBIN)
	cd ./cmd/gin-ninja-cli && go build -o $(CLI_INSTALL_PATH) .

test:
	go test ./...
	cd ./cmd/gin-ninja-cli && go test ./...

test-browser:
	cd ./examples/full/browsertests && go test ./...

staticcheck-install:
	mkdir -p $(GOBIN)
	GOTOOLCHAIN=go1.26.0 go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)

staticcheck: staticcheck-install
	$(STATICCHECK) ./...
	cd ./cmd/gin-ninja-cli && $(STATICCHECK) ./...
	cd ./examples/full/browsertests && $(STATICCHECK) ./...

validate: staticcheck test test-browser build

ci: validate
