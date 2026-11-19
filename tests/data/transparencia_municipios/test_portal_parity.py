"""Catalog integrity and Go-vs-Python parity tests for transparencia_municipios."""

from __future__ import annotations

import csv
import os
from pathlib import Path
from urllib.parse import urlparse

import pytest

from mcp_brasil.data.transparencia_municipios import client
from mcp_brasil.data.transparencia_municipios.constants import (
    ESTADOS_CSV_PATH,
    GO_BINARY_PATH,
    MEGASOFT_CSV_PATH,
    MUNICIPIOS_CSV_PATH,
)

REPO_ROOT = Path(__file__).resolve().parents[3]
GO_DATA_DIR = REPO_ROOT / "pkg" / "transparencia" / "data"


def _assert_valid_http_url(raw: str, context: str) -> None:
    val = raw.strip()
    assert val, f"Empty URL in {context}"
    assert not any(c.isspace() for c in val), f"Whitespace in URL ({context}): {val!r}"
    low = val.lower()
    for bad in ("http://http://", "https://http://", "https://https://", "http://https://"):
        assert bad not in low, f"Duplicated scheme in URL ({context}): {val!r}"
    parsed = urlparse(val)
    assert parsed.scheme in ("http", "https"), f"Invalid scheme ({context}): {val!r}"
    assert parsed.netloc, f"Missing host ({context}): {val!r}"


def test_go_and_python_csv_catalogs_are_identical() -> None:
    """Go embedded CSVs and Python fallback CSVs must never drift apart."""
    pairs = [
        (GO_DATA_DIR / "estados.csv", ESTADOS_CSV_PATH),
        (GO_DATA_DIR / "municipios.csv", MUNICIPIOS_CSV_PATH),
        (GO_DATA_DIR / "megasoft_municipios.csv", MEGASOFT_CSV_PATH),
    ]
    for go_path, py_path in pairs:
        assert go_path.exists(), f"Missing Go CSV: {go_path}"
        assert py_path.exists(), f"Missing Python CSV: {py_path}"
        assert go_path.read_bytes() == py_path.read_bytes(), (
            f"CSV drift detected between {go_path} and {py_path}"
        )


def test_all_portal_catalog_csv_urls_are_well_formed() -> None:
    """Every URL across megasoft, API portal, and HTML portal catalogs must parse cleanly."""
    # 1. megasoft_municipios.csv
    with open(MEGASOFT_CSV_PATH, encoding="utf-8") as f:
        reader = csv.DictReader(f)
        mega_rows = list(reader)
    assert len(mega_rows) == 268
    for row in mega_rows:
        ctx = f"megasoft:{row['codigo_ibge']}"
        _assert_valid_http_url(row["portal_url"], f"{ctx}:portal_url")
        if row.get("portal_camara", "").strip():
            _assert_valid_http_url(row["portal_camara"], f"{ctx}:portal_camara")
        if row.get("site_prefeitura", "").strip():
            _assert_valid_http_url(row["site_prefeitura"], f"{ctx}:site_prefeitura")

    # 2. portais_municipais.csv
    api_csv = GO_DATA_DIR / "portais_municipais.csv"
    with open(api_csv, encoding="utf-8") as f:
        api_rows = list(csv.DictReader(f))
    assert len(api_rows) >= 2100
    api_ibges: set[str] = set()
    for row in api_rows:
        ibge = row["codigo_ibge"].strip()
        assert ibge not in api_ibges, f"Duplicate IBGE in portais_municipais.csv: {ibge}"
        api_ibges.add(ibge)
        _assert_valid_http_url(row["portal_url"], f"api:{ibge}")

    # 3. portais_html_municipais.csv
    html_csv = GO_DATA_DIR / "portais_html_municipais.csv"
    with open(html_csv, encoding="utf-8") as f:
        html_rows = list(csv.DictReader(f))
    assert len(html_rows) >= 2600
    html_ibges: set[str] = set()
    for row in html_rows:
        ibge = row["codigo_ibge"].strip()
        assert ibge not in html_ibges, f"Duplicate IBGE in portais_html_municipais.csv: {ibge}"
        assert ibge not in api_ibges, f"Overlap between API and HTML catalogs: {ibge}"
        html_ibges.add(ibge)
        _assert_valid_http_url(row["portal_url"], f"html:{ibge}")
        if row.get("site_prefeitura", "").strip():
            _assert_valid_http_url(row["site_prefeitura"], f"html:{ibge}:site_prefeitura")


