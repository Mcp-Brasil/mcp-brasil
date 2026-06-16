"""Testes do normalizador de argumentos string-JSON (issue #8)."""

import json

from mcp_brasil._shared.tolerant_args import normalize_json_string_args


class TestExecutarLoteConsultas:
    def test_json_string_vira_lista(self) -> None:
        consultas = [{"tool": "camara_despesas_deputado", "args": {"ano": 2024}}]
        args = {"consultas": json.dumps(consultas)}
        normalize_json_string_args("executar_lote", args)
        assert args["consultas"] == consultas

    def test_lista_permanece_intacta(self) -> None:
        consultas = [{"tool": "x", "args": {}}]
        args = {"consultas": consultas}
        normalize_json_string_args("executar_lote", args)
        assert args["consultas"] is consultas

    def test_json_object_nao_e_coagido(self) -> None:
        # string JSON que parseia para dict (não list) -> não altera
        args = {"consultas": json.dumps({"tool": "x"})}
        normalize_json_string_args("executar_lote", args)
        assert isinstance(args["consultas"], str)


class TestCallToolArguments:
    def test_json_string_vira_dict(self) -> None:
        inner = {"codigo": 35, "ano": 2024}
        args = {"name": "ibge_listar_estados", "arguments": json.dumps(inner)}
        normalize_json_string_args("call_tool", args)
        assert args["arguments"] == inner

    def test_dict_permanece_intacto(self) -> None:
        inner = {"a": 1}
        args = {"name": "t", "arguments": inner}
        normalize_json_string_args("call_tool", args)
        assert args["arguments"] is inner

    def test_json_array_nao_e_coagido(self) -> None:
        # string JSON que parseia para list (não dict) -> não altera
        args = {"name": "t", "arguments": json.dumps([1, 2])}
        normalize_json_string_args("call_tool", args)
        assert isinstance(args["arguments"], str)


class TestSeguranca:
    def test_json_invalido_intacto(self) -> None:
        args = {"consultas": "not json {"}
        normalize_json_string_args("executar_lote", args)
        assert args["consultas"] == "not json {"

    def test_tool_fora_da_allowlist_intacta(self) -> None:
        args = {"arguments": json.dumps({"a": 1})}
        normalize_json_string_args("alguma_outra_tool", args)
        assert isinstance(args["arguments"], str)

    def test_campo_fora_da_allowlist_intacto(self) -> None:
        # string JSON legítima num campo não-allowlisted da tool allowlisted
        args = {"consultas": [{"tool": "x"}], "outro": json.dumps([1, 2])}
        normalize_json_string_args("executar_lote", args)
        assert isinstance(args["outro"], str)

    def test_arguments_none_nao_levanta(self) -> None:
        normalize_json_string_args("executar_lote", None)

    def test_arguments_vazio_nao_levanta(self) -> None:
        args: dict[str, object] = {}
        normalize_json_string_args("executar_lote", args)
        assert args == {}
