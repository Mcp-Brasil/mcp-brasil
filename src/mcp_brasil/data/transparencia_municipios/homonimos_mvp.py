"""MVP de banco relacional (DuckDB) para homônimos perfeitos no serviço público.

Coleta folhas de pagamento de todos os provedores cobertos pelo
``transparencia-go`` (Megasoft, TCE-PI, TCE-CE, Quasar Público e Fiorilli SCPI),
persiste em modelo relacional DuckDB e calcula um score explicável de
similaridade (proxy de probabilidade de mesma pessoa) entre ocorrências em
municípios distintos.

Uso responsável (LGPD / ACCEPTABLE_USE.md):
    Os resultados representam *indícios estatísticos para auditoria*, não prova
    de identidade ou irregularidade. CPF permanece mascarado (``cpf_parcial``).
"""

from __future__ import annotations

import concurrent.futures
import csv
import json
import math
import re
import subprocess
import tempfile
import unicodedata
from collections import defaultdict
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import duckdb

REPO_ROOT = Path(__file__).resolve().parents[4]
DATA_DIR = Path(__file__).resolve().parent / "data"
ESTADOS_CSV = DATA_DIR / "estados.csv"
MUNICIPIOS_CSV = DATA_DIR / "municipios.csv"
MEGASOFT_CSV = DATA_DIR / "megasoft_municipios.csv"
DEFAULT_DB_PATH = REPO_ROOT / "data" / "homonimos_mvp.duckdb"
DEFAULT_GO_BIN = REPO_ROOT / "bin" / "transparencia-go"

COMMON_TOKENS = frozenset(
    {
        "DA",
        "DE",
        "DO",
        "DAS",
        "DOS",
        "E",
        "SILVA",
        "SANTOS",
        "OLIVEIRA",
        "SOUZA",
        "SOUSA",
        "PEREIRA",
        "COSTA",
        "RODRIGUES",
        "ALMEIDA",
        "NASCIMENTO",
        "LIMA",
        "ARAUJO",
        "FERNANDES",
        "CARVALHO",
        "GOMES",
        "MARTINS",
        "ROCHA",
        "RIBEIRO",
        "ALVES",
        "FERREIRA",
        "MARIA",
        "JOSE",
        "ANTONIO",
        "JOAO",
        "FRANCISCO",
        "ANA",
        "PAULO",
        "PEDRO",
        "LUCAS",
        "CARLOS",
        "MARCOS",
        "LUIS",
        "LUIZ",
    }
)

PROFESSION_FAMILIES: dict[str, tuple[str, ...]] = {
    "MEDICINA": ("MEDIC", "CLINICO GERAL", "PEDIATRA", "GINECOLOG", "PSIQUIATR", "CIRURG"),
    "ENFERMAGEM": ("ENFERM", "TEC ENFERM", "TECNICO DE ENFERM", "AUX ENFERM"),
    "FARMACIA": ("FARMACEUT", "BIOQUIM"),
    "ODONTOLOGIA": ("ODONT", "DENTIST", "CIRURGIAO DENT"),
    "MAGISTERIO": ("PROFESSOR", "PROF ", "DOCENTE", "PEDAGOG", "MAGISTER"),
    "FISIOTERAPIA": ("FISIOTERAP",),
    "PSICOLOGIA": ("PSICOLOG",),
    "NUTRICAO": ("NUTRICION",),
    "ASSISTENCIA_SOCIAL": ("ASSISTENTE SOCIAL",),
    "DIREITO": ("ADVOGAD", "PROCURADOR", "ASSESSOR JURID"),
    "CONTABILIDADE": ("CONTADOR", "CONTABIL"),
    "MOTORISTA": ("MOTORISTA", "CONDUTOR"),
    "VIGILANCIA": ("VIGILANT", "GUARDA", "VIGIA"),
    "SERVICOS_GERAIS": (
        "AUXILIAR DE SERVICOS",
        "SERVICOS GERAIS",
        "GARI",
        "MERENDEIRA",
        "ZELADOR",
    ),
    "ADMINISTRATIVO": ("ASSISTENTE ADMIN", "AUXILIAR ADMIN", "AGENTE ADMIN", "ASSESSOR ADMIN"),
}


