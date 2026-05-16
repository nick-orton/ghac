.PHONY: build test test-integration clean

build:
	go build -o ghac ./cmd/ghac

test:
	go test ./...

test-integration:
	go test -tags integration ./...

clean:
	rm -f ghac
