"""Testes unitários para o MVP relacional de homônimos em DuckDB."""

from __future__ import annotations

import duckdb

from mcp_brasil.data.transparencia_municipios.homonimos_mvp import (
    compute_name_rarity,
    compute_pair_similarity,
    init_schema,
    load_covered_municipios,
    normalize_name,
    recompute_homonimos,
)


def test_normalize_name_and_rarity() -> None:
    assert normalize_name("  José   da Silva-Neto ") == "JOSE DA SILVA NETO"
    rare = compute_name_rarity("WESNEY JOSE MENDANHA", total_occurrences=2)
    common = compute_name_rarity("ANTONIO PEREIRA DA SILVA", total_occurrences=15)
    assert rare > common


def test_compute_pair_similarity_signals() -> None:
    a = {
        "uf": "GO",
        "cargo": "FARMACEUTICO",
        "vinculo": "Concursado",
        "situacao": "Ativo",
        "salario_bruto": 4500.0,
        "competencia": "10/2026",
        "cpf_parcial": "",
    }
    b = {
        "uf": "TO",
        "cargo": "FARMACÊUTICO P. I-C",
        "vinculo": "Concursado",
        "situacao": "Ativo",
        "salario_bruto": 3200.0,
        "competencia": "10/2026",
        "cpf_parcial": "",
    }
    sim_close = compute_pair_similarity(a, b, rarity_score=0.40, distancia_km=36.7)
    assert sim_close.score >= 0.78
    assert sim_close.classificacao == "MUITO_ALTA"

    # CPF mismatch vetoes match
    a_cpf = {**a, "cpf_parcial": "***.111.222-**"}
    b_cpf = {**b, "cpf_parcial": "***.999.888-**"}
    sim_mismatch = compute_pair_similarity(a_cpf, b_cpf, rarity_score=0.40, distancia_km=36.7)
    assert sim_mismatch.score == 0.01
    assert sim_mismatch.classificacao == "BAIXA"

    # CPF match boosts to near-certainty
    b_same_cpf = {**b, "cpf_parcial": "***.111.222-**"}
    sim_match = compute_pair_similarity(a_cpf, b_same_cpf, rarity_score=0.40, distancia_km=36.7)
    assert sim_match.score >= 0.95


def test_recompute_homonimos_duckdb() -> None:
    con = duckdb.connect(":memory:")
    init_schema(con)
    con.executemany(
        "INSERT INTO municipios VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)",
        [
            (
                "5204904",
                "Campos Belos",
                "GO",
                -13.03,
                -46.77,
                18000,
                "https://a",
                "prefeitura",
                "megasoft",
                "OK",
                1,
                1,
            ),
            (
                "1705557",
                "Combinado",
                "TO",
                -12.79,
                -46.53,
                5000,
                "https://b",
                "prefeitura",
                "megasoft",
                "OK",
                1,
                1,
            ),
        ],
    )
    con.executemany(
        "INSERT INTO servidores VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
        [
            (
                1,
                "5204904",
                "Wesney José Mendanha",
                "WESNEY JOSE MENDANHA",
                "",
                "1",
                "FARMACEUTICO",
                "SAUDE",
                "Concursado",
                "Ativo",
                4581.43,
                500.0,
                4081.43,
                "10/2026",
                "40",
                "01/01/2020",
            ),
            (
                2,
                "1705557",
                "WESNEY JOSE MENDANHA",
                "WESNEY JOSE MENDANHA",
                "",
                "2",
                "FARMACÊUTICO P. I-C",
                "SAUDE",
                "Concursado",
                "Ativo",
                3224.17,
                300.0,
                2924.17,
                "10/2026",
                "40",
                "01/02/2021",
            ),
        ],
    )
    stats = recompute_homonimos(con)
    assert stats == {"grupos": 1, "pares": 1}
    row = con.execute("SELECT score_similaridade, classificacao FROM pares_candidatos").fetchone()
    assert row is not None
    assert row[0] >= 0.78
    assert row[1] == "MUITO_ALTA"


def test_load_covered_municipios() -> None:
    mega = load_covered_municipios(providers="megasoft")
    prefs = load_covered_municipios(providers="megasoft", prefeituras_only=True)
    all_prov = load_covered_municipios(providers="all")
    assert len(mega) == 268
    assert len(prefs) == 242
    assert len(all_prov) == 268 + 224 + 184
