GOBIN := $(shell go env GOBIN)
ifeq ($(strip $(GOBIN)),)
GOBIN := $(shell go env GOPATH)/bin
endif

CLI_NAME := gin-ninja-cli
CLI_BUILD_DIR := $(CURDIR)/bin
CLI_BUILD_PATH := $(CLI_BUILD_DIR)/$(CLI_NAME)
CLI_INSTALL_PATH := $(GOBIN)/$(CLI_NAME)
GO_MODULES := . settings pkg/logger middleware orm admin bootstrap cache/redis filter order examples cmd/gin-ninja-cli examples/full/browsertests

.PHONY: build-cli install-cli test

build-cli:
	mkdir -p $(CLI_BUILD_DIR)
	cd ./cmd/gin-ninja-cli && go build -o $(CLI_BUILD_PATH) .

install-cli:
	mkdir -p $(GOBIN)
	cd ./cmd/gin-ninja-cli && go build -o $(CLI_INSTALL_PATH) .

test:
	@for module in $(GO_MODULES); do \
		echo "== $$module =="; \
		(cd $$module && GOWORK=off go test ./...) || exit 1; \
	done
