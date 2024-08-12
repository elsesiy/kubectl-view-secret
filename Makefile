SOURCES := $(shell find . -name '*.go')
BINARY := kubectl-view-secret
COV_REPORT := "coverage.txt"

build: kubectl-view-secret

bootstrap:
	./hack/kind-bootstrap.sh

test: $(SOURCES)
	go test -v -short -race -timeout 30s ./...

test-cov:
	go test ./... -coverprofile=$(COV_REPORT)
	go tool cover -html=$(COV_REPORT)

clean:
	@rm -rf $(BINARY)
	@kind delete cluster --name kvs-test

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $(BINARY) -ldflags="-s -w" ./cmd/$(BINARY).go
