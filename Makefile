PKG := ./cmd/vo
VERSION_PKG := voyage/internal/cli
OUT_DIR := dist
GOOS_LOCAL := $(shell go env GOOS)
GOARCH_LOCAL := $(shell go env GOARCH)
PLATFORM_LOCAL := $(GOOS_LOCAL)-$(GOARCH_LOCAL)
BIN_LOCAL := $(OUT_DIR)/vo-$(PLATFORM_LOCAL)
BIN_LINUX_AMD64 := $(OUT_DIR)/vo-linux-amd64
BIN_DARWIN_AMD64 := $(OUT_DIR)/vo-darwin-amd64
BIN_DARWIN_ARM64 := $(OUT_DIR)/vo-darwin-arm64
TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
HASH := $(shell git rev-parse --short HEAD 2>/dev/null)
VERSION := $(shell \
	if [ -n "$(TAG)" ]; then \
		if git describe --tags --exact-match >/dev/null 2>&1; then \
			echo "$(TAG)"; \
		elif [ -n "$(HASH)" ]; then \
			echo "$(TAG)-$(HASH)"; \
		else \
			echo "$(TAG)"; \
		fi; \
	else \
		echo "dev"; \
	fi)
LDFLAGS := -X '$(VERSION_PKG).Version=$(VERSION)'

.PHONY: build build-all build-linux-amd64 build-darwin-amd64 build-darwin-arm64 test version clean

build:
	mkdir -p $(OUT_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_LOCAL) $(PKG)

build-linux-amd64:
	mkdir -p $(OUT_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_LINUX_AMD64) $(PKG)

build-darwin-amd64:
	mkdir -p $(OUT_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DARWIN_AMD64) $(PKG)

build-darwin-arm64:
	mkdir -p $(OUT_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DARWIN_ARM64) $(PKG)

build-all: build-linux-amd64 build-darwin-amd64 build-darwin-arm64

test:
	go test ./...

version:
	@echo $(VERSION)

clean:
	rm -rf $(OUT_DIR)
