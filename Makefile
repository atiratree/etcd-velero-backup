BIN          ?= etcd-backup-plugin
REGISTRY     ?= quay.io/MY_NAME
VERSION      ?= latest
IMAGE        ?= $(REGISTRY)/$(BIN):$(VERSION)
GOOS         ?= $(shell go env GOOS 2>/dev/null || echo linux)
GOARCH       ?= $(shell go env GOARCH 2>/dev/null || echo amd64)
OUTPUT_DIR   ?= _output/bin/$(GOOS)/$(GOARCH)

.PHONY: all
all: build

.PHONY: local
local:
	CGO_ENABLED=0 go build -o $(OUTPUT_DIR)/$(BIN) .

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(OUTPUT_DIR)/$(BIN) .

.PHONY: container
container:
	podman build -t $(IMAGE) .

.PHONY: push
push: container
	podman push $(IMAGE)

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -rf _output

.PHONY: verify
verify:
	go vet ./...
