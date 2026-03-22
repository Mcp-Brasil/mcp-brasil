"""Feature metadata and auto-registry for mcp-brasil.

This module implements convention-based auto-discovery of features.
Any subpackage of mcp_brasil that exports a FEATURE_META and has a
server.py with an `mcp` object will be automatically discovered,
validated, and mounted on the root server.

Pattern inspired by: Flask blueprints, Django app registry,
pytest plugin discovery, FastAPI router auto-include.

See ADR-002 for the full rationale.

Usage:
    from fastmcp import FastMCP
    from mcp_brasil._shared.feature import FeatureRegistry

    mcp = FastMCP("mcp-brasil 🇧🇷")
    registry = FeatureRegistry()
    registry.discover()
    registry.mount_all(mcp)
"""

from __future__ import annotations

import importlib
import logging
import os
import pkgutil
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from fastmcp import FastMCP

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class FeatureMeta:
    """Declarative metadata for a feature.

    Every feature must export a FEATURE_META instance in its __init__.py.
    The registry uses these metadata for discovery, validation, docs,
    and runtime decisions (auth gating, feature flags).

    Example:
        # src/mcp_brasil/ibge/__init__.py
        from mcp_brasil._shared.feature import FeatureMeta

        FEATURE_META = FeatureMeta(
            name="ibge",
            description="IBGE: localidades, população, PIB, nomes",
            api_base="https://servicodados.ibge.gov.br/api",
        )
    """

    name: str
    description: str
    version: str = "0.1.0"
    api_base: str = ""
    requires_auth: bool = False
    auth_env_var: str | None = None
    enabled: bool = True
    tags: list[str] = field(default_factory=list)

    def is_auth_available(self) -> bool:
        """Check if required auth credentials are available."""
        if not self.requires_auth:
            return True
        if self.auth_env_var is None:
            return False
        return bool(os.environ.get(self.auth_env_var))


@dataclass
class RegisteredFeature:
    """A feature that has been discovered, validated, and registered."""

    meta: FeatureMeta
    server: FastMCP
    module_path: str
    tool_count: int = 0


class FeatureRegistry:
    """Auto-registry that discovers, validates, and mounts features.

    Uses pkgutil.iter_modules() to scan subpackages of mcp_brasil,
    import those following the convention (FEATURE_META + server.mcp),
    and mount them on the root FastMCP server.

    Convention (all required for auto-discovery):
        1. Subpackage of mcp_brasil/ (directory with __init__.py)
        2. Name does NOT start with '_'
        3. __init__.py exports FEATURE_META: FeatureMeta
        4. server.py exports mcp: FastMCP
        5. If requires_auth=True, auth_env_var must be set in env

    To disable a feature: set enabled=False in FEATURE_META.
    """

    def __init__(self) -> None:
        self._features: dict[str, RegisteredFeature] = {}
        self._skipped: dict[str, str] = {}

    @property
    def features(self) -> dict[str, RegisteredFeature]:
        """All discovered and registered features."""
        return dict(self._features)

    @property
    def skipped(self) -> dict[str, str]:
        """Features that were skipped, with reasons."""
        return dict(self._skipped)

    def discover(self, package_name: str = "mcp_brasil") -> FeatureRegistry:
        """Discover all features in the package.

        Scans subpackages of `package_name` and registers those that
        follow the convention. Features that fail validation are logged
        as warnings and skipped — they do not break the server.

        Args:
            package_name: Base package to scan. Default: "mcp_brasil".

        Returns:
            self, for chaining: registry.discover().mount_all(mcp)
        """
        package = importlib.import_module(package_name)

        for _finder, name, ispkg in pkgutil.iter_modules(
            package.__path__, package.__name__ + "."
        ):
            short_name = name.rsplit(".", 1)[-1]

            # Skip non-packages and private modules
            if not ispkg or short_name.startswith("_"):
                continue

            try:
                self._try_register(name, short_name)
            except Exception as exc:
                reason = str(exc)
                self._skipped[short_name] = reason
                logger.warning("Feature '%s' skipped: %s", short_name, reason)

        return self

    def _try_register(self, module_path: str, short_name: str) -> None:
        """Try to import and register a single feature."""
        # Step 1: Import feature __init__.py
        feature_module = importlib.import_module(module_path)

        # Step 2: Check FEATURE_META exists and is valid
        meta = getattr(feature_module, "FEATURE_META", None)
        if meta is None:
            raise ValueError(f"No FEATURE_META in {module_path}")

        if not isinstance(meta, FeatureMeta):
            raise TypeError(
                f"FEATURE_META in {module_path} is not a FeatureMeta instance"
            )

        # Step 3: Check if feature is enabled
        if not meta.enabled:
            self._skipped[short_name] = "disabled (enabled=False)"
            logger.info("Feature '%s' is disabled, skipping.", short_name)
            return

        # Step 4: Check auth if required
        if not meta.is_auth_available():
            self._skipped[short_name] = (
                f"missing env var {meta.auth_env_var}"
            )
            logger.warning(
                "Feature '%s' requires %s (not set), skipping.",
                short_name,
                meta.auth_env_var,
            )
            return

        # Step 5: Import server.py and get the mcp object
        server_module = importlib.import_module(f"{module_path}.server")
        server = getattr(server_module, "mcp", None)

        if server is None:
            raise ValueError(f"No `mcp` object in {module_path}.server")

        # Step 6: Register
        self._features[short_name] = RegisteredFeature(
            meta=meta,
            server=server,
            module_path=module_path,
        )
        logger.info(
            "Registered feature '%s' v%s",
            meta.name,
            meta.version,
        )

    def mount_all(self, root_server: FastMCP) -> None:
        """Mount all discovered features on the root server.

        Each feature is mounted at /{feature_name}, sorted alphabetically.

        Args:
            root_server: The root FastMCP server to mount features on.
        """
        for name, feature in sorted(self._features.items()):
            root_server.mount(f"/{name}", feature.server)
            logger.info(
                "Mounted /%s — %s", name, feature.meta.description
            )

    def summary(self) -> str:
        """Return a human-readable summary of registered features.

        Useful for logging at startup and generating docs.
        """
        lines = [
            f"mcp-brasil — {len(self._features)} feature(s) active, "
            f"{len(self._skipped)} skipped\n"
        ]

        if self._features:
            lines.append("Active:")
            for name, feat in sorted(self._features.items()):
                auth_icon = "🔑" if feat.meta.requires_auth else "🔓"
                lines.append(
                    f"  /{name:<20} {auth_icon} {feat.meta.description}"
                )

        if self._skipped:
            lines.append("\nSkipped:")
            for name, reason in sorted(self._skipped.items()):
                lines.append(f"  {name:<20} ⏭️  {reason}")

        return "\n".join(lines)

    def get_feature(self, name: str) -> RegisteredFeature | None:
        """Get a registered feature by name."""
        return self._features.get(name)

    def list_tools(self) -> list[str]:
        """List all tool names across all features (for introspection)."""
        tools: list[str] = []
        for name, feat in sorted(self._features.items()):
            # Access FastMCP internal tool registry if available
            if hasattr(feat.server, "_tool_manager"):
                for tool_name in feat.server._tool_manager._tools:
                    tools.append(f"/{name}/{tool_name}")
        return tools
