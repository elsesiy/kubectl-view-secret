SOURCES := $(shell find . -name '*.go')
BINARY := kubectl-view-secret

build: kubectl-view-secret

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $(BINARY) ./cmd/$(BINARY).go