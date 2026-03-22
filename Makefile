# mcp-brasil — Makefile

.DEFAULT_GOAL := help
.PHONY: help sync dev test test-feature lint ruff fix types run serve inspect ci clean

## —— Setup ——

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

sync: ## Install production dependencies
	uv sync

dev: ## Install all dependencies (prod + dev)
	uv sync --extra dev

## —— Quality ——

lint: ## Run lint + format check
	uv run ruff check src/ tests/ && uv run ruff format --check src/ tests/

ruff: ## Auto-fix lint + format
	uv run ruff check --fix src/ tests/ && uv run ruff format src/ tests/

types: ## Run mypy strict type checking
	uv run mypy src/mcp_brasil/

test: ## Run all tests
	uv run pytest -v

test-feature: ## Run tests for a specific feature (usage: make test-feature F=ibge)
	uv run pytest tests/$(F)/ -v

ci: lint types test ## Full CI pipeline: lint + types + test

## —— Server ——

run: ## Run MCP server (stdio)
	uv run fastmcp run mcp_brasil.server:mcp

serve: ## Run MCP server (HTTP :8000)
	uv run fastmcp run mcp_brasil.server:mcp --transport http --port 8000

inspect: ## Inspect MCP server (list tools, resources, prompts)
	uv run fastmcp inspect mcp_brasil.server:mcp

## —— Misc ——

clean: ## Remove build artifacts and caches
	rm -rf .pytest_cache .mypy_cache .ruff_cache dist/ build/ *.egg-info
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
