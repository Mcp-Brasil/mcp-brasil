"""HTTP client for the BrasilAPI.

Endpoints:
    - /cep/v1/{cep}                         → consultar_cep
    - /cnpj/v1/{cnpj}                       → consultar_cnpj
    - /ddd/v1/{ddd}                         → consultar_ddd
    - /banks/v1                             → listar_bancos
    - /banks/v1/{code}                      → consultar_banco
    - /cambio/v1/moedas                     → listar_moedas
    - /cambio/v1/cotacao/{moeda}/{data}     → consultar_cotacao
    - /feriados/v1/{ano}                    → consultar_feriados
    - /taxas/v1/{sigla}                     → consultar_taxa
    - /fipe/tabelas/v1                      → listar_tabelas_fipe
    - /fipe/marcas/v1/{tipo}                → listar_marcas_fipe
    - /fipe/veiculos/v1/{tipo}/{marca}      → buscar_veiculos_fipe
    - /isbn/v1/{isbn}                       → consultar_isbn
    - /ncm/v1?search={busca}                → buscar_ncm
    - /ncm/v1/{codigo}                      → consultar_ncm
    - /pix/v1/participants                  → listar_pix_participantes
    - /registrobr/v1/{dominio}              → consultar_registro_br
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import (
    BANKS_URL,
    CAMBIO_COTACAO_URL,
    CAMBIO_MOEDAS_URL,
    CEP_URL,
    CNPJ_URL,
    DDD_URL,
    FERIADOS_URL,
    FIPE_MARCAS_URL,
    FIPE_TABELAS_URL,
    FIPE_VEICULOS_URL,
    ISBN_URL,
    NCM_URL,
    PIX_URL,
    REGISTRO_BR_URL,
    TAXAS_URL,
)
from .schemas import (
    Banco,
    Cotacao,
    DddInfo,
    EmpresaCnpj,
    Endereco,
    Feriado,
    FipeMarca,
    FipeTabela,
    FipeVeiculo,
    Livro,
    Moeda,
    NcmItem,
    PixParticipante,
    RegistroBrDominio,
    TaxaOficial,
)


async def consultar_cep(cep: str) -> Endereco:
    """Fetch address data for a CEP."""
    clean = cep.replace("-", "").strip()
    data: dict[str, Any] = await http_get(f"{CEP_URL}/{clean}")
    return Endereco(**data)


async def consultar_cnpj(cnpj: str) -> EmpresaCnpj:
    """Fetch company data for a CNPJ."""
    clean = cnpj.replace(".", "").replace("/", "").replace("-", "").strip()
    data: dict[str, Any] = await http_get(f"{CNPJ_URL}/{clean}")
    return EmpresaCnpj(**data)


async def consultar_ddd(ddd: str) -> DddInfo:
    """Fetch cities and state for a DDD area code."""
    data: dict[str, Any] = await http_get(f"{DDD_URL}/{ddd.strip()}")
    return DddInfo(**data)


async def listar_bancos() -> list[Banco]:
    """Fetch all Brazilian banks."""
    data: list[dict[str, Any]] = await http_get(BANKS_URL)
    return [Banco(**b) for b in data]


async def consultar_banco(codigo: int) -> Banco:
    """Fetch a specific bank by code."""
    data: dict[str, Any] = await http_get(f"{BANKS_URL}/{codigo}")
    return Banco(**data)


async def listar_moedas() -> list[Moeda]:
    """Fetch available exchange currencies."""
    data: list[dict[str, Any]] = await http_get(CAMBIO_MOEDAS_URL)
    return [
        Moeda(
            simbolo=m.get("simbolo", ""),
            nome_formatado=m.get("nomeFormatado", ""),
            tipo_moeda=m.get("tipoMoeda"),
        )
        for m in data
    ]


async def consultar_cotacao(moeda: str, data_ref: str) -> Cotacao:
    """Fetch exchange rate for a currency on a specific date."""
    url = f"{CAMBIO_COTACAO_URL}/{moeda.upper()}/{data_ref}"
    raw: dict[str, Any] = await http_get(url)
    cotacoes = raw.get("cotacoes", [])
    cotacao = cotacoes[0] if cotacoes else {}
    return Cotacao(
        moeda=raw.get("moeda", moeda),
        data=raw.get("data", data_ref),
        valor_compra=cotacao.get("cotacao_compra") or cotacao.get("valor_compra"),
        valor_venda=cotacao.get("cotacao_venda") or cotacao.get("valor_venda"),
    )


async def consultar_feriados(ano: int) -> list[Feriado]:
    """Fetch national holidays for a given year."""
    data: list[dict[str, Any]] = await http_get(f"{FERIADOS_URL}/{ano}")
    return [Feriado(**f) for f in data]


async def consultar_taxa(sigla: str) -> TaxaOficial:
    """Fetch an official economic rate by abbreviation."""
    data: dict[str, Any] = await http_get(f"{TAXAS_URL}/{sigla.upper()}")
    return TaxaOficial(nome=data.get("nome", sigla), valor=data.get("valor"))


async def listar_tabelas_fipe() -> list[FipeTabela]:
    """Fetch FIPE reference tables."""
    data: list[dict[str, Any]] = await http_get(FIPE_TABELAS_URL)
    return [FipeTabela(**t) for t in data]


async def listar_marcas_fipe(
    tipo_veiculo: str, tabela_referencia: int | None = None
) -> list[FipeMarca]:
    """Fetch vehicle brands from FIPE by vehicle type."""
    url = f"{FIPE_MARCAS_URL}/{tipo_veiculo}"
    params: dict[str, str] = {}
    if tabela_referencia is not None:
        params["tabela_referencia"] = str(tabela_referencia)
    data: list[dict[str, Any]] = await http_get(url, params=params or None)
    return [FipeMarca(**m) for m in data]


async def buscar_veiculos_fipe(
    tipo_veiculo: str, codigo_marca: str, tabela_referencia: int | None = None
) -> list[FipeVeiculo]:
    """Fetch vehicles from FIPE by type and brand."""
    url = f"{FIPE_VEICULOS_URL}/{tipo_veiculo}/{codigo_marca}"
    params: dict[str, str] = {}
    if tabela_referencia is not None:
        params["tabela_referencia"] = str(tabela_referencia)
    data: list[dict[str, Any]] = await http_get(url, params=params or None)
    return [FipeVeiculo(**v) for v in data]


async def consultar_isbn(isbn: str) -> Livro:
    """Fetch book data by ISBN."""
    clean = isbn.replace("-", "").strip()
    data: dict[str, Any] = await http_get(f"{ISBN_URL}/{clean}")
    return Livro(**data)


async def buscar_ncm(busca: str) -> list[NcmItem]:
    """Search NCM codes by description or code."""
    data: list[dict[str, Any]] = await http_get(NCM_URL, params={"search": busca})
    return [NcmItem(**n) for n in data]


async def consultar_ncm(codigo: str) -> NcmItem:
    """Fetch a specific NCM code."""
    clean = codigo.replace(".", "").strip()
    data: dict[str, Any] = await http_get(f"{NCM_URL}/{clean}")
    return NcmItem(**data)


async def listar_pix_participantes() -> list[PixParticipante]:
    """Fetch all PIX participants."""
    data: list[dict[str, Any]] = await http_get(PIX_URL)
    return [PixParticipante(**p) for p in data]


async def consultar_registro_br(dominio: str) -> RegistroBrDominio:
    """Check domain availability at Registro.br."""
    data: dict[str, Any] = await http_get(f"{REGISTRO_BR_URL}/{dominio}")
    return RegistroBrDominio(**data)
