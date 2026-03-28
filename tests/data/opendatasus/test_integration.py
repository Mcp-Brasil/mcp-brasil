"""Integration tests for the OpenDataSUS feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.opendatasus.schemas import DatasetOpenDataSUS, RecursoDataset
from mcp_brasil.data.opendatasus.server import mcp

CLIENT_MODULE = "mcp_brasil.data.opendatasus.client"


@pytest.fixture
def opendatasus_client() -> Client:
    return Client(mcp)


MOCK_DATASET = DatasetOpenDataSUS(
    id="abc-123",
    nome="test-dataset",
    titulo="Test Dataset",
    organizacao="MS",
    tags=["test"],
    recursos=[RecursoDataset(id="r1", nome="data.csv", formato="CSV")],
    total_recursos=1,
)


class TestOpenDataSUSIntegration:
    @pytest.mark.asyncio
    async def test_server_has_5_tools(self, opendatasus_client: Client) -> None:
        async with opendatasus_client:
            tools = await opendatasus_client.list_tools()
            assert len(tools) == 5

    @pytest.mark.asyncio
    async def test_server_has_2_resources(self, opendatasus_client: Client) -> None:
        async with opendatasus_client:
            resources = await opendatasus_client.list_resources()
            assert len(resources) == 2

    @pytest.mark.asyncio
    async def test_server_has_1_prompt(self, opendatasus_client: Client) -> None:
        async with opendatasus_client:
            prompts = await opendatasus_client.list_prompts()
            assert len(prompts) == 1

    @pytest.mark.asyncio
    async def test_buscar_datasets_tool(self, opendatasus_client: Client) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_datasets",
            new_callable=AsyncMock,
            return_value=([MOCK_DATASET], 1),
        ):
            async with opendatasus_client:
                result = await opendatasus_client.call_tool("buscar_datasets", {"query": "leitos"})
                assert "Test Dataset" in result.data

    @pytest.mark.asyncio
    async def test_listar_conhecidos_tool(self, opendatasus_client: Client) -> None:
        async with opendatasus_client:
            result = await opendatasus_client.call_tool("listar_datasets_conhecidos", {})
            assert "hospitais-leitos" in result.data

    @pytest.mark.asyncio
    async def test_read_datasets_resource(self, opendatasus_client: Client) -> None:
        async with opendatasus_client:
            resources = await opendatasus_client.list_resources()
            ds_uri = next(r for r in resources if "datasets" in str(r.uri)).uri
            content = await opendatasus_client.read_resource(ds_uri)
            text = content[0].text if hasattr(content[0], "text") else str(content[0])
            assert "hospitais-leitos" in text

    @pytest.mark.asyncio
    async def test_get_prompt(self, opendatasus_client: Client) -> None:
        async with opendatasus_client:
            result = await opendatasus_client.get_prompt(
                "pesquisa_epidemiologica", {"tema": "dengue"}
            )
            text = result.messages[0].content.text
            assert "dengue" in text
