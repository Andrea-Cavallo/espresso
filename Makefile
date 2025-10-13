.PHONY: help test coverage lint fmt vet build clean install

# Variables
BINARY_NAME=espresso
GO=go
GOTEST=$(GO) test
GOVET=$(GO) vet
GOFMT=gofmt
GOLINT=golangci-lint

help: ## Mostra questo help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Esegui tutti i test
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

test-short: ## Esegui test veloci
	@echo "Running short tests..."
	$(GOTEST) -short ./...

coverage: ## Genera report coverage
	@echo "Generating coverage report..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

coverage-cli: ## Mostra coverage nel terminale
	@echo "Coverage summary..."
	$(GOTEST) -cover ./...

bench: ## Esegui benchmark
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

lint: ## Esegui linting
	@echo "Running linter..."
	$(GOLINT) run ./...

fmt: ## Formatta il codice
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet: ## Esegui go vet
	@echo "Running go vet..."
	$(GOVET) ./...

tidy: ## Tidy go modules
	@echo "Tidying modules..."
	$(GO) mod tidy

verify: fmt vet lint test ## Verifica tutto prima del commit

build: ## Build il progetto
	@echo "Building..."
	$(GO) build -o bin/$(BINARY_NAME) ./...

clean: ## Pulisci build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out coverage.html
	$(GO) clean

install: ## Installa le dipendenze
	@echo "Installing dependencies..."
	$(GO) mod download

check: ## Controlla dipendenze obsolete
	@echo "Checking for updates..."
	$(GO) list -u -m all

deps-upgrade: ## Aggiorna dipendenze
	@echo "Upgrading dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

ci: verify coverage ## Esegui pipeline CI locale
	@echo "CI checks completed!"

release-dry: ## Simula release
	@echo "Dry run release..."
	goreleaser release --snapshot --clean

.DEFAULT_GOAL := help