SOURCES := $(shell find . -name '*.go')
BINARY := kubectl-view-secret
COV_REPORT := "coverage.txt"

build: kubectl-view-secret

test: $(SOURCES)
	go test -v -short -race -timeout 30s ./...

test-cov:
	go test ./... -coverprofile=$(COV_REPORT)
	go tool cover -html=$(COV_REPORT)

clean:
	@rm -rf $(BINARY)

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $(BINARY) -ldflags="-s -w" ./cmd/$(BINARY).go
