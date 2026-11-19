"""Fixtures globais para testes do mcp-brasil."""

import os

# Disable BM25 search transform for tests so the root server exposes all tools
# directly. This must happen before any mcp_brasil module is imported.
os.environ.setdefault("MCP_BRASIL_TOOL_SEARCH", "none")

# Force auth off during tests — prevents a local .env file from leaking
# MCP_BRASIL_API_TOKEN (or any other auth env var) into the test environment.
os.environ.setdefault("MCP_BRASIL_AUTH_MODE", "none")

for var in ("NO_PROXY", "no_proxy"):
    val = os.environ.get(var)
    if val and "[::1]" in val:
        os.environ[var] = val.replace("[::1]", "::1")
