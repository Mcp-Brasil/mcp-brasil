"""HTTP client for the PNCP API.

Endpoints:
    - /contratacoes/publicacao?q=...     → buscar_contratacoes
    - /contratos?q=...                   → buscar_contratos
    - /contratos?dataInicial=...         → buscar_contratos_por_data
    - /atas?q=...                        → buscar_atas
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import ATAS_URL, CONTRATACOES_URL, CONTRATOS_URL, DEFAULT_PAGE_SIZE
from .schemas import (
    AtaRegistroPreco,
    AtaResultado,
    Contratacao,
    ContratacaoResultado,
    Contrato,
    ContratoResultado,
)


def _parse_contratacao(item: dict[str, Any]) -> Contratacao:
    """Parse a raw API response item into a Contratacao model."""
    orgao = item.get("orgaoEntidade", {}) or {}
    return Contratacao(
        orgao_cnpj=orgao.get("cnpj"),
        orgao_nome=orgao.get("razaoSocial"),
        ano=item.get("anoCompra"),
        numero_sequencial=item.get("sequencialCompra"),
        numero_controle_pncp=item.get("numeroControlePNCP"),
        objeto=item.get("objetoCompra"),
        modalidade_id=item.get("modalidadeId"),
        modalidade_nome=item.get("modalidadeNome"),
        situacao_id=item.get("situacaoCompraId"),
        situacao_nome=item.get("situacaoCompraNome"),
        valor_estimado=item.get("valorTotalEstimado"),
        valor_homologado=item.get("valorTotalHomologado"),
        data_publicacao=item.get("dataPublicacaoPncp"),
        data_abertura=item.get("dataAberturaProposta"),
        uf=orgao.get("ufSigla"),
        municipio=orgao.get("municipioNome"),
        esfera=orgao.get("esferaNome"),
        link_pncp=item.get("linkPncp"),
    )


def _parse_contrato(item: dict[str, Any]) -> Contrato:
    """Parse a raw API response item into a Contrato model."""
    orgao = item.get("orgaoEntidade", {}) or {}
    fornecedor = item.get("fornecedor", {}) or {}
    return Contrato(
        orgao_cnpj=orgao.get("cnpj"),
        orgao_nome=orgao.get("razaoSocial"),
        numero_contrato=item.get("numeroContratoEmpenho"),
        objeto=item.get("objetoContrato"),
        fornecedor_cnpj=fornecedor.get("cnpj") or fornecedor.get("cpfCnpj"),
        fornecedor_nome=fornecedor.get("razaoSocial") or fornecedor.get("nomeRazaoSocial"),
        valor_inicial=item.get("valorInicial"),
        valor_final=item.get("valorFinal"),
        vigencia_inicio=item.get("dataVigenciaInicio"),
        vigencia_fim=item.get("dataVigenciaFim"),
        data_publicacao=item.get("dataPublicacaoPncp"),
        situacao=item.get("nomeStatus"),
    )


def _parse_ata(item: dict[str, Any]) -> AtaRegistroPreco:
    """Parse a raw API response item into an AtaRegistroPreco model."""
    orgao = item.get("orgaoEntidade", {}) or {}
    fornecedor = item.get("fornecedor", {}) or {}
    return AtaRegistroPreco(
        orgao_cnpj=orgao.get("cnpj"),
        orgao_nome=orgao.get("razaoSocial"),
        numero_ata=item.get("numeroAta") or item.get("numeroAtaRegistroPreco"),
        objeto=item.get("objetoAta") or item.get("objetoContrato"),
        fornecedor_cnpj=fornecedor.get("cnpj") or fornecedor.get("cpfCnpj"),
        fornecedor_nome=fornecedor.get("razaoSocial") or fornecedor.get("nomeRazaoSocial"),
        valor_total=item.get("valorTotal") or item.get("valorInicial"),
        vigencia_inicio=item.get("dataVigenciaInicio"),
        vigencia_fim=item.get("dataVigenciaFim"),
        situacao=item.get("nomeStatus"),
    )


async def buscar_contratacoes(
    query: str,
    cnpj_orgao: str | None = None,
    data_inicial: str | None = None,
    data_final: str | None = None,
    pagina: int = 1,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> ContratacaoResultado:
    """Search published procurement processes."""
    params: dict[str, str] = {
        "q": query,
        "pagina": str(pagina),
        "tamanhoPagina": str(tamanho),
    }
    if cnpj_orgao:
        params["cnpjOrgao"] = cnpj_orgao
    if data_inicial:
        params["dataInicial"] = data_inicial
    if data_final:
        params["dataFinal"] = data_final

    data: dict[str, Any] = await http_get(CONTRATACOES_URL, params=params)
    items = data.get("data", data.get("resultado", []))
    contratacoes = [_parse_contratacao(item) for item in items] if isinstance(items, list) else []
    return ContratacaoResultado(
        total=data.get("totalRegistros", data.get("count", len(contratacoes))),
        contratacoes=contratacoes,
    )


async def buscar_contratos(
    query: str | None = None,
    cnpj_fornecedor: str | None = None,
    cnpj_orgao: str | None = None,
    data_inicial: str | None = None,
    data_final: str | None = None,
    pagina: int = 1,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> ContratoResultado:
    """Search public contracts."""
    params: dict[str, str] = {
        "pagina": str(pagina),
        "tamanhoPagina": str(tamanho),
    }
    if query:
        params["q"] = query
    if cnpj_fornecedor:
        params["cnpjFornecedor"] = cnpj_fornecedor
    if cnpj_orgao:
        params["cnpjOrgao"] = cnpj_orgao
    if data_inicial:
        params["dataInicial"] = data_inicial
    if data_final:
        params["dataFinal"] = data_final

    data: dict[str, Any] = await http_get(CONTRATOS_URL, params=params)
    items = data.get("data", data.get("resultado", []))
    contratos = [_parse_contrato(item) for item in items] if isinstance(items, list) else []
    return ContratoResultado(
        total=data.get("totalRegistros", data.get("count", len(contratos))),
        contratos=contratos,
    )


async def buscar_atas(
    query: str | None = None,
    cnpj_orgao: str | None = None,
    data_inicial: str | None = None,
    data_final: str | None = None,
    pagina: int = 1,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> AtaResultado:
    """Search price record minutes (atas de registro de preço)."""
    params: dict[str, str] = {
        "pagina": str(pagina),
        "tamanhoPagina": str(tamanho),
    }
    if query:
        params["q"] = query
    if cnpj_orgao:
        params["cnpjOrgao"] = cnpj_orgao
    if data_inicial:
        params["dataInicial"] = data_inicial
    if data_final:
        params["dataFinal"] = data_final

    data: dict[str, Any] = await http_get(ATAS_URL, params=params)
    items = data.get("data", data.get("resultado", []))
    atas = [_parse_ata(item) for item in items] if isinstance(items, list) else []
    return AtaResultado(
        total=data.get("totalRegistros", data.get("count", len(atas))),
        atas=atas,
    )
