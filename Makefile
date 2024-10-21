VERSION = v0.0.2

#### Build ####
build: ## Build the binary
	@echo "--> Building development binary"
	@go build -ldflags="-X github.com/liftedinit/cosmos-dump/cmd/cosmos-dump.Version=$(VERSION)" -o bin/cosmos-dump ./main.go

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
	@find . -name '*.go' -exec goimports -w -local github.com/liftedinit/cosmos-dump {} \;

.PHONY: format

#### GOVULNCHECK ####
govulncheck_version=v1.1.3

govulncheck: ## Run govulncheck
	@echo "--> Running govulncheck"
	@go install golang.org/x/vuln/cmd/govulncheck@$(govulncheck_version)
	@govulncheck ./...

.PHONY: govulncheck
