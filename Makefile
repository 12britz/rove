BINARY := rove
VERSION := 0.1.0
PREFIX ?= $(HOME)/.local/bin

LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build install test vet run clean

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

install: build
	mkdir -p $(PREFIX)
	cp bin/$(BINARY) $(PREFIX)/$(BINARY)
	@echo "installed to $(PREFIX)/$(BINARY) (make sure that is on your PATH)"

test:
	go test ./...

vet:
	go vet ./...

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin