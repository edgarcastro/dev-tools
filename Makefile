.PHONY: build test vet install

build:
	go build -o bin/devtools ./cmd/devtools

test:
	go vet ./...
	go test ./...

# Build the installer and launch it.
install: build
	./bin/devtools
