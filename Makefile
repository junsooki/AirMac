.PHONY: all build-host build-controller run-host run-controller run-signaling clean

GO := go
BIN_DIR := bin
HOST_BIN := $(BIN_DIR)/airmac-host
CONTROLLER_BIN := $(BIN_DIR)/airmac-controller

all: build-host build-controller

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build-host: $(BIN_DIR)
	CGO_ENABLED=1 $(GO) build -o $(HOST_BIN) ./cmd/host

build-controller: $(BIN_DIR)
	CGO_ENABLED=1 $(GO) build -o $(CONTROLLER_BIN) ./cmd/controller

run-host: build-host
	$(HOST_BIN) -signaling ws://localhost:8080

run-controller: build-controller
	$(CONTROLLER_BIN) -signaling ws://localhost:8080

run-signaling:
	cd signaling-server && npm start

clean:
	rm -rf $(BIN_DIR)

test:
	$(GO) test -v -race ./internal/...