def strip_accents(text: str) -> str:
    """Remove acentos mantendo apenas caracteres ASCII base."""
    nfkd = unicodedata.normalize("NFKD", text)
    return "".join(ch for ch in nfkd if not unicodedata.combining(ch))


def normalize_name(name: str) -> str:
    """Normaliza nome completo para detecção de homônimos perfeitos."""
    cleaned = strip_accents((name or "").strip().upper())
    cleaned = re.sub(r"[^A-Z0-9\s]", " ", cleaned)
    return re.sub(r"\s+", " ", cleaned).strip()


def haversine_km(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
    """Calcula distância geodésica aproximada em km entre dois pontos."""
    r = 6371.0
    dlat = math.radians(lat2 - lat1)
    dlon = math.radians(lon2 - lon1)
    a = (
        math.sin(dlat / 2) ** 2
        + math.cos(math.radians(lat1)) * math.cos(math.radians(lat2)) * math.sin(dlon / 2) ** 2
    )
    return 2.0 * r * math.asin(math.sqrt(min(1.0, a)))


def detect_profession_family(cargo: str) -> str | None:
    """Mapeia cargo textual para uma família profissional ampla."""
    norm = normalize_name(cargo)
    if not norm:
        return None
    for family, prefixes in PROFESSION_FAMILIES.items():
        if any(p in norm for p in prefixes):
            return family
    return None


def compute_name_rarity(nome_norm: str, total_occurrences: int) -> float:
    """Calcula score de raridade do nome em [0.05, 0.50]."""
    tokens = [t for t in nome_norm.split() if len(t) > 1]
    if not tokens:
        return 0.05
    uncommon_ratio = sum(1 for t in tokens if t not in COMMON_TOKENS) / len(tokens)
    length_bonus = min(len(tokens), 5) / 5.0
    freq_penalty = 1.0 / (1.0 + math.log1p(max(0, total_occurrences - 2)))
    raw = 0.10 + (0.22 * uncommon_ratio) + (0.08 * length_bonus) + (0.10 * freq_penalty)
    return round(max(0.05, min(0.50, raw)), 4)


def _is_inactive_or_zero(situacao: str, vinculo: str, salario_bruto: float) -> bool:
    s = normalize_name(situacao)
    v = normalize_name(vinculo)
    if "APOSENTAD" in s or "PENSIONIST" in s or "APOSENTAD" in v or "PENSIONIST" in v:
        return True
    return salario_bruto <= 0


def classify_score(score: float) -> str:
    """Classifica score de similaridade em faixas legíveis."""
    if score >= 0.78:
        return "MUITO_ALTA"
    if score >= 0.60:
        return "ALTA"
    if score >= 0.42:
        return "MEDIA"
    return "BAIXA"


@dataclass(frozen=True)
class PairSimilarity:
    """Resultado explicável da comparação entre dois vínculos homônimos."""

    score: float
    classificacao: str
    sinais: dict[str, Any]


def compute_pair_similarity(
    a: dict[str, Any],
    b: dict[str, Any],
    *,
    rarity_score: float,
    distancia_km: float,
) -> PairSimilarity:
    """Calcula probabilidade/similaridade explicável em [0.01, 0.99] entre dois registros."""
    sinais: dict[str, Any] = {
        "raridade_nome": rarity_score,
        "mesma_uf": a.get("uf") == b.get("uf"),
        "distancia_km": round(distancia_km, 1),
    }

    cpf_a = (a.get("cpf_parcial") or "").strip()
    cpf_b = (b.get("cpf_parcial") or "").strip()
    if cpf_a and cpf_b:
        if cpf_a == cpf_b:
            sinais["cpf_parcial"] = "match"
            cpf_delta = 0.45
        else:
            sinais["cpf_parcial"] = "mismatch"
            return PairSimilarity(score=0.01, classificacao="BAIXA", sinais=sinais)
    else:
        sinais["cpf_parcial"] = "ausente"
        cpf_delta = 0.0

    # Proximidade geográfica
    if distancia_km <= 50:
        geo_delta = 0.18
        sinais["proximidade"] = "vizinhos_<=50km"
    elif distancia_km <= 150:
        geo_delta = 0.10
        sinais["proximidade"] = "microrregiao_<=150km"
    elif distancia_km <= 400:
        geo_delta = 0.03
        sinais["proximidade"] = "regional_<=400km"
    else:
        geo_delta = -0.08
        sinais["proximidade"] = "distante_>400km"

    # Compatibilidade de cargo
    fam_a = detect_profession_family(a.get("cargo", ""))
    fam_b = detect_profession_family(b.get("cargo", ""))
    norm_cargo_a = normalize_name(a.get("cargo", ""))
    norm_cargo_b = normalize_name(b.get("cargo", ""))

    if fam_a and fam_b and fam_a == fam_b:
        cargo_delta = 0.16
        sinais["cargo_compatibilidade"] = f"mesma_familia:{fam_a}"
    elif norm_cargo_a and norm_cargo_a == norm_cargo_b:
        cargo_delta = 0.16
        sinais["cargo_compatibilidade"] = "cargo_identico"
    else:
        toks_a = {t for t in norm_cargo_a.split() if len(t) > 3}
        toks_b = {t for t in norm_cargo_b.split() if len(t) > 3}
        if toks_a and toks_b and (toks_a & toks_b):
            cargo_delta = 0.06
            sinais["cargo_compatibilidade"] = "sobreposicao_parcial"
        else:
            cargo_delta = -0.04
            sinais["cargo_compatibilidade"] = "distintos"

    # Atividade / remuneração / competência
    inact_a = _is_inactive_or_zero(
        a.get("situacao", ""), a.get("vinculo", ""), float(a.get("salario_bruto") or 0)
    )
    inact_b = _is_inactive_or_zero(
        b.get("situacao", ""), b.get("vinculo", ""), float(b.get("salario_bruto") or 0)
    )
    comp_a = (a.get("competencia") or "").strip()
    comp_b = (b.get("competencia") or "").strip()

    if not inact_a and not inact_b:
        ativ_delta = 0.08 if (comp_a and comp_a == comp_b) else 0.04
        sinais["atividade"] = "ambos_ativos_remunerados"
    else:
        ativ_delta = -0.10
        sinais["atividade"] = "ao_menos_um_inativo_ou_zerado"

    score = round(
        max(0.01, min(0.99, rarity_score + cpf_delta + geo_delta + cargo_delta + ativ_delta)), 4
    )
    return PairSimilarity(score=score, classificacao=classify_score(score), sinais=sinais)


def _load_uf_map() -> dict[str, str]:
    uf_by_cod: dict[str, str] = {}
    with ESTADOS_CSV.open(encoding="utf-8-sig") as f:
        for row in csv.DictReader(f):
            uf_by_cod[row["codigo_uf"].strip()] = row["uf"].strip()
    return uf_by_cod


def load_covered_municipios(
    *,
    providers: str = "all",
    prefeituras_only: bool = False,
    extra_ufs: list[str] | None = None,
) -> list[dict[str, Any]]:
    """Carrega municípios cobertos pelos provedores (Megasoft, TCE-PI, TCE-CE e UFs extras)."""
    uf_by_cod = _load_uf_map()
    ibge_meta: dict[str, dict[str, Any]] = {}
    with MUNICIPIOS_CSV.open(encoding="utf-8-sig") as f:
        for row in csv.DictReader(f):
            row["uf"] = uf_by_cod.get(row.get("codigo_uf", "").strip(), "")
            ibge_meta[row["codigo_ibge"]] = row

    selected = {p.strip().lower() for p in providers.split(",") if p.strip()}
    if "all" in selected:
        selected = {"megasoft", "tce_pi", "tce_ce"}

    by_ibge: dict[str, dict[str, Any]] = {}

    if "megasoft" in selected:
        with MEGASOFT_CSV.open(encoding="utf-8-sig") as f:
            for row in csv.DictReader(f):
                ibge = row["codigo_ibge"]
                portal_url = (row.get("portal_url") or "").strip()
                is_camara = "camara" in portal_url.lower()
                if prefeituras_only and is_camara:
                    continue
                meta = ibge_meta.get(ibge, {})
                by_ibge[ibge] = {
                    "codigo_ibge": ibge,
                    "nome": row["nome"],
                    "uf": row["uf"],
                    "latitude": float(meta.get("latitude") or 0.0),
                    "longitude": float(meta.get("longitude") or 0.0),
                    "populacao": int(meta.get("populacao") or 0),
                    "portal_url": portal_url,
                    "tipo_portal": "camara" if is_camara else "prefeitura",
                    "plataforma": "megasoft",
                }

    target_ufs: dict[str, str] = {}
    if "tce_pi" in selected:
        target_ufs["PI"] = "tce_pi"
    if "tce_ce" in selected:
        target_ufs["CE"] = "tce_ce_sim"
    for uf in extra_ufs or []:
        u = uf.strip().upper()
        if u and u not in target_ufs:
            target_ufs[u] = "discover"

    if target_ufs:
        for ibge, meta in ibge_meta.items():
            uf = meta.get("uf", "")
            if uf in target_ufs and ibge not in by_ibge:
                by_ibge[ibge] = {
                    "codigo_ibge": ibge,
                    "nome": meta["nome"],
                    "uf": uf,
                    "latitude": float(meta.get("latitude") or 0.0),
                    "longitude": float(meta.get("longitude") or 0.0),
                    "populacao": int(meta.get("populacao") or 0),
                    "portal_url": "",
                    "tipo_portal": "prefeitura",
                    "plataforma": target_ufs[uf],
                }

    return list(by_ibge.values())


def _bulk_insert(con: duckdb.DuckDBPyConnection, table: str, rows: list[tuple[Any, ...]]) -> None:
    """Insere lotes grandes no DuckDB via CSV temporário (muito mais rápido que executemany)."""
    if not rows:
        return
    with tempfile.TemporaryDirectory() as tmpdir:
        csv_path = Path(tmpdir) / f"{table}.csv"
        with csv_path.open("w", newline="", encoding="utf-8") as f:
            writer = csv.writer(f)
            for row in rows:
                writer.writerow(["" if v is None else v for v in row])
        con.execute(
            f"COPY {table} FROM '{csv_path.as_posix()}' "
            "(FORMAT CSV, HEADER FALSE, DELIM ',', QUOTE '\"', ESCAPE '\"')"
        )


def init_schema(con: duckdb.DuckDBPyConnection) -> None:
    """Cria as tabelas relacionais do MVP no DuckDB."""
    con.execute(
        """
        CREATE TABLE IF NOT EXISTS municipios (
            codigo_ibge VARCHAR PRIMARY KEY,
            nome VARCHAR NOT NULL,
            uf VARCHAR(2) NOT NULL,
            latitude DOUBLE,
            longitude DOUBLE,
            populacao INTEGER,
            portal_url VARCHAR,
            tipo_portal VARCHAR,
            plataforma VARCHAR,
            coleta_status VARCHAR,
            total_registros INTEGER,
            coletados INTEGER,
            coletado_em TIMESTAMP
        );

        CREATE TABLE IF NOT EXISTS servidores (
            id BIGINT PRIMARY KEY,
            codigo_ibge VARCHAR NOT NULL,
            nome_raw VARCHAR NOT NULL,
            nome_normalizado VARCHAR NOT NULL,
            cpf_parcial VARCHAR,
            matricula VARCHAR,
            cargo VARCHAR,
            lotacao VARCHAR,
            vinculo VARCHAR,
            situacao VARCHAR,
            salario_bruto DOUBLE,
            descontos DOUBLE,
            salario_liquido DOUBLE,
            competencia VARCHAR,
            carga_horaria VARCHAR,
            data_admissao VARCHAR
        );

        CREATE TABLE IF NOT EXISTS homonimos_grupos (
            nome_normalizado VARCHAR PRIMARY KEY,
            qtd_municipios INTEGER NOT NULL,
            qtd_ufs INTEGER NOT NULL,
            qtd_registros INTEGER NOT NULL,
            score_raridade DOUBLE NOT NULL,
            max_score_par DOUBLE NOT NULL
        );

        CREATE TABLE IF NOT EXISTS pares_candidatos (
            id BIGINT PRIMARY KEY,
            nome_normalizado VARCHAR NOT NULL,
            ibge_a VARCHAR NOT NULL,
            municipio_a VARCHAR NOT NULL,
            uf_a VARCHAR(2) NOT NULL,
            cargo_a VARCHAR,
            vinculo_a VARCHAR,
            situacao_a VARCHAR,
            salario_bruto_a DOUBLE,
            competencia_a VARCHAR,
            cpf_parcial_a VARCHAR,
            carga_horaria_a VARCHAR,
            ibge_b VARCHAR NOT NULL,
            municipio_b VARCHAR NOT NULL,
            uf_b VARCHAR(2) NOT NULL,
            cargo_b VARCHAR,
            vinculo_b VARCHAR,
            situacao_b VARCHAR,
            salario_bruto_b DOUBLE,
            competencia_b VARCHAR,
            cpf_parcial_b VARCHAR,
            carga_horaria_b VARCHAR,
            distancia_km DOUBLE NOT NULL,
            score_similaridade DOUBLE NOT NULL,
            classificacao VARCHAR NOT NULL,
            sinais_json VARCHAR NOT NULL
        );
        """
    )


def _fetch_one_municipio(
    mun: dict[str, Any], *, go_bin: Path, limit: int, timeout_s: int
) -> tuple[dict[str, Any], dict[str, Any]]:
    cmd = [
        str(go_bin),
        "servidores",
        mun["codigo_ibge"],
        "--limit",
        str(limit),
        "--json",
    ]
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout_s)
        if res.returncode != 0 or not res.stdout.strip():
            return mun, {"disponivel": False, "erro": res.stderr.strip() or "exit_nonzero"}
        return mun, json.loads(res.stdout)
    except Exception as exc:
        return mun, {"disponivel": False, "erro": str(exc)}


