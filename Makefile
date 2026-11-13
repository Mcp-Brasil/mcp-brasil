# mcp-brasil — Makefile

.DEFAULT_GOAL := help
.PHONY: help sync dev test test-feature lint fix types run serve inspect ci clean build changelog version release-patch release-minor release-major diagrams

## —— Setup ——

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

sync: ## Install production dependencies
	uv sync

dev: ## Install all dependencies (prod + dev)
	uv sync --group dev

## —— Quality ——

lint: ## Run lint + format check
	uv run ruff check src/ tests/ && uv run ruff format --check src/ tests/

fix: ## Auto-fix lint + format
	uv run ruff check --fix src/ tests/ && uv run ruff format src/ tests/

types: ## Run mypy strict type checking
	uv run mypy src/mcp_brasil/

test: ## Run all tests
	uv run pytest -v

test-feature: ## Run tests for a specific feature (usage: make test-feature F=ibge)
	uv run pytest tests/data/$(F)/ -v 2>/dev/null || uv run pytest tests/agentes/$(F)/ -v

test-go: ## Run Go tests
	go test -v ./...

bench-go: ## Run Go benchmarks
	go test -bench=. ./pkg/transparencia/...

build-go: ## Build native Go engine binary
	mkdir -p bin && go build -o bin/transparencia-go ./cmd/transparencia-go

ci: lint types test-go test ## Full CI pipeline: lint + types + test-go + test


## —— Server ——

run: ## Run MCP server (stdio)
	uv run python -m mcp_brasil.server

serve: ## Run MCP server (HTTP :8000)
	uv run python -c "from mcp_brasil.server import mcp; mcp.run(transport='streamable-http', host='0.0.0.0', port=8000)"

inspect: ## Inspect MCP server tools/resources/prompts
	uv run python -c "from mcp_brasil.server import mcp, registry; print(registry.summary())"

## —— Release ——

version: ## Show current version
	@uv version

build: ## Build package (sdist + wheel)
	uv build

changelog: ## Generate CHANGELOG.md with git-cliff
	git cliff -o CHANGELOG.md

release-patch: ci ## Release patch version (0.1.0 → 0.1.1)
	@scripts/release.sh patch

release-minor: ci ## Release minor version (0.1.0 → 0.2.0)
	@scripts/release.sh minor

release-major: ci ## Release major version (0.1.0 → 1.0.0)
	@scripts/release.sh major

## —— Docs ——

diagrams: ## Generate architecture diagrams (requires graphviz)
	uv run python scripts/generate_diagrams.py

## —— Misc ——

clean: ## Remove build artifacts and caches
	rm -rf .pytest_cache .mypy_cache .ruff_cache dist/ build/ *.egg-info
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