@pytest.mark.asyncio
async def test_python_fallback_all_27_ufs_and_special_tcms(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    """Pure-Python fallback must return complete TCE/TCM and Assembleia metadata for all 27 UFs."""
    monkeypatch.setattr(client, "GO_BINARY_PATH", tmp_path / "missing-transparencia-go")

    estados = await client.listar_estados()
    assert len(estados) == 27
    assert sum(e.total_municipios for e in estados) == 5571

    for est in estados:
        dos = await client.conectar_estado(est.uf)
        assert dos.estado.uf == est.uf
        assert dos.total_municipios == est.total_municipios
        assert dos.assembleia_nome, f"Missing Assembleia name for {est.uf}"
        _assert_valid_http_url(dos.assembleia_url, f"assembleia:{est.uf}")
        _assert_valid_http_url(dos.portal_transparencia, f"transparencia:{est.uf}")
        _assert_valid_http_url(dos.tribunal_contas_url, f"tce:{est.uf}")

    # Verify municipal TCM overrides (SP capital, RJ capital, GO, BA, PA)
    sp_cap = await client.conectar_municipio("3550308", "SP")
    assert "TCM-SP" in sp_cap.tribunal_contas_competente
    campinas = await client.conectar_municipio("3509502", "SP")
    assert "TCE-SP" in campinas.tribunal_contas_competente

    rj_cap = await client.conectar_municipio("3304557", "RJ")
    assert "TCM-Rio" in rj_cap.tribunal_contas_competente
    niteroi = await client.conectar_municipio("3303302", "RJ")
    assert "TCE-RJ" in niteroi.tribunal_contas_competente

    go_muni = await client.conectar_municipio("5200050", "GO")
    assert "TCM-GO" in go_muni.tribunal_contas_competente
    assert "megasofttransparencia.com.br" in go_muni.portal_transparencia_prefeitura

    ba_muni = await client.conectar_municipio("2927408", "BA")
    assert "TCM-BA" in ba_muni.tribunal_contas_competente

    pa_muni = await client.conectar_municipio("1501402", "PA")
    assert "TCM-PA" in pa_muni.tribunal_contas_competente


@pytest.mark.asyncio
async def test_python_fallback_matches_go_binary_parity(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    """When bin/transparencia-go is built, Python fallback must match Go output."""
    if not GO_BINARY_PATH.exists() or not os.access(GO_BINARY_PATH, os.X_OK):
        pytest.skip("bin/transparencia-go not built; run `make build-go` to test Go/Python parity")

    # 1. Collect Go outputs
    go_estados = await client.listar_estados()
    go_stats = await client.obter_estatisticas()
    go_mega = await client.listar_municipios_megasoft()
    go_samples = [
        await client.conectar_municipio("3550308", "SP"),
        await client.conectar_municipio("3304557", "RJ"),
        await client.conectar_municipio("5200050", "GO"),
        await client.conectar_municipio("5007950", "MS"),
        await client.conectar_municipio("2927408", "BA"),
        await client.conectar_municipio("1501402", "PA"),
    ]
    go_uf_dossiers = {e.uf: await client.conectar_estado(e.uf) for e in go_estados}

    # 2. Switch to pure-Python fallback
    monkeypatch.setattr(client, "GO_BINARY_PATH", tmp_path / "missing-transparencia-go")

    py_estados = await client.listar_estados()
    py_stats = await client.obter_estatisticas()
    py_mega = await client.listar_municipios_megasoft()
    py_samples = [
        await client.conectar_municipio("3550308", "SP"),
        await client.conectar_municipio("3304557", "RJ"),
        await client.conectar_municipio("5200050", "GO"),
        await client.conectar_municipio("5007950", "MS"),
        await client.conectar_municipio("2927408", "BA"),
        await client.conectar_municipio("1501402", "PA"),
    ]
    py_uf_dossiers = {e.uf: await client.conectar_estado(e.uf) for e in py_estados}

    assert [e.uf for e in py_estados] == [e.uf for e in go_estados]
    assert py_stats.total_municipios == go_stats.total_municipios == 5571
    assert py_stats.total_ufs == go_stats.total_ufs == 27
    assert py_stats.populacao_total == go_stats.populacao_total
    assert py_stats.municipios_por_uf == go_stats.municipios_por_uf
    assert len(py_mega) == len(go_mega) == 268

    for uf in go_uf_dossiers:
        assert py_uf_dossiers[uf].portal_transparencia == go_uf_dossiers[uf].portal_transparencia
        assert py_uf_dossiers[uf].tribunal_contas_url == go_uf_dossiers[uf].tribunal_contas_url
        assert py_uf_dossiers[uf].assembleia_url == go_uf_dossiers[uf].assembleia_url

    for py_dos, go_dos in zip(py_samples, go_samples, strict=True):
        assert py_dos.municipio.codigo_ibge == go_dos.municipio.codigo_ibge
        assert py_dos.tribunal_contas_competente == go_dos.tribunal_contas_competente
        assert py_dos.tribunal_contas_url == go_dos.tribunal_contas_url
        assert py_dos.portal_transparencia_prefeitura == go_dos.portal_transparencia_prefeitura
