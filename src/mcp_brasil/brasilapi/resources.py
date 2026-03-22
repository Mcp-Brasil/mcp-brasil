"""Static reference data for the BrasilAPI feature.

Resources are read-only data sources that clients can pull.
They provide context to LLMs without requiring tool calls.
"""

from __future__ import annotations

import json

from .constants import TAXAS_CONHECIDAS


def taxas_disponiveis() -> str:
    """Catálogo de taxas e índices oficiais disponíveis para consulta."""
    data = [{"sigla": k, "descricao": v} for k, v in TAXAS_CONHECIDAS.items()]
    return json.dumps(data, ensure_ascii=False, indent=2)


def tipos_veiculo_fipe() -> str:
    """Tipos de veículo aceitos nas consultas FIPE."""
    data = [
        {"tipo": "carros", "descricao": "Automóveis e utilitários"},
        {"tipo": "caminhoes", "descricao": "Caminhões e micro-ônibus"},
        {"tipo": "motos", "descricao": "Motocicletas e scooters"},
    ]
    return json.dumps(data, ensure_ascii=False, indent=2)


def endpoints_brasilapi() -> str:
    """Catálogo de todos os endpoints disponíveis na BrasilAPI."""
    data = [
        {"endpoint": "CEP", "descricao": "Consulta endereço por CEP", "auth": False},
        {"endpoint": "CNPJ", "descricao": "Consulta empresa por CNPJ", "auth": False},
        {"endpoint": "DDD", "descricao": "Cidades e estado por DDD", "auth": False},
        {"endpoint": "Bancos", "descricao": "Lista de bancos brasileiros", "auth": False},
        {"endpoint": "Câmbio", "descricao": "Moedas e cotações", "auth": False},
        {"endpoint": "Feriados", "descricao": "Feriados nacionais por ano", "auth": False},
        {"endpoint": "Taxas", "descricao": "SELIC, CDI, IPCA e outras", "auth": False},
        {"endpoint": "FIPE", "descricao": "Tabelas, marcas e veículos", "auth": False},
        {"endpoint": "ISBN", "descricao": "Consulta livro por ISBN", "auth": False},
        {"endpoint": "NCM", "descricao": "Nomenclatura Comum do Mercosul", "auth": False},
        {"endpoint": "PIX", "descricao": "Participantes do sistema PIX", "auth": False},
        {"endpoint": "Registro.br", "descricao": "Disponibilidade de domínio .br", "auth": False},
    ]
    return json.dumps(data, ensure_ascii=False, indent=2)
