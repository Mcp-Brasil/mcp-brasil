#!/usr/bin/env python
"""Canary de saúde das fontes de dados.

Sonda a `api_base` declarada em cada `FEATURE_META` e reporta as que
aparentam estar fora do ar ou bloqueando o servidor.

Por que existe
--------------
A suíte de testes é 100% mockada (`respx`): mede o parser contra a fixture,
nunca o contrato contra a fonte. Isso deixou passar, sem nenhum sinal,
features mortas em produção por meses — `tce_ce` ficou 100% fora do ar desde
2026-03-23 com o README anunciando "4 tools operacionais", e `tabua_mares`
só foi descoberta porque um usuário abriu issue.

O que ele PEGA
--------------
- DNS que não resolve
- conexão recusada / timeout / erro de TLS, persistindo entre tentativas
- HTTP 5xx persistente

E sinaliza como ATENÇÃO (não falha):
- HTTP 401/403 em feature que não declara `requires_auth` — possível WAF novo

O que ele NÃO pega  (leia antes de confiar)
-------------------------------------------
Mudança de caminho ou de schema em endpoint específico. A `api_base` de
várias features **não é um endpoint válido por si só**: `servicodados.ibge.gov.br/api`
devolve 503 enquanto `/api/v1/localidades/estados` devolve 200. Por isso
qualquer resposta HTTP conta como "host vivo" — o critério é deliberadamente
conservador, para que uma issue automática signifique alguma coisa.

Pegar quebra de endpoint exige smoke test por feature, chamando uma URL real
com parâmetros válidos. Features podem declarar essa URL em `CANARY_URLS`
abaixo; quando declarada, ela é sondada no lugar da `api_base` e aí sim um
404 conta como falha.

Uso
---
    uv run python scripts/canary_sources.py              # texto
    uv run python scripts/canary_sources.py --json
    uv run python scripts/canary_sources.py --markdown

Sai com 1 se houver FALHA; ATENÇÃO sozinha não quebra o build.
"""

from __future__ import annotations

import argparse
import asyncio
import json
import logging
import sys
from dataclasses import dataclass, field

import httpx

from mcp_brasil._shared.feature import FeatureRegistry

TIMEOUT = httpx.Timeout(25.0, connect=12.0)
CONCURRENCY = 6
TENTATIVAS = 3
BACKOFF = 3.0

# Navegador-like: vários WAFs gov.br têm denylist de assinaturas de ferramenta
# (curl, python-httpx, python-requests), o que produziria falso positivo.
HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 "
        "(KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"
    ),
    "Accept": "*/*",
}

# Endpoints reais, com parâmetros válidos, para features onde a api_base não
# serve como sonda. Quando presente, um 404/4xx AQUI conta como falha — é o
# único jeito de pegar mudança de caminho como a que matou tce_ce e tabua_mares.
# Adicione conforme for validando cada feature ao vivo.
CANARY_URLS: dict[str, str] = {
    "ibge": "https://servicodados.ibge.gov.br/api/v1/localidades/estados",
    "tabua_mares": "https://tabuamare.api.br/api/v2/states",
    "brasilapi": "https://brasilapi.com.br/api/banks/v1",
}


@dataclass
class Resultado:
    feature: str
    url: str
    status: int | None = None
    erro: str | None = None
    requires_auth: bool = False
    endpoint_real: bool = False
    tentativas_gastas: int = 0

    @property
    def instavel(self) -> bool:
        """Respondeu, mas só depois de falhar em pelo menos uma tentativa.

        Vários portais gov.br oscilam: `legis.senado.leg.br` respondeu 1 de 5
        vezes numa medição. Tratar isso como morte gera issue falsa; ignorar
        esconde degradação real. Fica numa categoria própria.
        """
        return self.erro is None and self.status is not None and self.tentativas_gastas > 1

    @property
    def nivel(self) -> str:
        """'ok' | 'atencao' | 'falha'."""
        if self.erro is not None or self.status is None:
            return "falha"
        if self.status >= 500:
            return "falha"
        # Num endpoint real declarado em CANARY_URLS, 4xx é quebra de contrato.
        if self.endpoint_real and self.status >= 400:
            return "falha"
        if self.status in (401, 403):
            # Esperado quando a feature declara exigir credencial.
            return "ok" if self.requires_auth else "atencao"
        if self.instavel:
            return "atencao"
        # Na api_base, qualquer outra resposta HTTP só prova que o host está vivo.
        return "ok"

    @property
    def motivo(self) -> str:
        if self.erro:
            return f"{self.erro} — {self.tentativas_gastas} tentativa(s), nenhuma respondeu"
        if self.status in (401, 403):
            return f"HTTP {self.status} — possível WAF novo ou auth passou a ser exigida"
        if self.status is not None and self.status >= 500:
            return f"HTTP {self.status} — erro no servidor da fonte"
        if self.instavel:
            return f"instável — HTTP {self.status} só na tentativa {self.tentativas_gastas}"
        return f"HTTP {self.status}"


