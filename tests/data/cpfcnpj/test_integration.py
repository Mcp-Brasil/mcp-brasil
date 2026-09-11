"""Testes de integração da feature CPF.CNPJ (fastmcp.Client in-process).

Verificam o registro das 6 tools e o comportamento de feature oculta sem token,
usando a API real de FeatureMeta / FeatureRegistry.
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil._shared.feature import FeatureRegistry
from mcp_brasil.data.cpfcnpj import FEATURE_META
from mcp_brasil.data.cpfcnpj.schemas import EmpresaCnpj, Saldo
from mcp_brasil.data.cpfcnpj.server import mcp

CLIENT = "mcp_brasil.data.cpfcnpj.client"


class TestToolsRegistradas:
    @pytest.mark.asyncio
    async def test_todas_as_6_tools(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            esperado = {
                "validar_documento",
                "consultar_cnpj",
                "situacao_cnpj",
                "qsa_cnpj",
                "inscricoes_estaduais",
                "saldo",
            }
            assert esperado.issubset(names), f"Faltando: {esperado - names}"
            assert len(esperado) == 6

    @pytest.mark.asyncio
    async def test_tools_tem_docstring(self) -> None:
        async with Client(mcp) as c:
            for tool in await c.list_tools():
                assert tool.description, f"Tool {tool.name} sem descrição"


class TestFeatureOcultaSemToken:
    def test_is_auth_available_sem_token(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv("CPFCNPJ_TOKEN", raising=False)
        assert FEATURE_META.requires_auth is True
        assert FEATURE_META.auth_env_var == "CPFCNPJ_TOKEN"
        assert FEATURE_META.is_auth_available() is False

    def test_is_auth_available_com_token(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("CPFCNPJ_TOKEN", "algum-token")
        assert FEATURE_META.is_auth_available() is True

    def test_registry_pula_sem_token(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv("CPFCNPJ_TOKEN", raising=False)
        reg = FeatureRegistry()
        reg.discover("mcp_brasil.data")
        assert "cpfcnpj" not in reg.features
        assert "cpfcnpj" in reg.skipped
        assert "CPFCNPJ_TOKEN" in reg.skipped["cpfcnpj"]

    def test_registry_monta_com_token(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("CPFCNPJ_TOKEN", "algum-token")
        reg = FeatureRegistry()
        reg.discover("mcp_brasil.data")
        assert "cpfcnpj" in reg.features


class TestExecucaoTools:
    @pytest.mark.asyncio
    async def test_validar_documento_e2e(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("validar_documento", {"documento": "11.222.333/0001-81"})
            assert "CNPJ" in result.data
            assert "11222333000181" in result.data

    @pytest.mark.asyncio
    async def test_consultar_cnpj_e2e(self) -> None:
        emp = EmpresaCnpj(status=1, cnpj="11.222.333/0001-81", razao="EMPRESA EXEMPLO LTDA")
        with patch(f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=emp):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_cnpj", {"cnpj": "11222333000181"})
                assert "EMPRESA EXEMPLO LTDA" in result.data

    @pytest.mark.asyncio
    async def test_saldo_e2e(self) -> None:
        with patch(
            f"{CLIENT}.saldo",
            new_callable=AsyncMock,
            return_value=Saldo(id=5, nome="CNPJ B", saldo=123),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("saldo", {"pacote": 5})
                assert "123" in result.data
