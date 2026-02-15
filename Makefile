.PHONY: build run test lint clean docker-build

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=tarotd

build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/tarotd

run:
	$(GOCMD) run ./cmd/tarotd

test:
	$(GOTEST) -v -race -cover ./...

test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

deps:
	$(GOCMD) mod download
	$(GOCMD) mod tidy

docker:
	docker build -t tarot-as-a-service .
