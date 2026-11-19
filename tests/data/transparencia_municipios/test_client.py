"""Tests for client.py in transparencia_municipios."""

import pytest

from mcp_brasil.data.transparencia_municipios import client


@pytest.mark.asyncio
async def test_listar_estados_returns_27() -> None:
    estados = await client.listar_estados()
    assert len(estados) == 27
    ufs = {e.uf for e in estados}
    assert "SP" in ufs
    assert "RJ" in ufs
    assert "DF" in ufs
    sp = next(e for e in estados if e.uf == "SP")
    assert sp.total_municipios == 645
    assert "transparencia.sp.gov.br" in sp.portal_transparencia


@pytest.mark.asyncio
async def test_buscar_municipios_sao_paulo() -> None:
    res = await client.buscar_municipios("São Paulo", limit=5)
    assert res.total_encontrado >= 1
    top = res.municipios[0]
    assert top.nome == "São Paulo"
    assert top.uf == "SP"
    assert top.codigo_ibge == "3550308"
    assert top.capital is True


@pytest.mark.asyncio
async def test_buscar_municipios_sem_acento() -> None:
    res = await client.buscar_municipios("florianopolis", limit=5)
    assert len(res.municipios) >= 1
    assert res.municipios[0].nome == "Florianópolis"
    assert res.municipios[0].uf == "SC"


@pytest.mark.asyncio
async def test_buscar_municipios_por_codigo_ibge() -> None:
    res = await client.buscar_municipios("4106902")
    assert len(res.municipios) == 1
    assert res.municipios[0].nome == "Curitiba"
    assert res.municipios[0].uf == "PR"


@pytest.mark.asyncio
async def test_conectar_municipio_dossier() -> None:
    dos = await client.conectar_municipio("3550308", "SP")
    assert dos.municipio.nome == "São Paulo"
    assert "transparencia.sao-paulo.sp.gov.br" in dos.portal_transparencia_prefeitura
    assert "TCM-SP" in dos.tribunal_contas_competente
    assert "tcm.sp.gov.br" in dos.tribunal_contas_url
    assert "siconfi" in dos.siconfi_url
    assert "pncp" in dos.pncp_url


@pytest.mark.asyncio
async def test_conectar_estado_dossier() -> None:
    dos = await client.conectar_estado("MG")
    assert dos.estado.nome == "Minas Gerais"
    assert dos.total_municipios == 853
    assert "transparencia.mg.gov.br" in dos.portal_transparencia
    assert "tce.mg.gov.br" in dos.tribunal_contas_url


@pytest.mark.asyncio
async def test_obter_estatisticas_nacionais() -> None:
    stats = await client.obter_estatisticas()
    assert stats.total_municipios == 5571
    assert stats.total_ufs == 27
    assert stats.populacao_total > 0
    assert len(stats.capitais) == 27
    assert "Sudeste" in stats.municipios_por_regiao
    assert stats.municipios_por_uf["SP"] == 645


@pytest.mark.asyncio
async def test_comparar_transparencia() -> None:
    rel = await client.comparar_transparencia(["SP", "Curitiba"])
    assert rel.total_itens == 2
    assert rel.itens[0].tipo == "estado"
    assert rel.itens[1].tipo == "municipio"


@pytest.mark.asyncio
async def test_buscar_por_cnpj() -> None:
    dos = await client.buscar_por_cnpj("01613940000119")
    assert dos.municipio.nome == "Abadia de Goiás"
    assert dos.municipio.uf == "GO"
    assert dos.municipio.populacao > 0
    assert dos.cnpj == "01613940000119"


@pytest.mark.asyncio
async def test_consultar_fiscal_dca(
    monkeypatch: pytest.MonkeyPatch, tmp_path: pytest.TempPathFactory
) -> None:
    from pathlib import Path

    import respx
    from httpx import Response

    monkeypatch.setattr(client, "GO_BINARY_PATH", Path(str(tmp_path)) / "missing-go")
    with respx.mock(assert_all_called=False) as router:
        router.get(url__startswith="https://apidatalake.tesouro.gov.br/ords/siconfi/tt/dca").mock(
            return_value=Response(
                200,
                json={
                    "items": [
                        {
                            "exercicio": 2023,
                            "instituicao": "Prefeitura Municipal de São Paulo - SP",
                            "cod_ibge": 3550308,
                            "populacao": 11451999,
                            "anexo": "DCA-Anexo I-AB",
                            "conta": "Ativo Total",
                            "cod_conta": "1.0.0.0.00.0.0",
                            "coluna": "Saldo Final",
                            "valor": 123456789.10,
                        }
                    ]
                },
            )
        )
        items = await client.consultar_fiscal("3550308", ano=2023, tipo="dca")
        assert len(items) > 0
        ativo = next((it for it in items if "Ativo" in it.conta), None)
        assert ativo is not None
        assert ativo.valor > 0


