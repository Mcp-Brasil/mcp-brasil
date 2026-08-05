"""HTTP client for the TCE-CE API.

Endpoints:
    - /municipios     → listar_municipios
    - /licitacoes     → buscar_licitacoes
    - /contrato       → buscar_contratos
    - /notas_empenhos → buscar_empenhos
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import (
    CONTRATOS_URL,
    DEFAULT_QUANTIDADE,
    EMPENHOS_URL,
    LICITACOES_URL,
    MUNICIPIOS_URL,
)
from .schemas import (
    Contrato,
    ContratoResultado,
    Empenho,
    EmpenhoResultado,
    Licitacao,
    Municipio,
)


async def listar_municipios() -> list[Municipio]:
    """Lista todos os municípios cearenses."""
    raw: dict[str, Any] = await http_get(MUNICIPIOS_URL)
    items = raw.get("elements", [])
    return [
        Municipio(
            codigo_municipio=str(item.get("codigo_municipio", "")),
            nome_municipio=item.get("nome_municipio"),
        )
        for item in items
    ]


async def buscar_licitacoes(
    *,
    codigo_municipio: str,
    data_realizacao: str,
) -> list[Licitacao]:
    """Busca licitações de um município cearense.

    Args:
        codigo_municipio: Código do município (ex: "057" para Fortaleza).
        data_realizacao: Data ou intervalo (yyyy-mm-dd ou yyyy-mm-dd_yyyy-mm-dd).
    """
    if "_" in data_realizacao:
        data_inicio, data_fim = data_realizacao.split("_", 1)
    else:
        data_inicio = data_fim = data_realizacao

    params: dict[str, Any] = {
        "codigo_municipio": codigo_municipio,
        "data_inicio": data_inicio,
        "data_fim": data_fim,
    }
    raw: dict[str, Any] = await http_get(LICITACOES_URL, params=params)
    items = raw.get("elements", [])
    return [
        Licitacao(
            codigo_municipio=str(item.get("codigo_municipio", "")),
            numero_licitacao=item.get("numero_licitacao"),
            data_realizacao=item.get("data_realizacao_autuacao_licitacao"),
            modalidade_licitacao=item.get("modalidade_licitacao"),
            objeto=(item.get("descricao1_objeto_licitacao") or "")
            + (item.get("descricao2_objeto_licitacao") or ""),
            valor_orcado_estimado=item.get("valor_orcado_estimado"),
            data_homologacao=item.get("data_homologacao"),
            nome_responsavel_homologacao=item.get("nome_responsavel_homologacao"),
            modalidade_processo_administrativo=item.get("modalidade_processo_administrativo"),
        )
        for item in items
    ]


async def buscar_contratos(
    *,
    codigo_municipio: str,
    data_contrato: str,
    quantidade: int = DEFAULT_QUANTIDADE,
    deslocamento: int = 0,
) -> ContratoResultado:
    """Busca contratos de um município cearense.

    Args:
        codigo_municipio: Código do município.
        data_contrato: Data ou intervalo (yyyy-mm-dd ou yyyy-mm-dd_yyyy-mm-dd).
        quantidade: Página (max 100).
        deslocamento: Offset.
    """
    params: dict[str, Any] = {
        "codigo_municipio": codigo_municipio,
        "data_contrato": data_contrato,
        "$start_index": deslocamento,
    }
    raw: dict[str, Any] = await http_get(CONTRATOS_URL, params=params)
    items = raw.get("elements", [])
    total = raw.get("total", len(items))
    contratos = [
        Contrato(
            codigo_municipio=str(item.get("codigo_municipio", "")),
            numero_contrato=item.get("numero_contrato"),
            data_contrato=item.get("data_contrato"),
            tipo_contrato=item.get("tipo_contrato"),
            modalidade_contrato=item.get("modalidade_contrato"),
            objeto=item.get("descricao_objeto_contrato"),
            valor_total_contrato=item.get("valor_total_contrato"),
            data_inicio_vigencia=item.get("data_inicio_vigencia_contrato"),
            data_fim_vigencia=item.get("data_fim_vigencia_contrato"),
        )
        for item in items
    ]
    return ContratoResultado(contratos=contratos, total=total)


async def buscar_empenhos(
    *,
    codigo_municipio: int,
    data_referencia: int,
    codigo_orgao: str,
    quantidade: int = DEFAULT_QUANTIDADE,
    deslocamento: int = 0,
) -> EmpenhoResultado:
    """Busca notas de empenho de um município cearense.

    Args:
        codigo_municipio: Código numérico do município.
        data_referencia: Ano-mês no formato yyyymm (ex: 202401).
        codigo_orgao: Código do órgão (ex: "01" = Câmara, "02" = Prefeitura).
        quantidade: Página (max 100).
        deslocamento: Offset.
    """
    params: dict[str, Any] = {
        "codigo_municipio": codigo_municipio,
        "data_referencia_empenho": data_referencia,
        "codigo_orgao": codigo_orgao,
        "$start_index": deslocamento,
    }
    raw: dict[str, Any] = await http_get(EMPENHOS_URL, params=params)
    items = raw.get("elements", [])
    total = raw.get("total", len(items))
    empenhos = [
        Empenho(
            codigo_municipio=item.get("codigo_municipio"),
            numero_empenho=item.get("numero_empenho"),
            data_emissao=item.get("data_emissao_empenho"),
            valor_empenho=item.get("valor_empenho"),
            nome_negociante=item.get("nome_negociante"),
            numero_documento_negociante=item.get("numero_documento_negociante"),
            historico=(item.get("historico1_empenho") or "")
            + (item.get("historico2_empenho") or ""),
            codigo_orgao=item.get("codigo_orgao"),
            codigo_funcao=item.get("codigo_funcao"),
        )
        for item in items
    ]
    return EmpenhoResultado(empenhos=empenhos, total=total)
