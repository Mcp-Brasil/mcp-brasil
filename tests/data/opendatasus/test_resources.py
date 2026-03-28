"""Tests for the OpenDataSUS resources."""

import json

from mcp_brasil.data.opendatasus import resources


class TestDatasetsConhecidos:
    def test_returns_json(self) -> None:
        result = resources.datasets_conhecidos()
        data = json.loads(result)
        assert isinstance(data, list)
        assert len(data) == 5

    def test_has_required_fields(self) -> None:
        result = resources.datasets_conhecidos()
        data = json.loads(result)
        for item in data:
            assert "nome" in item
            assert "titulo" in item
            assert "descricao" in item


class TestTagsComuns:
    def test_returns_json(self) -> None:
        result = resources.tags_comuns()
        data = json.loads(result)
        assert isinstance(data, list)
        assert len(data) >= 10

    def test_contains_known_tags(self) -> None:
        result = resources.tags_comuns()
        data = json.loads(result)
        assert "covid-19" in data
        assert "leitos" in data
