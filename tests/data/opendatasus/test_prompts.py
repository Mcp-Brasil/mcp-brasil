"""Tests for the OpenDataSUS prompts."""

from mcp_brasil.data.opendatasus.prompts import pesquisa_epidemiologica


class TestPesquisaEpidemiologica:
    def test_includes_tema(self) -> None:
        result = pesquisa_epidemiologica("dengue")
        assert "dengue" in result

    def test_includes_tool_names(self) -> None:
        result = pesquisa_epidemiologica("vacinação")
        assert "buscar_datasets" in result
        assert "detalhar_dataset" in result
        assert "consultar_datastore" in result

    def test_includes_instructions(self) -> None:
        result = pesquisa_epidemiologica("srag")
        assert "pesquisador" in result.lower() or "saúde" in result.lower()
