#!/usr/bin/env python3
"""Constrói o banco relacional DuckDB de homônimos perfeitos (todos os provedores)."""

from __future__ import annotations

import argparse
from pathlib import Path

import duckdb

from mcp_brasil.data.transparencia_municipios.homonimos_mvp import (
    DEFAULT_DB_PATH,
    DEFAULT_GO_BIN,
    collect_and_persist,
    init_schema,
    load_covered_municipios,
    recompute_homonimos,
)


def main() -> None:
    parser = argparse.ArgumentParser(description="Constrói DuckDB de homônimos (todos provedores)")
    parser.add_argument("--db-path", type=Path, default=DEFAULT_DB_PATH)
    parser.add_argument("--go-bin", type=Path, default=DEFAULT_GO_BIN)
    parser.add_argument(
        "--providers",
        type=str,
        default="all",
        help="Provedores: all, megasoft, tce_pi, tce_ce (ou separados por vírgula)",
    )
    parser.add_argument(
        "--ufs",
        type=str,
        default="",
        help="UFs extras para descoberta dinâmica (Quasar/Fiorilli), ex: SP,MG",
    )
    parser.add_argument("--limit", type=int, default=25000)
    parser.add_argument("--workers", type=int, default=16)
    parser.add_argument("--prefeituras-only", action="store_true")
    parser.add_argument("--skip-collect", action="store_true")
    args = parser.parse_args()

    args.db_path.parent.mkdir(parents=True, exist_ok=True)
    con = duckdb.connect(str(args.db_path))
    init_schema(con)

    if not args.skip_collect:
        extra_ufs = [u.strip() for u in args.ufs.split(",") if u.strip()]
        muns = load_covered_municipios(
            providers=args.providers,
            prefeituras_only=args.prefeituras_only,
            extra_ufs=extra_ufs,
        )
        print(
            f"Coletando {len(muns)} municípios (providers={args.providers}, "
            f"workers={args.workers})..."
        )
        c_stats = collect_and_persist(
            con,
            municipios=muns,
            go_bin=args.go_bin,
            limit=args.limit,
            workers=args.workers,
        )
        print(
            f"Coleta concluída: {c_stats['municipios_ok']}/{c_stats['municipios_total']} "
            f"portais OK | {c_stats['servidores']} servidores gravados."
        )

    print("\n=== COBERTURA POR PROVEDOR ===")
    for row in con.execute(
        """
        SELECT plataforma, coleta_status, COUNT(*), COALESCE(SUM(coletados), 0)
        FROM municipios
        GROUP BY 1, 2
        ORDER BY 1, 2
        """
    ).fetchall():
        print(f"  {row[0]} | {row[1]}: {row[2]} municípios ({row[3]} servidores)")

    print("\nCalculando grupos de homônimos e scores de similaridade...")
    h_stats = recompute_homonimos(con)
    print(f"Grupos: {h_stats['grupos']} | Pares candidatos: {h_stats['pares']}")

    print("\n=== DISTRIBUIÇÃO POR CLASSIFICAÇÃO ===")
    for row in con.execute(
        "SELECT classificacao, COUNT(*), ROUND(AVG(score_similaridade), 3) "
        "FROM pares_candidatos GROUP BY 1 ORDER BY 2 DESC"
    ).fetchall():
        print(f"  {row[0]}: {row[1]} pares (score médio {row[2]})")

    print("\n=== TOP 15 PARES (MAIOR SCORE) ===")
    for row in con.execute(
        """
        SELECT nome_normalizado, municipio_a || '/' || uf_a, cargo_a,
               municipio_b || '/' || uf_b, cargo_b, distancia_km,
               score_similaridade, classificacao, cpf_parcial_a, cpf_parcial_b
        FROM pares_candidatos
        ORDER BY score_similaridade DESC, distancia_km ASC
        LIMIT 15
        """
    ).fetchall():
        cpf_info = f" | CPF: {row[8]}/{row[9]}" if (row[8] or row[9]) else ""
        print(
            f"  [{row[6]:.2f} {row[7]}] {row[0]} | {row[1]} ({row[2]}) "
            f"<-> {row[3]} ({row[4]}) | {row[5]:.1f} km{cpf_info}"
        )

    print("\n=== TOP 10 GRUPOS EM 3+ MUNICÍPIOS (POR MAX_SCORE_PAR) ===")
    for row in con.execute(
        """
        SELECT nome_normalizado, qtd_municipios, qtd_ufs, qtd_registros,
               ROUND(score_raridade, 3), ROUND(max_score_par, 3)
        FROM homonimos_grupos
        WHERE qtd_municipios >= 3
        ORDER BY max_score_par DESC, qtd_municipios DESC
        LIMIT 10
        """
    ).fetchall():
        print(
            f"  {row[0]} | {row[1]} municípios ({row[2]} UFs, {row[3]} reg) "
            f"| raridade={row[4]} | max_score_par={row[5]}"
        )


if __name__ == "__main__":
    main()
