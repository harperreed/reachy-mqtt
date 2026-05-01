# ABOUTME: Build and development targets for the reachy-mqtt bridge daemon.
# ABOUTME: Provides build, test, clean, run, and Docker targets.

BINARY := reachy-mqtt
GO := go

.PHONY: all build test vet clean run docker-build

all: vet test build

build:
	$(GO) build -o $(BINARY) .

test:
	$(GO) test -v ./...

vet:
	$(GO) vet ./...

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY)

docker-build:
	docker build -t $(BINARY) .
