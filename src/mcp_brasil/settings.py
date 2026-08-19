"""Configuração global do mcp-brasil.

Valores podem ser sobrescritos via variáveis de ambiente.
Carrega o arquivo `.env` do diretório de trabalho atual, se existir.
Para apontar outro caminho, use `MCP_BRASIL_DOTENV=/caminho/para/.env`.
"""

from __future__ import annotations

import os
from pathlib import Path

from dotenv import load_dotenv

# `load_dotenv()` sem argumento SOBE a árvore de diretórios a partir do arquivo
# que o chamou. Num pacote instalado isso parte de site-packages e pode acabar
# carregando um `.env` de fora do projeto, injetando credenciais sem que
# ninguém tenha pedido. Carregamos apenas o `.env` do diretório atual — que é
# o que a documentação sempre prometeu — com escape hatch explícito.
_dotenv_path = Path(os.environ.get("MCP_BRASIL_DOTENV", "")) or Path.cwd() / ".env"
if _dotenv_path.is_file():
    load_dotenv(_dotenv_path)

# --- HTTP Client ---
HTTP_TIMEOUT: float = float(os.environ.get("MCP_BRASIL_HTTP_TIMEOUT", "30.0"))
HTTP_MAX_RETRIES: int = int(os.environ.get("MCP_BRASIL_HTTP_MAX_RETRIES", "3"))
HTTP_BACKOFF_BASE: float = float(os.environ.get("MCP_BRASIL_HTTP_BACKOFF_BASE", "1.0"))
USER_AGENT: str = os.environ.get("MCP_BRASIL_USER_AGENT", "mcp-brasil/0.1.0")

# --- Tool Discovery ---
# "bm25" (default): BM25 search transform — replaces list_tools with search_tools + call_tool
# "code_mode": Experimental CodeMode transform — search + get_tags + execute
# "none": No transform — all 154+ tools visible at once
TOOL_SEARCH: str = os.environ.get("MCP_BRASIL_TOOL_SEARCH", "bm25")

# --- Authentication ---
# When set, requires Authorization: Bearer <token> for HTTP transport.
# Leave empty/unset for local/stdio usage without auth.
MCP_BRASIL_API_TOKEN: str | None = os.environ.get("MCP_BRASIL_API_TOKEN") or None

# --- Authentication Strategy ---
# Modes: "none" | "static" | "oauth" | "multi"
# Auto-detect (backward compat): if unset, defaults to "static" when
# MCP_BRASIL_API_TOKEN is set, else "none".
_AUTH_MODE_RAW: str = os.environ.get("MCP_BRASIL_AUTH_MODE", "").strip().lower()
AUTH_MODE: str = _AUTH_MODE_RAW or ("static" if MCP_BRASIL_API_TOKEN else "none")

# OAuth — common
OAUTH_PROVIDER: str = os.environ.get("MCP_BRASIL_OAUTH_PROVIDER", "").strip().lower()
MCP_BRASIL_BASE_URL: str = os.environ.get("MCP_BRASIL_BASE_URL", "").strip()


def _parse_scopes(raw: str, default: list[str] | None = None) -> list[str]:
    """Parse a comma-separated list of scopes into a list[str].

    Returns ``default`` (or an empty list) when ``raw`` contains no entries.
    """
    items = [s.strip() for s in raw.split(",") if s.strip()]
    return items if items else (default or [])


# Azure (Entra ID)
AZURE_CLIENT_ID: str = os.environ.get("AZURE_CLIENT_ID", "")
AZURE_CLIENT_SECRET: str = os.environ.get("AZURE_CLIENT_SECRET", "")
AZURE_TENANT_ID: str = os.environ.get("AZURE_TENANT_ID", "")
AZURE_REQUIRED_SCOPES: list[str] = _parse_scopes(
    os.environ.get("AZURE_REQUIRED_SCOPES", ""), ["read"]
)

# Google
GOOGLE_CLIENT_ID: str = os.environ.get("GOOGLE_CLIENT_ID", "")
GOOGLE_CLIENT_SECRET: str = os.environ.get("GOOGLE_CLIENT_SECRET", "")
GOOGLE_REQUIRED_SCOPES: list[str] = _parse_scopes(
    os.environ.get("GOOGLE_REQUIRED_SCOPES", ""), ["openid", "email"]
)

# GitHub
GITHUB_CLIENT_ID: str = os.environ.get("GITHUB_CLIENT_ID", "")
GITHUB_CLIENT_SECRET: str = os.environ.get("GITHUB_CLIENT_SECRET", "")
GITHUB_REQUIRED_SCOPES: list[str] = _parse_scopes(os.environ.get("GITHUB_REQUIRED_SCOPES", ""))

# WorkOS (AuthKit)
AUTHKIT_DOMAIN: str = os.environ.get("AUTHKIT_DOMAIN", "").strip()
AUTHKIT_REQUIRED_SCOPES: list[str] = _parse_scopes(os.environ.get("AUTHKIT_REQUIRED_SCOPES", ""))

# --- LLM Discovery (recomendar_tools) ---
ANTHROPIC_API_KEY: str = os.environ.get("ANTHROPIC_API_KEY", "")

# --- Datasets (ADR-004: local cache com DuckDB) ---
# Opt-in: lista CSV dos datasets a habilitar (ex: "spu_siapa,cnpj_rfb").
# Datasets não listados NÃO são carregados nem expostos como tools.
DATASETS_ENABLED: list[str] = _parse_scopes(os.environ.get("MCP_BRASIL_DATASETS", ""))

# Diretório raiz do cache de datasets (XDG-compliant).
DATASET_CACHE_DIR: str = os.environ.get(
    "MCP_BRASIL_DATASET_CACHE_DIR",
    os.path.join(os.path.expanduser("~"), ".cache", "mcp-brasil"),
)

# Modo de refresh: "auto" (respeita TTL), "never" (nunca baixa; falha se cache vazio),
# "force" (força re-download em toda inicialização).
DATASET_REFRESH_MODE: str = os.environ.get("MCP_BRASIL_DATASET_REFRESH", "auto").strip().lower()

# Limite total de cache em GB (soft limit — futura feature de LRU eviction).
DATASET_MAX_CACHE_GB: float = float(os.environ.get("MCP_BRASIL_DATASET_MAX_CACHE_GB", "20"))

# Datasets com PII liberada. Default: todas as colunas PII mascaradas.
DATASETS_ALLOW_PII: list[str] = _parse_scopes(os.environ.get("MCP_BRASIL_LGPD_ALLOW_PII", ""))

# Timeout do download em segundos (datasets grandes precisam de mais).
DATASET_DOWNLOAD_TIMEOUT: float = float(os.environ.get("MCP_BRASIL_DATASET_TIMEOUT", "600"))
