.PHONY: test test-verbose test-coverage coverage-html test-func fmt lint tidy clean build check help

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

coverage-html: test-coverage
	go tool cover -html=coverage.out -o coverage.html

test-func:
	go test -v -run $(FUNC) ./...

fmt:
	gofmt -l -s -w .

lint:
	golangci-lint run --timeout=5m

tidy:
	go mod tidy

build:
	go build ./...

clean:
	rm -f coverage.out coverage.html

check: fmt lint test

help:
	@echo "make test           run the test suite"
	@echo "make test-verbose   run the test suite with -v"
	@echo "make test-coverage  run tests with coverage"
	@echo "make coverage-html  generate an HTML coverage report"
	@echo "make test-func FUNC=Name  run a single test"
	@echo "make fmt            gofmt all files"
	@echo "make lint           run golangci-lint"
	@echo "make tidy           go mod tidy"
	@echo "make build          build all packages"
	@echo "make clean          remove coverage artifacts"
	@echo "make check          fmt + lint + test"
