default: all

all: test clean

test:
	gotest --race -v ./...

format fmt:
	gofmt -l -w .

clean:
	go mod tidy
	go clean

.PHONY: all test format fmt clean
