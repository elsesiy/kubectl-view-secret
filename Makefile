SOURCES := $(shell find . -name '*.go')
BINARY := kubectl-view-secret

build: kubectl-view-secret

test: $(SOURCES)
	go test -v -short -race -timeout 30s ./...

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $(BINARY) -ldflags="-s -w" ./cmd/$(BINARY).go