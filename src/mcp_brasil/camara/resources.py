"""Resources for the Câmara feature — static reference data for LLM context."""

from __future__ import annotations

import json

from .constants import CAMARA_API_BASE


def tipos_proposicao() -> str:
    """Tipos de proposição legislativa da Câmara dos Deputados."""
    data = [
        {"sigla": "PL", "nome": "Projeto de Lei", "descricao": "Lei ordinária federal"},
        {
            "sigla": "PLP",
            "nome": "Projeto de Lei Complementar",
            "descricao": "Regulamenta disposições constitucionais",
        },
        {
            "sigla": "PEC",
            "nome": "Proposta de Emenda à Constituição",
            "descricao": "Altera a Constituição Federal",
        },
        {
            "sigla": "MPV",
            "nome": "Medida Provisória",
            "descricao": "Editada pelo Presidente, tem força de lei imediata",
        },
        {
            "sigla": "PDL",
            "nome": "Projeto de Decreto Legislativo",
            "descricao": "Matérias de competência exclusiva do Congresso",
        },
        {
            "sigla": "PRC",
            "nome": "Projeto de Resolução da Câmara",
            "descricao": "Regula matérias internas da Câmara",
        },
        {
            "sigla": "REQ",
            "nome": "Requerimento",
            "descricao": "Solicitações diversas (CPI, audiência, etc.)",
        },
        {"sigla": "INC", "nome": "Indicação", "descricao": "Sugestão a outro poder ou órgão"},
    ]
    return json.dumps(data, ensure_ascii=False)


def legislaturas_recentes() -> str:
    """Legislaturas recentes da Câmara dos Deputados."""
    data = [
        {
            "id": 57,
            "inicio": "2023-02-01",
            "fim": "2027-01-31",
            "descricao": "57ª Legislatura (2023-2027) — atual",
        },
        {
            "id": 56,
            "inicio": "2019-02-01",
            "fim": "2023-01-31",
            "descricao": "56ª Legislatura (2019-2023)",
        },
        {
            "id": 55,
            "inicio": "2015-02-01",
            "fim": "2019-01-31",
            "descricao": "55ª Legislatura (2015-2019)",
        },
        {
            "id": 54,
            "inicio": "2011-02-01",
            "fim": "2015-01-31",
            "descricao": "54ª Legislatura (2011-2015)",
        },
    ]
    return json.dumps(data, ensure_ascii=False)


def info_api() -> str:
    """Informações gerais sobre a API da Câmara dos Deputados."""
    data = {
        "nome": "API de Dados Abertos da Câmara dos Deputados",
        "url_base": CAMARA_API_BASE,
        "autenticacao": "Não requer autenticação",
        "documentacao": "https://dadosabertos.camara.leg.br/swagger/api.html",
        "formato": "JSON (envelope com campos 'dados' e 'links')",
        "paginacao": "Parâmetros 'pagina' e 'itens' (padrão: 15 itens por página)",
        "filtros_comuns": {
            "deputados": ["nome", "siglaPartido", "siglaUf", "idLegislatura"],
            "proposicoes": ["siglaTipo", "numero", "ano", "keywords"],
            "votacoes": ["dataInicio", "dataFim"],
            "eventos": ["dataInicio", "dataFim"],
        },
    }
    return json.dumps(data, ensure_ascii=False)
