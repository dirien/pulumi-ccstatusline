.PHONY: build test lint fmt clean install snapshot

BINARY_NAME=pulumi-ccstatusline
BUILD_DIR=bin
GO=go

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	$(GO) test -v -race -cover ./...

lint:
	golangci-lint run ./...

fmt:
	golangci-lint run --fix ./...

clean:
	rm -rf $(BUILD_DIR) dist

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)

snapshot:
	goreleaser release --snapshot --clean