def collect_and_persist(
    con: duckdb.DuckDBPyConnection,
    *,
    municipios: list[dict[str, Any]],
    go_bin: Path = DEFAULT_GO_BIN,
    limit: int = 25000,
    workers: int = 12,
    timeout_s: int = 60,
) -> dict[str, int]:
    """Coleta folhas dos municípios informados e grava em ``municipios`` e ``servidores``."""
    con.execute("DELETE FROM servidores;")
    con.execute("DELETE FROM municipios;")

    now = datetime.now(timezone.utc)
    mun_rows: list[tuple[Any, ...]] = []
    srv_rows: list[tuple[Any, ...]] = []
    srv_id = 1
    ok_count = 0

    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as ex:
        futures = [
            ex.submit(_fetch_one_municipio, m, go_bin=go_bin, limit=limit, timeout_s=timeout_s)
            for m in municipios
        ]
        for fut in concurrent.futures.as_completed(futures):
            mun, data = fut.result()
            servs = data.get("servidores") or []
            disp = bool(data.get("disponivel")) and len(servs) > 0
            status = "OK" if disp else ("VAZIO" if data.get("disponivel") else "FALHA")
            if disp:
                ok_count += 1
            plat = (data.get("plataforma") or mun.get("plataforma") or "").strip()
            purl = (data.get("portal_url") or mun.get("portal_url") or "").strip()
            mun_rows.append(
                (
                    mun["codigo_ibge"],
                    mun["nome"],
                    mun["uf"],
                    mun["latitude"],
                    mun["longitude"],
                    mun["populacao"],
                    purl,
                    mun["tipo_portal"],
                    plat,
                    status,
                    int(data.get("total_registros") or len(servs)),
                    len(servs),
                    now.isoformat(),
                )
            )
            for s in servs:
                raw_name = (s.get("nome") or "").strip()
                norm_name = normalize_name(raw_name)
                if not norm_name:
                    continue
                srv_rows.append(
                    (
                        srv_id,
                        mun["codigo_ibge"],
                        raw_name,
                        norm_name,
                        (s.get("cpf_parcial") or "").strip(),
                        (s.get("matricula") or "").strip(),
                        (s.get("cargo") or "").strip(),
                        (s.get("lotacao") or "").strip(),
                        (s.get("vinculo") or "").strip(),
                        (s.get("situacao") or "").strip(),
                        float(s.get("salario_bruto") or 0.0),
                        float(s.get("descontos") or 0.0),
                        float(s.get("salario_liquido") or 0.0),
                        (s.get("competencia") or "").strip(),
                        (s.get("carga_horaria") or "").strip(),
                        (s.get("data_admissao") or "").strip(),
                    )
                )
                srv_id += 1

    _bulk_insert(con, "municipios", mun_rows)
    _bulk_insert(con, "servidores", srv_rows)
    return {
        "municipios_total": len(mun_rows),
        "municipios_ok": ok_count,
        "servidores": len(srv_rows),
    }


