GOBIN := $(shell go env GOBIN)
ifeq ($(strip $(GOBIN)),)
GOBIN := $(shell go env GOPATH)/bin
endif

CLI_NAME := gin-ninja-cli
CLI_BUILD_DIR := $(CURDIR)/bin
CLI_BUILD_PATH := $(CLI_BUILD_DIR)/$(CLI_NAME)
CLI_INSTALL_PATH := $(GOBIN)/$(CLI_NAME)

.PHONY: build-cli install-cli

build-cli:
	mkdir -p $(CLI_BUILD_DIR)
	go build -o $(CLI_BUILD_PATH) ./cmd/gin-ninja

install-cli:
	mkdir -p $(GOBIN)
	go build -o $(CLI_INSTALL_PATH) ./cmd/gin-ninja
