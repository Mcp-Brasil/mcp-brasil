"""Testes do FeatureRegistry e FeatureMeta."""

import os
from unittest.mock import patch

import pytest
from fastmcp import Client, FastMCP

from mcp_brasil import settings
from mcp_brasil._shared.feature import FeatureMeta, FeatureRegistry, RegisteredFeature

# ---------------------------------------------------------------------------
# FeatureMeta
# ---------------------------------------------------------------------------


class TestFeatureMeta:
    def test_create_minimal(self) -> None:
        meta = FeatureMeta(name="ibge", description="IBGE API")
        assert meta.name == "ibge"
        assert meta.description == "IBGE API"
        assert meta.version == "0.1.0"
        assert meta.enabled is True
        assert meta.requires_auth is False

    def test_create_with_auth(self) -> None:
        meta = FeatureMeta(
            name="transparencia",
            description="Portal da Transparência",
            requires_auth=True,
            auth_env_var="TRANSPARENCIA_API_KEY",
        )
        assert meta.requires_auth is True
        assert meta.auth_env_var == "TRANSPARENCIA_API_KEY"

    def test_is_auth_available_no_auth_required(self) -> None:
        meta = FeatureMeta(name="ibge", description="IBGE")
        assert meta.is_auth_available() is True

    def test_is_auth_available_missing_env_var(self) -> None:
        meta = FeatureMeta(
            name="t",
            description="T",
            requires_auth=True,
            auth_env_var="FAKE_KEY_NOT_SET",
        )
        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop("FAKE_KEY_NOT_SET", None)
            assert meta.is_auth_available() is False

    def test_is_auth_available_env_var_set(self) -> None:
        meta = FeatureMeta(
            name="t",
            description="T",
            requires_auth=True,
            auth_env_var="TEST_MCP_KEY",
        )
        with patch.dict(os.environ, {"TEST_MCP_KEY": "secret"}):
            assert meta.is_auth_available() is True

    def test_is_auth_available_requires_auth_no_env_var(self) -> None:
        meta = FeatureMeta(name="t", description="T", requires_auth=True)
        assert meta.is_auth_available() is False

    def test_frozen(self) -> None:
        meta = FeatureMeta(name="ibge", description="IBGE")
        with pytest.raises(AttributeError):
            meta.name = "other"  # type: ignore[misc]

    def test_tags_default_empty(self) -> None:
        meta = FeatureMeta(name="ibge", description="IBGE")
        assert meta.tags == []

    def test_tags_custom(self) -> None:
        meta = FeatureMeta(name="ibge", description="IBGE", tags=["geo", "censo"])
        assert meta.tags == ["geo", "censo"]


# ---------------------------------------------------------------------------
# FeatureRegistry
# ---------------------------------------------------------------------------


class TestFeatureRegistry:
    def test_empty_registry(self) -> None:
        registry = FeatureRegistry()
        assert registry.features == {}
        assert registry.skipped == {}

    def test_discover_returns_self_for_chaining(self) -> None:
        """discover() retorna self para permitir chaining."""
        registry = FeatureRegistry()
        result = registry.discover("mcp_brasil.data")
        assert result is registry

    def test_discover_finds_ibge(self) -> None:
        """Discovery encontra a feature ibge in data package."""
        registry = FeatureRegistry()
        registry.discover("mcp_brasil.data")
        assert "ibge" in registry.features

    def test_discover_finds_redator(self) -> None:
        """Discovery encontra a feature redator in agentes package."""
        registry = FeatureRegistry()
        registry.discover("mcp_brasil.agentes")
        assert "redator" in registry.features

    def test_summary_empty(self) -> None:
        registry = FeatureRegistry()
        summary = registry.summary()
        assert "0 feature(s) active" in summary
        assert "0 skipped" in summary

    def test_get_feature_not_found(self) -> None:
        registry = FeatureRegistry()
        assert registry.get_feature("nonexistent") is None

    def test_mount_all_empty(self) -> None:
        """Mount com registry vazio não levanta exceção."""
        registry = FeatureRegistry()
        root = FastMCP("test-root")
        registry.mount_all(root)  # should not raise

    def test_register_and_mount_manual(self) -> None:
        """Registra uma feature manualmente e monta no root."""
        registry = FeatureRegistry()

        meta = FeatureMeta(name="test_feat", description="Test feature")
        sub_server = FastMCP("test-sub")

        @sub_server.tool
        def ping() -> str:
            """Ping tool."""
            return "pong"

        registry._features["test_feat"] = RegisteredFeature(
            meta=meta,
            server=sub_server,
            module_path="fake.module",
        )

        root = FastMCP("test-root")
        registry.mount_all(root)

        assert registry.get_feature("test_feat") is not None
        assert "test_feat" in registry.summary()

    def test_summary_with_features(self) -> None:
        registry = FeatureRegistry()
        meta = FeatureMeta(name="ibge", description="IBGE dados")
        sub = FastMCP("sub")
        registry._features["ibge"] = RegisteredFeature(meta=meta, server=sub, module_path="m")
        summary = registry.summary()
        assert "1 feature(s) active" in summary
        assert "ibge" in summary
        assert "IBGE dados" in summary

    def test_summary_with_skipped(self) -> None:
        registry = FeatureRegistry()
        registry._skipped["broken"] = "missing FEATURE_META"
        summary = registry.summary()
        assert "1 skipped" in summary
        assert "broken" in summary

    def test_skipped_returns_copy(self) -> None:
        registry = FeatureRegistry()
        registry._skipped["x"] = "reason"
        skipped = registry.skipped
        skipped["y"] = "other"
        assert "y" not in registry._skipped

    def test_features_returns_copy(self) -> None:
        registry = FeatureRegistry()
        features = registry.features
        features["fake"] = None  # type: ignore[assignment]
        assert "fake" not in registry._features


