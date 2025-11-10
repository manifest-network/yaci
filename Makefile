VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -ldflags "-X github.com/manifest-network/yaci/cmd/yaci.Version=$(VERSION)" -tags manifest

#### Build ####
build: ## Build the binary
	@echo "--> Building development binary (version: $(VERSION))"
	@go build $(BUILD_FLAGS) -o bin/yaci ./main.go

.PHONY: build

#### Test ####
test: ## Run tests
	@echo "--> Running tests"
	@go test -v -short -race ./...

test-e2e: ## Run end-to-end tests
	@echo "--> Running end-to-end tests"
	@go test -v -race ./cmd/yaci/postgres_test.go -timeout 30m

.PHONY: test test-e2e

#### Coverage ####
COV_ROOT="/tmp/yaci-coverage"
COV_UNIT="${COV_ROOT}/unit"
COV_E2E="${COV_ROOT}/e2e"
COV_PKG="github.com/manifest-network/yaci/..."

coverage: ## Run tests with coverage
	@echo "--> Creating GOCOVERDIR"
	@mkdir -p ${COV_UNIT} ${COV_E2E}
	@echo "--> Cleaning up coverage files, if any"
	@rm -rf ${COV_UNIT}/* ${COV_E2E}/*
	@echo "--> Running short tests with coverage"
	@go test -v -short -timeout 30m -race -covermode=atomic -cover -cpu=$$(nproc) -coverpkg=${COV_PKG} ./... -args -test.gocoverdir="${COV_UNIT}"
	@echo "--> Running end-to-end tests with coverage"
	@go test -v -race -timeout 30m -race -covermode=atomic -cover -cpu=$$(nproc) -coverpkg=${COV_PKG} ./cmd/yaci/postgres_test.go -args -test.gocoverdir="${COV_E2E}"
	@echo "--> Merging coverage reports"
	@go tool covdata merge -i=${COV_UNIT},${COV_E2E} -o ${COV_ROOT}
	@echo "--> Converting binary coverage report to text format"
	@go tool covdata textfmt -i=${COV_ROOT} -o ${COV_ROOT}/coverage-merged.out
	@echo "--> Generating coverage report"
	@go tool cover -func=${COV_ROOT}/coverage-merged.out
	@echo "--> Generating HTML coverage report"
	@go tool cover -html=${COV_ROOT}/coverage-merged.out -o coverage.html
	@echo "--> Coverage report available at coverage.html"
	@echo "--> Cleaning up coverage files"
	@rm -rf ${COV_UNIT}/* ${COV_E2E}/*
	@echo "--> Running coverage complete"

#### Docker ####
docker-infra-up:
	@echo "--> Running docker compose up --build --wait -d"
	@cd docker/infra && docker compose up --build --wait -d && cd -

docker-infra-down:
	@echo "--> Running docker compose down -v"
	@cd docker/infra && docker compose down -v && cd -

docker-up:
	@echo "--> Running docker compose up --build --wait -d"
	@cd docker/yaci && docker compose up --build --wait -d && cd -

docker-down:
	@echo "--> Running docker compose down -v"
	@cd docker/yaci && docker compose down -v && cd -

.PHONY: docker-up docker-down docker-infra-up docker-infra-down

####  Linting  ####
golangci_lint_cmd=golangci-lint
golangci_version=v1.61.0

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run ./... --timeout 15m

lint-fix:
	@echo "--> Running linter and fixing issues"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run ./... --fix --timeout 15m

.PHONY: lint lint-fix

#### FORMAT ####
goimports_version=v0.26.0

format: ## Run formatter (goimports)
	@echo "--> Running goimports"
	@go install golang.org/x/tools/cmd/goimports@$(goimports_version)
	@find . -name '*.go' -exec goimports -w -local github.com/manifest-network/yaci {} \;

.PHONY: format

#### GOVULNCHECK ####
govulncheck_version=latest

govulncheck: ## Run govulncheck
	@echo "--> Running govulncheck"
	@go install golang.org/x/vuln/cmd/govulncheck@$(govulncheck_version)
	@govulncheck ./...

.PHONY: govulncheck
