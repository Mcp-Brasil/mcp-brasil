"""Tests for the batch execution module."""

from __future__ import annotations

from typing import Any
from unittest.mock import AsyncMock, MagicMock

import pytest

from mcp_brasil._shared import batch


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    return ctx


@pytest.fixture(autouse=True)
def _reset_dispatch() -> None:
    """Clear dispatch table before each test."""
    batch._dispatch.clear()


class TestBuildDispatch:
    def test_builds_from_registry(self) -> None:
        """Should discover tools from feature modules."""
        result = batch.build_dispatch(_real_registry())
        # Should find at least ibge tools
        assert any(k.startswith("ibge_") for k in result)

    def test_finds_nested_features(self) -> None:
        """Should discover tools in sub-packages like compras/pncp."""
        result = batch.build_dispatch(_real_registry())
        assert any(k.startswith("compras_pncp_") for k in result)
        assert any(k.startswith("compras_dadosabertos_") for k in result)

    def test_caches_result(self) -> None:
        """Second call should return cached dispatch table."""
        reg = _real_registry()
        first = batch.build_dispatch(reg)
        second = batch.build_dispatch(reg)
        assert first is second


class TestExecuteBatch:
    @pytest.mark.asyncio
    async def test_empty_list(self) -> None:
        ctx = _mock_ctx()
        result = await batch.execute_batch([], ctx)
        assert "Nenhuma consulta" in result

    @pytest.mark.asyncio
    async def test_exceeds_limit(self) -> None:
        ctx = _mock_ctx()
        queries = [{"tool": "x", "args": {}} for _ in range(11)]
        result = await batch.execute_batch(queries, ctx)
        assert "Máximo de 10" in result

    @pytest.mark.asyncio
    async def test_unknown_tool(self) -> None:
        ctx = _mock_ctx()
        result = await batch.execute_batch([{"tool": "nonexistent_tool", "args": {}}], ctx)
        assert "não encontrada" in result

    @pytest.mark.asyncio
    async def test_calls_tool_with_ctx(self) -> None:
        """Should pass ctx to tools that accept it."""
        calls: list[dict[str, Any]] = []

        async def tool_with_ctx(ctx: object, param: str) -> str:
            calls.append({"ctx": ctx, "param": param})
            return "resultado ok"

        batch._dispatch["test_tool"] = tool_with_ctx

        ctx = _mock_ctx()
        result = await batch.execute_batch(
            [{"tool": "test_tool", "args": {"param": "value"}}], ctx
        )
        assert "resultado ok" in result
        assert len(calls) == 1

    @pytest.mark.asyncio
    async def test_calls_tool_without_ctx(self) -> None:
        """Should work with tools that don't take ctx."""

        async def no_ctx_tool(name: str) -> str:
            return f"hello {name}"

        batch._dispatch["greet"] = no_ctx_tool

        ctx = _mock_ctx()
        result = await batch.execute_batch([{"tool": "greet", "args": {"name": "world"}}], ctx)
        assert "hello world" in result

    @pytest.mark.asyncio
    async def test_parallel_execution(self) -> None:
        """Should execute multiple queries concurrently."""
        call_count = 0

        async def counting_tool(n: int) -> str:
            nonlocal call_count
            call_count += 1
            return f"result-{n}"

        batch._dispatch["counter"] = counting_tool

        ctx = _mock_ctx()
        result = await batch.execute_batch(
            [
                {"tool": "counter", "args": {"n": 1}},
                {"tool": "counter", "args": {"n": 2}},
                {"tool": "counter", "args": {"n": 3}},
            ],
            ctx,
        )
        assert call_count == 3
        assert "result-1" in result
        assert "result-2" in result
        assert "result-3" in result

    @pytest.mark.asyncio
    async def test_handles_tool_error(self) -> None:
        """Should catch exceptions and include error in results."""

        async def failing_tool() -> str:
            msg = "API timeout"
            raise TimeoutError(msg)

        batch._dispatch["fail"] = failing_tool

        ctx = _mock_ctx()
        result = await batch.execute_batch([{"tool": "fail", "args": {}}], ctx)
        assert "Erro" in result
        assert "timeout" in result.lower()

    @pytest.mark.asyncio
    async def test_mixed_success_and_failure(self) -> None:
        """Should return partial results when some tools fail."""

        async def ok_tool() -> str:
            return "success"

        async def bad_tool() -> str:
            msg = "oops"
            raise ValueError(msg)

        batch._dispatch["ok"] = ok_tool
        batch._dispatch["bad"] = bad_tool

        ctx = _mock_ctx()
        result = await batch.execute_batch(
            [
                {"tool": "ok", "args": {}},
                {"tool": "bad", "args": {}},
            ],
            ctx,
        )
        assert "success" in result
        assert "Erro" in result


class TestExecuteBatchStringArgs:
    def teardown_method(self) -> None:
        for key in ("feriados", "some_tool", "typed_tool"):
            batch._dispatch.pop(key, None)

    @pytest.mark.asyncio
    async def test_args_as_json_string(self) -> None:
        """Should accept args as JSON string and execute the tool normally."""

        async def tool(ano: int) -> str:
            return f"ano={ano}"

        batch._dispatch["feriados"] = tool
        ctx = _mock_ctx()
        result = await batch.execute_batch([{"tool": "feriados", "args": '{"ano": 2026}'}], ctx)
        assert "ano=2026" in result

    @pytest.mark.asyncio
    async def test_args_invalid_json_string_returns_error(self) -> None:
        """Invalid JSON string should return an error in the result, not raise an exception."""

        async def tool(x: int) -> str:
            return "ok"

        batch._dispatch["some_tool"] = tool
        ctx = _mock_ctx()
        result = await batch.execute_batch([{"tool": "some_tool", "args": "nao-e-json"}], ctx)
        assert "não é JSON válido" in result

    @pytest.mark.asyncio
    async def test_args_wrong_type_returns_error(self) -> None:
        """Invalid type (e.g. list) should return a descriptive error, not raise an exception."""

        async def tool(x: int) -> str:
            return "ok"

        batch._dispatch["typed_tool"] = tool
        ctx = _mock_ctx()
        result = await batch.execute_batch([{"tool": "typed_tool", "args": [1, 2, 3]}], ctx)
        assert "deve ser um objeto JSON" in result


def _real_registry() -> batch.FeatureRegistry:
    """Build a real registry from the project for integration testing."""
    from mcp_brasil._shared.feature import FeatureRegistry

    reg = FeatureRegistry()
    reg.discover("mcp_brasil.data")
    reg.discover("mcp_brasil.agentes")
    return reg