def recompute_homonimos(con: duckdb.DuckDBPyConnection) -> dict[str, int]:
    """Recalcula ``homonimos_grupos`` e ``pares_candidatos`` a partir da tabela ``servidores``."""
    con.execute("DELETE FROM homonimos_grupos;")
    con.execute("DELETE FROM pares_candidatos;")

    muns_map: dict[str, dict[str, Any]] = {}
    for row in con.execute(
        "SELECT codigo_ibge, nome, uf, latitude, longitude FROM municipios"
    ).fetchall():
        muns_map[row[0]] = {
            "codigo_ibge": row[0],
            "nome": row[1],
            "uf": row[2],
            "latitude": row[3] or 0.0,
            "longitude": row[4] or 0.0,
        }

    # Agrupa somente nomes presentes em >= 2 municípios distintos
    by_name: dict[str, list[dict[str, Any]]] = defaultdict(list)
    rows = con.execute(
        """
        SELECT s.nome_normalizado, s.codigo_ibge, s.cargo, s.vinculo, s.situacao,
               s.salario_bruto, s.competencia, s.cpf_parcial, s.carga_horaria
        FROM servidores s
        WHERE s.nome_normalizado IN (
            SELECT nome_normalizado
            FROM servidores
            GROUP BY nome_normalizado
            HAVING COUNT(DISTINCT codigo_ibge) >= 2
        )
        """
    ).fetchall()

    for r in rows:
        mun = muns_map.get(r[1], {})
        by_name[r[0]].append(
            {
                "codigo_ibge": r[1],
                "municipio": mun.get("nome", r[1]),
                "uf": mun.get("uf", ""),
                "lat": mun.get("latitude", 0.0),
                "lon": mun.get("longitude", 0.0),
                "cargo": r[2] or "",
                "vinculo": r[3] or "",
                "situacao": r[4] or "",
                "salario_bruto": float(r[5] or 0.0),
                "competencia": r[6] or "",
                "cpf_parcial": r[7] or "",
                "carga_horaria": r[8] or "",
            }
        )

    grupos_rows: list[tuple[Any, ...]] = []
    pares_rows: list[tuple[Any, ...]] = []
    par_id = 1

    for nome_norm, entries in by_name.items():
        distinct_muns = {e["codigo_ibge"] for e in entries}
        distinct_ufs = {e["uf"] for e in entries if e["uf"]}
        rarity = compute_name_rarity(nome_norm, len(entries))

        # Deduplica múltiplos vínculos no mesmo município mantendo o de maior salário
        best_per_mun: dict[str, dict[str, Any]] = {}
        for e in entries:
            ibge = e["codigo_ibge"]
            if (
                ibge not in best_per_mun
                or e["salario_bruto"] > best_per_mun[ibge]["salario_bruto"]
            ):
                best_per_mun[ibge] = e

        mun_entries = sorted(best_per_mun.values(), key=lambda x: x["codigo_ibge"])
        max_pair_score = 0.0

        for i, a in enumerate(mun_entries):
            for b in mun_entries[i + 1 :]:
                dist = haversine_km(a["lat"], a["lon"], b["lat"], b["lon"])
                sim = compute_pair_similarity(a, b, rarity_score=rarity, distancia_km=dist)
                if sim.score > max_pair_score:
                    max_pair_score = sim.score
                pares_rows.append(
                    (
                        par_id,
                        nome_norm,
                        a["codigo_ibge"],
                        a["municipio"],
                        a["uf"],
                        a["cargo"],
                        a["vinculo"],
                        a["situacao"],
                        a["salario_bruto"],
                        a["competencia"],
                        a["cpf_parcial"],
                        a["carga_horaria"],
                        b["codigo_ibge"],
                        b["municipio"],
                        b["uf"],
                        b["cargo"],
                        b["vinculo"],
                        b["situacao"],
                        b["salario_bruto"],
                        b["competencia"],
                        b["cpf_parcial"],
                        b["carga_horaria"],
                        round(dist, 2),
                        sim.score,
                        sim.classificacao,
                        json.dumps(sim.sinais, ensure_ascii=False),
                    )
                )
                par_id += 1

        grupos_rows.append(
            (
                nome_norm,
                len(distinct_muns),
                len(distinct_ufs),
                len(entries),
                rarity,
                max_pair_score,
            )
        )

    _bulk_insert(con, "homonimos_grupos", grupos_rows)
    _bulk_insert(con, "pares_candidatos", pares_rows)
    return {"grupos": len(grupos_rows), "pares": len(pares_rows)}
