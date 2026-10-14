"""Static reference data for the TCE-CE feature."""

from __future__ import annotations

import json


def endpoints_tce_ce() -> str:
    """Endpoints usados da API de Dados Abertos do TCE-CE (SIM 2.0).

    Base: https://api-dados-abertos.tce.ce.gov.br/sim (Swagger na mesma URL).
    A API completa tem cerca de 100 rotas (folha, RPPS, obras, veículos, etc.).
    """
    endpoints = [
        {
            "endpoint": "/municipios",
            "descricao": "184 municípios cearenses com código SIM e IBGE",
            "parametros_obrigatorios": [],
        },
        {
            "endpoint": "/processos_administrativos_contratacoes",
            "descricao": "Licitações, dispensas e adesões (tool buscar_licitacoes_ce)",
            "parametros_obrigatorios": [
                "codigo_municipio",
                "data_inicio (yyyyMMdd)",
                "data_fim (yyyyMMdd)",
            ],
        },
        {
            "endpoint": "/contratos",
            "descricao": "Contratos e aditivos (não traz o contratado)",
            "parametros_obrigatorios": [
                "codigo_municipio",
                "data_inicio (yyyyMMdd)",
                "data_fim (yyyyMMdd)",
            ],
        },
        {
            "endpoint": "/contratados",
            "descricao": "Contratado de cada contrato (nome e CNPJ/CPF)",
            "parametros_obrigatorios": [
                "codigo_municipio",
                "data_inicio (yyyyMMdd)",
                "data_fim (yyyyMMdd)",
            ],
        },
        {
            "endpoint": "/notas_empenhos",
            "descricao": "Notas de empenho com credor",
            "parametros_obrigatorios": [
                "codigo_municipio",
                "exercicio_orcamento (AAAA00)",
                "data_referencia_doc (AAAAMM)",
            ],
        },
    ]
    return json.dumps(
        {
            "base": "https://api-dados-abertos.tce.ce.gov.br/sim",
            "paginacao": "$count (max 1000) e $start_index",
            "restricao": "acesso apenas de IP brasileiro",
            "endpoints": endpoints,
        },
        ensure_ascii=False,
        indent=2,
    )
