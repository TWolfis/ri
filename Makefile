BIN := bin/ri

.PHONY: build install run test cover vet fmt check clean

build:
	go build -o $(BIN) ./cmd/ri

install:
	go install ./cmd/ri

# Pass CLI arguments with ARGS, e.g. make run ARGS="-type go -name /tmp/demo -skip-commands"
run:
	go run ./cmd/ri $(ARGS)

test:
	go test ./...

# Writes coverage.out (uploaded to Codecov by CI) and prints the total
cover:
	go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet:
	go vet ./...

fmt:
	gofmt -l -w .

# Everything CI would check: formatting, vet and tests
check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	$(MAKE) vet test

clean:
	rm -rf bin