@pytest.mark.asyncio
async def test_consultar_entregas(
    monkeypatch: pytest.MonkeyPatch, tmp_path: pytest.TempPathFactory
) -> None:
    from pathlib import Path

    import respx
    from httpx import Response

    monkeypatch.setattr(client, "GO_BINARY_PATH", Path(str(tmp_path)) / "missing-go")
    with respx.mock(assert_all_called=False) as router:
        router.get(
            url__startswith="https://apidatalake.tesouro.gov.br/ords/siconfi/tt/extrato_entregas"
        ).mock(
            return_value=Response(
                200,
                json={
                    "items": [
                        {
                            "exercicio": 2024,
                            "cod_ibge": 3550308,
                            "instituicao": "Prefeitura Municipal de São Paulo - SP",
                            "populacao": 11451999,
                            "entregavel": "MSC Agregada",
                            "periodo": 12,
                            "periodicidade": "M",
                            "status_relatorio": "HO",
                            "data_status": "2025-01-30T10:00:00Z",
                            "forma_envio": "I",
                            "tipo_relatorio": "M",
                        }
                    ]
                },
            )
        )
        entregas = await client.consultar_entregas("3550308", ano=2024)
        assert len(entregas) > 0
        assert any("MSC" in it.entregavel for it in entregas)


@pytest.mark.asyncio
async def test_new_hub_wrappers_with_mocked_go(monkeypatch: pytest.MonkeyPatch) -> None:
    sample_muni = {
        "codigo_ibge": "3166600",
        "nome": "Serra da Saudade",
        "latitude": -19.43,
        "longitude": -45.79,
        "capital": False,
        "codigo_uf": 31,
        "uf": "MG",
        "regiao": "Sudeste",
        "siafi_id": "5331",
        "ddd": "37",
        "fuso_horario": "America/Sao_Paulo",
        "slug": "serra-da-saudade",
        "cnpj": "18500000000199",
        "populacao": 833,
    }

    async def fake_run(cmd: list[str]) -> dict[str, object]:
        sub = cmd[0]
        if sub == "discover":
            return {
                "municipio": sample_muni,
                "url_resolvida": "https://transparencia.serradasaudade.mg.gov.br",
                "plataforma": "quasar_publico",
                "versao": "v26.10.30",
                "cod_administracao": 2,
                "servidores_suportado": True,
                "fonte_descoberta": "dns_probe",
                "tempo_ms": 120,
            }
        if sub == "servidores":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Quasar)",
                "plataforma": "quasar_publico",
                "portal_url": "https://transparencia.serradasaudade.mg.gov.br",
                "total_registros": 1,
                "servidores": [
                    {
                        "nome": "AILTON JOSE SILVA",
                        "cpf_parcial": "***.111.222-**",
                        "cargo": "SERVICOS GERAIS",
                        "salario_bruto": 2585.63,
                        "salario_liquido": 2048.87,
                        "competencia": "Nov/2025",
                    }
                ],
                "disponivel": True,
            }
        if sub == "despesas":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Quasar)",
                "plataforma": "quasar_publico",
                "total_registros": 1,
                "despesas": [
                    {
                        "numero_empenho": "10/2025",
                        "fornecedor": "FORNECEDOR A",
                        "valor_empenhado": 1000.0,
                        "valor_pago": 1000.0,
                    }
                ],
                "disponivel": True,
            }
        if sub == "receitas":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Quasar)",
                "plataforma": "quasar_publico",
                "total_registros": 1,
                "receitas": [
                    {
                        "codigo": "1.7.1",
                        "descricao": "FPM",
                        "valor_previsto": 50000.0,
                        "valor_arrecadado": 48000.0,
                    }
                ],
                "disponivel": True,
            }
        if sub == "licitacoes":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Quasar)",
                "plataforma": "quasar_publico",
                "total_registros": 1,
                "licitacoes": [
                    {
                        "numero": "01/2025",
                        "modalidade": "Pregão",
                        "objeto": "Aquisição de materiais",
                        "valor_estimado": 15000.0,
                    }
                ],
                "disponivel": True,
            }
        if sub == "contratos":
            return {
                "municipio": sample_muni,
                "fonte": "PNCP",
                "plataforma": "pncp",
                "total_registros": 1,
                "contratos": [
                    {
                        "numero": "CT-01/2025",
                        "fornecedor": "EMPRESA X",
                        "objeto": "Serviços",
                        "valor": 14000.0,
                    }
                ],
                "disponivel": True,
            }
        raise AssertionError(f"Unexpected subcommand: {cmd}")

    monkeypatch.setattr(client, "_run_go_binary", fake_run)

    disc = await client.descobrir_portal_municipio("Serra da Saudade", "MG")
    assert disc.plataforma == "quasar_publico"
    assert disc.servidores_suportado is True

    serv = await client.consultar_servidores_municipio("Serra da Saudade", "MG")
    assert serv.disponivel is True
    assert serv.servidores[0].nome == "AILTON JOSE SILVA"

    desp = await client.consultar_despesas_municipio("Serra da Saudade", "MG")
    assert desp.despesas[0].valor_pago == 1000.0

    rec = await client.consultar_receitas_municipio("Serra da Saudade", "MG")
    assert rec.receitas[0].valor_arrecadado == 48000.0

    lic = await client.consultar_licitacoes_municipio("Serra da Saudade", "MG")
    assert lic.licitacoes[0].numero == "01/2025"

    cont = await client.consultar_contratos_municipio("Serra da Saudade", "MG")
    assert cont.contratos[0].valor == 14000.0


