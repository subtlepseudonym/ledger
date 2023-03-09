default: build

all: test build clean

build: format
	go build -o plaid2csv *.go

test:
	gotest --race -v ./...

format fmt:
	gofmt -l -w .

clean:
	go mod tidy
	go clean

.PHONY: all build clean format fmt test
