"""HTTP client for the TCU API.

Endpoints:
    - Acórdãos: GET /acordao/recupera-acordaos
    - Inabilitados: GET /condenacao/consulta/inabilitados
    - Inidôneos: GET /condenacao/consulta/inidoneos
    - Certidões APF: GET /certidoes/{cnpj}
    - Pedidos do Congresso: GET /scn/pedidos_congresso
    - Cálculo de Débito: POST /calculadora/calcular-saldos-debito
    - Termos Contratuais: GET /termos-contratuais
    - CADIRREG: GET /recuperapessoacadirreg/{cpf}
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get, http_post

from .constants import (
    ACORDAOS_URL,
    CADIRREG_URL,
    CALCULAR_DEBITO_URL,
    CERTIDOES_URL,
    INABILITADOS_URL,
    INIDONEOS_URL,
    PEDIDOS_CONGRESSO_URL,
    TERMOS_CONTRATUAIS_URL,
)
from .schemas import (
    Acordao,
    CertidaoConsolidada,
    Inabilitado,
    Inidoneo,
    ParcelaDebito,
    PedidoCongresso,
    PessoaCadirreg,
    ResultadoDebito,
    TermoContratual,
)


async def consultar_acordaos(
    inicio: int = 0,
    quantidade: int = 10,
) -> list[Acordao]:
    """Fetch acórdãos (decisões colegiadas) do TCU.

    API: GET https://dados-abertos.apps.tcu.gov.br/api/acordao/recupera-acordaos

    Args:
        inicio: Índice inicial dos resultados (paginação).
        quantidade: Quantidade de registros a retornar.
    """
    params: dict[str, Any] = {
        "inicio": inicio,
        "quantidade": quantidade,
    }
    data: list[dict[str, Any]] = await http_get(ACORDAOS_URL, params=params)
    return [Acordao(**item) for item in data]


async def consultar_inabilitados(
    cpf: str | None = None,
    offset: int = 0,
    limit: int = 25,
) -> list[Inabilitado]:
    """Fetch pessoas inabilitadas para função pública pelo TCU.

    API: GET https://contas.tcu.gov.br/ords/condenacao/consulta/inabilitados[/{CPF}]

    Args:
        cpf: CPF (somente números) para filtrar pessoa específica.
        offset: Deslocamento para paginação.
        limit: Quantidade por página.
    """
    url = f"{INABILITADOS_URL}/{cpf}" if cpf else INABILITADOS_URL
    params: dict[str, Any] = {"offset": offset, "limit": limit}
    data: dict[str, Any] = await http_get(url, params=params)
    items: list[dict[str, Any]] = data.get("items", [])
    return [Inabilitado(**item) for item in items]


async def consultar_inidoneos(
    cpf_cnpj: str | None = None,
    offset: int = 0,
    limit: int = 25,
) -> list[Inidoneo]:
    """Fetch licitantes declarados inidôneos pelo TCU.

    API: GET https://contas.tcu.gov.br/ords/condenacao/consulta/inidoneos[/{CPF_CNPJ}]

    Args:
        cpf_cnpj: CPF ou CNPJ (somente números) para filtrar.
        offset: Deslocamento para paginação.
        limit: Quantidade por página.
    """
    url = f"{INIDONEOS_URL}/{cpf_cnpj}" if cpf_cnpj else INIDONEOS_URL
    params: dict[str, Any] = {"offset": offset, "limit": limit}
    data: dict[str, Any] = await http_get(url, params=params)
    items: list[dict[str, Any]] = data.get("items", [])
    return [Inidoneo(**item) for item in items]


async def consultar_certidoes(cnpj: str) -> CertidaoConsolidada:
    """Fetch certidões consolidadas de pessoa jurídica (APF).

    Consulta 4 cadastros: TCU Inidôneos, CNJ CNIA, CGU CEIS, CGU CNEP.

    API: GET https://certidoes-apf.apps.tcu.gov.br/api/rest/publico/certidoes/{cnpj}

    Args:
        cnpj: CNPJ (somente números, sem formatação).
    """
    data: dict[str, Any] = await http_get(f"{CERTIDOES_URL}/{cnpj}")
    return CertidaoConsolidada(**data)


async def consultar_pedidos_congresso(
    numero_processo: str | None = None,
    page: int = 0,
) -> list[PedidoCongresso]:
    """Fetch solicitações do Congresso Nacional ao TCU.

    API: GET https://contas.tcu.gov.br/ords/api/publica/scn/pedidos_congresso

    Args:
        numero_processo: Número do processo TCU para filtrar.
        page: Página dos resultados.
    """
    if numero_processo:
        url = f"{PEDIDOS_CONGRESSO_URL}/{numero_processo}"
        params: dict[str, Any] = {}
    else:
        url = PEDIDOS_CONGRESSO_URL
        params = {"page": page}

    data: dict[str, Any] = await http_get(url, params=params)
    items: list[dict[str, Any]] = data.get("items", [])
    return [PedidoCongresso(**item) for item in items]


async def calcular_debito(
    data_atualizacao: str,
    parcelas: list[ParcelaDebito],
    aplica_juros: bool = True,
) -> ResultadoDebito:
    """Calcula débito atualizado com correção monetária (SELIC) e juros.

    API: POST https://divida.apps.tcu.gov.br/api/publico/calculadora/calcular-saldos-debito

    Args:
        data_atualizacao: Data de atualização (DD/MM/AAAA).
        parcelas: Lista de parcelas do débito.
        aplica_juros: Se deve aplicar juros de mora.
    """
    body: dict[str, Any] = {
        "dataAtualizacao": data_atualizacao,
        "aplicaJuros": aplica_juros,
        "parcelasDebito": [
            {
                "dataFato": p.data_fato,
                "indicativoDebitoCredito": p.indicativo,
                "valorOriginal": p.valor_original,
            }
            for p in parcelas
        ],
    }
    data: dict[str, Any] = await http_post(CALCULAR_DEBITO_URL, json_body=body)
    return ResultadoDebito(**data)


async def consultar_termos_contratuais() -> list[TermoContratual]:
    """Fetch termos contratuais firmados pelo TCU.

    API: GET https://contas.tcu.gov.br/contrata2RS/api/publico/termos-contratuais

    Nota: Retorna TODOS os registros (sem paginação). ~3800+ registros.
    """
    data: list[dict[str, Any]] = await http_get(TERMOS_CONTRATUAIS_URL, timeout=60.0)
    return [TermoContratual(**item) for item in data]


async def consultar_cadirreg(cpf: str) -> list[PessoaCadirreg]:
    """Fetch pessoa com contas irregulares no CADIRREG.

    API: GET https://dados-abertos.apps.tcu.gov.br/api/recuperapessoacadirreg/{cpf}

    Args:
        cpf: CPF (somente números).
    """
    data: list[dict[str, Any]] = await http_get(f"{CADIRREG_URL}/{cpf}")
    return [PessoaCadirreg(**item) for item in data]
