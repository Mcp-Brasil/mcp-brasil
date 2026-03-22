"""mcp-brasil root server — auto-discovers and mounts all features.

This file uses FeatureRegistry for zero-touch feature onboarding.
You should NEVER need to edit this file to add a new feature.
Just create a new directory following the convention in ADR-001/002.

Usage:
    fastmcp run mcp_brasil.server:mcp
    fastmcp run mcp_brasil.server:mcp --transport http --port 8000
"""

import logging

from fastmcp import FastMCP

from ._shared.feature import FeatureRegistry

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(name)s] %(levelname)s: %(message)s",
)
logger = logging.getLogger("mcp-brasil")

# Create the root server
mcp = FastMCP("mcp-brasil 🇧🇷")

# Auto-discover and mount all features
registry = FeatureRegistry()
registry.discover()
registry.mount_all(mcp)

logger.info("\n%s", registry.summary())


# Expose a meta-tool for introspection
@mcp.tool
def listar_features() -> str:
    """Lista todas as features (APIs) disponíveis no mcp-brasil.

    Use esta tool para saber quais APIs governamentais estão conectadas
    e quais tools cada uma oferece.

    Returns:
        Resumo das features ativas com descrição e status de autenticação.
    """
    return registry.summary()


if __name__ == "__main__":
    mcp.run()
