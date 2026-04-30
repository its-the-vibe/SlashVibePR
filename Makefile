BINARY := slashvibeprs

.PHONY: build test lint fmt clean

## build: Compile the binary
build:
	go build -o $(BINARY) .

## test: Run all unit tests
test:
	go test ./...

## lint: Run go vet and check formatting
lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "The following files are not gofmt-formatted:"; gofmt -l .; exit 1)

## fmt: Format source code with gofmt
fmt:
	gofmt -w .

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
