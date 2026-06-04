.PHONY: all build build-openwrt-mips build-openwrt-arm64 build-openwrt-amd64 test test-short lint tidy clean

BIN_DIR := bin
LDFLAGS := -ldflags="-s -w"

all: build test

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-linux .

build-openwrt-mips:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
		go build -tags openwrt $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-mipsle .

build-openwrt-arm64:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 \
		go build -tags openwrt $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-arm64 .

build-openwrt-amd64:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 \
		go build -tags openwrt $(LDFLAGS) -o $(BIN_DIR)/singbox-ui-amd64 .

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
