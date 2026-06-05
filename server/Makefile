.PHONY: all build build-docker build-linux-amd64 build-linux-arm64 build-linux-mipsle test test-short lint tidy clean

BIN_DIR := bin
LDFLAGS := -ldflags="-s -w"

all: build test

# Default: native runtime (без Docker)
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-linux .

# Сборка с Docker runtime
build-docker:
	@mkdir -p $(BIN_DIR)
	go build -tags docker $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-docker-linux .

# Кросс-компиляция для OpenWrt/Linux (без Docker)
build-linux-mipsle:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
		go build $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-mipsle .

build-linux-arm64:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 \
		go build $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-arm64 .

build-linux-amd64:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 \
		go build $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-amd64 .

test:
	go test ./... -v

test-short:
	go test ./... -short

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
