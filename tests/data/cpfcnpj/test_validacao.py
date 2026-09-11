"""Testes de validação e normalização local de CPF/CNPJ.

Casos verificados no relatório de contrato da API (04 §7).
"""

import pytest

from mcp_brasil.data.cpfcnpj.validacao import (
    classificar_documento,
    normalizar_cnpj,
    normalizar_cpf,
    validar_cnpj,
    validar_cpf,
)


class TestValidarCpf:
    @pytest.mark.parametrize(
        "cpf,esperado",
        [
            ("11144477735", True),
            ("111.444.777-35", True),
            ("12345678900", False),
            ("00000000000", False),
            ("11111111111", False),
            ("1114447773", False),  # 10 dígitos
            ("111 444 777 35", False),  # espaço interno
            (" 11144477735", True),  # espaço à esquerda tolerado (bordas)
            ("11144477735\n", True),  # quebra de linha à direita tolerada
            ("111444777 35", False),  # espaço interno rejeita
            ("", False),
        ],
    )
    def test_validar_cpf(self, cpf: str, esperado: bool) -> None:
        assert validar_cpf(cpf) is esperado


class TestValidarCnpj:
    @pytest.mark.parametrize(
        "cnpj,esperado",
        [
            ("27272134000118", True),  # numérico (ALAS)
            ("11222333000181", True),  # numérico
            ("11.222.333/0001-81", True),  # com máscara padrão
            ("12ABC34501DE35", True),  # alfanumérico (exemplo da doc)
            ("12.ABC.345/01DE-35", True),  # alfanumérico com máscara
            ("99999999999999", False),  # DV numérico errado
            ("12ABC34501DE99", False),  # DV alfanumérico errado
            ("33 000 167 0001 01", False),  # espaço interno
            ("11222333000181 ", True),  # espaço à direita tolerado (bordas)
            (" 11222333000181", True),  # espaço à esquerda tolerado (bordas)
            ("11.222.333/0001-81\n", True),  # quebra de linha à direita tolerada
            ("11222 333000181", False),  # espaço interno rejeita
            ("AAAAAAAAAAAAAA", False),  # sequência repetida
            ("112223330001", False),  # 12 posições
            ("", False),
        ],
    )
    def test_validar_cnpj(self, cnpj: str, esperado: bool) -> None:
        assert validar_cnpj(cnpj) is esperado


class TestNormalizacao:
    def test_normalizar_cnpj_preserva_letras_maiusculas(self) -> None:
        assert normalizar_cnpj("12.abc.345/01de-35") == "12ABC34501DE35"

    def test_normalizar_cnpj_numerico(self) -> None:
        assert normalizar_cnpj("11.222.333/0001-81") == "11222333000181"

    def test_normalizar_cpf_apenas_digitos(self) -> None:
        assert normalizar_cpf("111.444.777-35") == "11144477735"


class TestClassificarDocumento:
    @pytest.mark.parametrize(
        "documento,tipo",
        [
            ("11144477735", "cpf"),
            ("111.444.777-35", "cpf"),
            ("11222333000181", "cnpj"),
            ("12ABC34501DE35", "cnpj"),
            ("12.ABC.345/01DE-35", "cnpj"),
            ("123", "desconhecido"),
            ("", "desconhecido"),
        ],
    )
    def test_classificar(self, documento: str, tipo: str) -> None:
        assert classificar_documento(documento) == tipo
