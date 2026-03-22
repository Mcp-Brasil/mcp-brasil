"""Configuração global do mcp-brasil.

Valores podem ser sobrescritos via variáveis de ambiente.
Carrega automaticamente o arquivo .env na raiz do projeto.
"""

from __future__ import annotations

import os

from dotenv import load_dotenv

load_dotenv()

# --- HTTP Client ---
HTTP_TIMEOUT: float = float(os.environ.get("MCP_BRASIL_HTTP_TIMEOUT", "30.0"))
HTTP_MAX_RETRIES: int = int(os.environ.get("MCP_BRASIL_HTTP_MAX_RETRIES", "3"))
HTTP_BACKOFF_BASE: float = float(os.environ.get("MCP_BRASIL_HTTP_BACKOFF_BASE", "1.0"))
USER_AGENT: str = os.environ.get("MCP_BRASIL_USER_AGENT", "mcp-brasil/0.1.0")