# ---------------------------------------------------------------------------
# Integration: mount e call via fastmcp.Client
# ---------------------------------------------------------------------------


class TestRegistryIntegration:
    @pytest.mark.asyncio
    async def test_mounted_tool_callable(self) -> None:
        """Tool montada via registry é chamável pelo Client."""
        sub = FastMCP("sub")

        @sub.tool
        def echo(msg: str) -> str:
            """Echo a message."""
            return f"echo: {msg}"

        root = FastMCP("root")
        root.mount(sub, namespace="test")

        async with Client(root) as client:
            result = await client.call_tool("test_echo", {"msg": "hello"})
            assert result.data == "echo: hello"

    @pytest.mark.asyncio
    async def test_root_server_starts_empty(self) -> None:
        """Root server sem features montadas funciona."""
        from mcp_brasil.server import mcp

        async with Client(mcp) as client:
            tools = await client.list_tools()
            tool_names = [t.name for t in tools]
            assert "listar_features" in tool_names

    @pytest.mark.asyncio
    async def test_listar_features_tool(self) -> None:
        """Meta-tool listar_features retorna summary."""
        from mcp_brasil.server import mcp

        async with Client(mcp) as client:
            result = await client.call_tool("listar_features", {})
            assert "mcp-brasil" in result.data


# ---------------------------------------------------------------------------
# Allow/deny lists (MCP_BRASIL_FEATURES / MCP_BRASIL_FEATURES_DISABLED)
# ---------------------------------------------------------------------------


class TestFeatureSelection:
    def test_allowlist_keeps_only_listed_features(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(settings, "FEATURES_ENABLED", ["ibge"])
        monkeypatch.setattr(settings, "FEATURES_DISABLED", [])
        registry = FeatureRegistry().discover("mcp_brasil.data")
        assert list(registry.features) == ["ibge"]
        assert "brasilapi" in registry.skipped

    def test_denylist_drops_listed_features(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(settings, "FEATURES_ENABLED", [])
        monkeypatch.setattr(settings, "FEATURES_DISABLED", ["ibge"])
        registry = FeatureRegistry().discover("mcp_brasil.data")
        assert "ibge" not in registry.features
        assert "brasilapi" in registry.features

    def test_denylist_wins_over_allowlist(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(settings, "FEATURES_ENABLED", ["ibge", "brasilapi"])
        monkeypatch.setattr(settings, "FEATURES_DISABLED", ["ibge"])
        registry = FeatureRegistry().discover("mcp_brasil.data")
        assert "ibge" not in registry.features
        assert "brasilapi" in registry.features

    def test_no_lists_keeps_default_behaviour(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(settings, "FEATURES_ENABLED", [])
        monkeypatch.setattr(settings, "FEATURES_DISABLED", [])
        registry = FeatureRegistry().discover("mcp_brasil.data")
        assert "ibge" in registry.features
        assert "brasilapi" in registry.features