@dataclass
class Relatorio:
    ok: list[Resultado] = field(default_factory=list)
    atencao: list[Resultado] = field(default_factory=list)
    falhas: list[Resultado] = field(default_factory=list)

    @property
    def total(self) -> int:
        return len(self.ok) + len(self.atencao) + len(self.falhas)


async def _probe(
    client: httpx.AsyncClient, feat: str, url: str, auth: bool, real: bool
) -> Resultado:
    r = Resultado(feature=feat, url=url, requires_auth=auth, endpoint_real=real)
    for tentativa in range(1, TENTATIVAS + 1):
        r.erro = None
        r.tentativas_gastas = tentativa
        try:
            resp = await client.get(url, headers=HEADERS, follow_redirects=True)
            r.status = resp.status_code
            # 5xx pode ser transitório — insiste antes de declarar falha.
            if resp.status_code < 500:
                return r
        except httpx.TimeoutException:
            r.erro = "timeout"
        except httpx.ConnectError as exc:
            r.erro = f"falha de conexão/DNS ({type(exc).__name__})"
        except httpx.HTTPError as exc:
            r.erro = f"{type(exc).__name__}: {exc}"
        if tentativa < TENTATIVAS:
            await asyncio.sleep(BACKOFF * tentativa)
    return r


def _coletar_fontes() -> list[tuple[str, str, bool, bool]]:
    reg = FeatureRegistry()
    reg.discover("mcp_brasil.data")
    fontes: list[tuple[str, str, bool, bool]] = []
    for nome, feat in sorted(reg.features.items()):
        meta = feat.meta
        override = CANARY_URLS.get(nome)
        base = override or getattr(meta, "api_base", None)
        if base:
            fontes.append(
                (nome, base, bool(getattr(meta, "requires_auth", False)), override is not None)
            )
    return fontes


async def rodar() -> Relatorio:
    fontes = _coletar_fontes()
    sem = asyncio.Semaphore(CONCURRENCY)

    async with httpx.AsyncClient(timeout=TIMEOUT) as client:

        async def limitado(n: str, u: str, a: bool, r: bool) -> Resultado:
            async with sem:
                return await _probe(client, n, u, a, r)

        resultados = await asyncio.gather(*[limitado(*f) for f in fontes])

    rel = Relatorio()
    for r in resultados:
        getattr(rel, {"ok": "ok", "atencao": "atencao", "falha": "falhas"}[r.nivel]).append(r)
    return rel


def _tabela(rs: list[Resultado]) -> list[str]:
    return ["| Feature | URL | Motivo |", "|---|---|---|"] + [
        f"| `{r.feature}` | {r.url} | {r.motivo} |" for r in sorted(rs, key=lambda x: x.feature)
    ]


def _markdown(rel: Relatorio) -> str:
    out = [
        f"**{len(rel.falhas)} falha(s)** e **{len(rel.atencao)} atenção(ões)** "
        f"em {rel.total} fontes.",
        "",
    ]
    if rel.falhas:
        out += ["## Falhas", "", *_tabela(rel.falhas), ""]
    if rel.atencao:
        out += ["## Atenção (não quebra o build)", "", *_tabela(rel.atencao), ""]
    if not rel.falhas and not rel.atencao:
        out.append("Nenhum problema detectado. ✅")
    out += [
        "",
        "> Este canary sonda a `api_base` de cada feature (ou a URL real declarada em "
        "`CANARY_URLS`). Ele **não** detecta mudança de schema, e na `api_base` qualquer "
        "resposta HTTP conta como host vivo — vários prefixos devolvem 404/503 por design.",
    ]
    return "\n".join(out)


def main() -> int:
    ap = argparse.ArgumentParser(description="Canary de saúde das fontes de dados.")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--markdown", action="store_true")
    args = ap.parse_args()

    logging.disable(logging.CRITICAL)
    rel = asyncio.run(rodar())

    if args.json:
        print(
            json.dumps(
                {
                    "total": rel.total,
                    "ok": len(rel.ok),
                    "atencao": [
                        {"feature": r.feature, "url": r.url, "motivo": r.motivo}
                        for r in rel.atencao
                    ],
                    "falhas": [
                        {"feature": r.feature, "url": r.url, "motivo": r.motivo}
                        for r in rel.falhas
                    ],
                },
                ensure_ascii=False,
                indent=2,
            )
        )
    elif args.markdown:
        print(_markdown(rel))
    else:
        for r in sorted(rel.falhas, key=lambda x: x.feature):
            print(f"FALHA    {r.feature:<22} {r.motivo:<58} {r.url}")
        for r in sorted(rel.atencao, key=lambda x: x.feature):
            print(f"ATENCAO  {r.feature:<22} {r.motivo:<58} {r.url}")
        print(
            f"\n{len(rel.ok)} ok / {len(rel.atencao)} atenção / "
            f"{len(rel.falhas)} falhas de {rel.total}"
        )

    return 1 if rel.falhas else 0


if __name__ == "__main__":
    sys.exit(main())