@pytest.mark.asyncio
async def test_listar_municipios_megasoft_catalog() -> None:
    all_munis = await client.listar_municipios_megasoft()
    assert len(all_munis) == 268
    go_munis = await client.listar_municipios_megasoft(uf="GO")
    assert len(go_munis) == 156
    ms_munis = await client.listar_municipios_megasoft(uf="MS")
    assert len(ms_munis) == 2
    tacuru = next((m for m in ms_munis if m.nome == "Tacuru"), None)
    assert tacuru is not None
    assert tacuru.codigo_ibge == "5007950"
    assert "tacuru" in tacuru.portal_url.lower()


@pytest.mark.asyncio
async def test_megasoft_modules_with_mocked_go(monkeypatch: pytest.MonkeyPatch) -> None:
    sample_muni = {
        "codigo_ibge": "5007950",
        "nome": "Tacuru",
        "latitude": -23.636,
        "longitude": -55.0141,
        "capital": False,
        "codigo_uf": 50,
        "uf": "MS",
        "regiao": "Centro-Oeste",
        "siafi_id": "9815",
        "ddd": "67",
        "fuso_horario": "America/Porto_Velho",
        "slug": "tacuru",
        "cnpj": "03888989000100",
        "populacao": 11205,
    }

    async def fake_run(cmd: list[str]) -> dict:
        sub = cmd[0]
        if sub == "diarias":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Megasoft)",
                "plataforma": "megasoft",
                "total_registros": 1,
                "diarias": [{"beneficiario": "JOAO MOTORISTA", "valor": 300.0}],
                "disponivel": True,
            }
        if sub == "obras":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Megasoft)",
                "plataforma": "megasoft",
                "total_registros": 1,
                "obras": [{"titulo": "ASFALTO", "valor": 80000.0, "situacao": "EM ANDAMENTO"}],
                "disponivel": True,
            }
        if sub == "convenios":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Megasoft)",
                "plataforma": "megasoft",
                "total_registros": 1,
                "convenios": [{"numero": "CV-01", "descricao": "ESCOLA", "valor": 50000.0}],
                "disponivel": True,
            }
        if sub == "legislacao":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Megasoft)",
                "plataforma": "megasoft",
                "total_registros": 1,
                "atos": [{"numero": "123", "ano": 2025, "descricao": "LEI MUNICIPAL"}],
                "disponivel": True,
            }
        if sub == "concursos":
            return {
                "municipio": sample_muni,
                "fonte": "Portal Municipal (Megasoft)",
                "plataforma": "megasoft",
                "total_registros": 1,
                "aprovados": [
                    {"nome": "MARIA CONCURSADA", "cargo": "MEDICA", "situacao": "APROVADO"}
                ],
                "disponivel": True,
            }
        if sub == "servidor-detalhe":
            return {
                "municipio": sample_muni,
                "nome": "MARIA DA SILVA",
                "matricula": "M100",
                "cpf_parcial": "***.456.789-**",
                "cargo": "PROFESSORA",
                "salario_bruto": 5000.0,
                "descontos": 800.0,
                "salario_liquido": 4200.0,
                "vencimentos": [{"descricao": "BASE", "valor": 5000.0}],
                "verbas_desconto": [{"descricao": "PREV", "valor": 800.0}],
                "fonte": "Portal Municipal (Megasoft)",
            }
        raise AssertionError(f"Unexpected subcommand: {cmd}")

    monkeypatch.setattr(client, "_run_go_binary", fake_run)

    diar = await client.consultar_diarias_municipio("Tacuru", "MS")
    assert diar.diarias[0].beneficiario == "JOAO MOTORISTA"
    assert diar.diarias[0].valor == 300.0

    obr = await client.consultar_obras_municipio("Tacuru", "MS")
    assert obr.obras[0].titulo == "ASFALTO"

    conv = await client.consultar_convenios_municipio("Tacuru", "MS")
    assert conv.convenios[0].valor == 50000.0

    leg = await client.consultar_legislacao_municipio("Tacuru", "MS")
    assert leg.atos[0].numero == "123"

    conc = await client.consultar_concursos_municipio("Tacuru", "MS")
    assert conc.aprovados[0].nome == "MARIA CONCURSADA"

    det = await client.consultar_detalhe_servidor_municipio("Tacuru", "M100", "MS")
    assert det.salario_bruto == 5000.0
    assert det.cpf_parcial == "***.456.789-**"
