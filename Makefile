.PHONY: build test fmt format lint fix run clean

build:
	go build -o go-pathfinder .

test:
	go test ./... -count=1

fmt format:
	gofmt -s -w .

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || \
	  (echo 'golangci-lint not installed, falling back to go vet' && go vet ./...)

fix: fmt
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run --fix ./... || true

run: build
	./go-pathfinder

clean:
	rm -f go-pathfinder
