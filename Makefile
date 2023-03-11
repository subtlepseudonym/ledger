default: build

all: test build clean

build:
	mkdir -p bin
	cd cmd && \
	for dir in *; do \
		go build -o "../bin/$$dir" "./$$dir"; \
	done

test:
	gotest --race -v ./...

format fmt:
	go fmt ./...

clean:
	go mod tidy
	go clean

.PHONY: all build clean format fmt test
