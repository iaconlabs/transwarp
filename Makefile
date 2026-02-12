# Transwarp Automation Makefile

.PHONY: help dev-on dev-off release bundle test-all bench-all

ARGS ?=

# Default version for release
VERSION ?= v0.0.1


GOTEST := $(shell command -v gotest 2> /dev/null)

ifdef GOTEST
    TEST_CMD := gotest
else
    TEST_CMD := go test
endif

TEST_FLAGS := -race

help: ## Show this help message
	@echo 'Usage: make [target] [VERSION=vX.Y.Z]'
	@echo ''
	@echo 'Targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

dev-on: ## Link all modules locally for development (patch_mods on)
	@echo "🔗 Linking modules locally..."
	@bash ./tools/patch_mods.sh on
	@echo "✅ Local development mode enabled."

dev-off: ## Unlink local modules for production/push (patch_mods off)
	@echo "🌐 Unlinking local modules..."
	@bash ./tools/patch_mods.sh off
	@echo "✅ Production mode enabled (remote dependencies)."

release: dev-off ## Prepare a release by unlinking and tagging (Usage: make release VERSION=v1.0.0)
	`@echo` "🏷️ Tagging version $(VERSION)..."
	`@bash` ./tools/tagger.sh $(VERSION)
	`@echo` "🚀 Version $(VERSION) tagged for all modules."

bundle: ## Consolidate project code for review or AI context
	@echo "📦 Bundling codebase..."
	@bash ./tools/bundle_code.sh

test-all: ## Run all tests. Usage: make test-all ARGS="-v"
	@echo "Using test command: $(TEST_CMD)"
	@find . -name "go.mod" -execdir sh -c "$(TEST_CMD) $(TEST_FLAGS) ./..." \;

bench-all: ## Run benchmarks for all modules
	@echo "⚡ Running all benchmarks..."
	@find . -name "go.mod" -execdir go test -bench=. -run=^$ -benchmem $(ARGS) ./... \;


update-deps: ## Update all dependencies to latest minor/patch versions
	@echo "🆙 Updating dependencies in all modules..."
	@find . -name "go.mod" -execdir go get -u ./... \;
	@find . -name "go.mod" -execdir go mod tidy \;
	@echo "✅ All modules updated and tidied."

clean-sums: ## Remove all go.sum files recursively
	@echo "🗑️ Removing all go.sum files..."
	@find . -name "go.sum" -type f -delete
	@echo "✅ go.sum files removed."