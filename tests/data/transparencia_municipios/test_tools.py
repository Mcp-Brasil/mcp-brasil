"""Tests for tools.py in transparencia_municipios."""

import pytest

from mcp_brasil.data.transparencia_municipios import tools


@pytest.mark.asyncio
async def test_buscar_municipios_formatting() -> None:
    res = await tools.buscar_municipios("Campinas", "SP")
    assert "Campinas" in res
    assert "3509502" in res
    assert "transparencia.campinas.sp.gov.br" in res


@pytest.mark.asyncio
async def test_listar_estados_formatting() -> None:
    res = await tools.listar_estados()
    assert "27 Unidades Federativas" in res
    assert "São Paulo" in res
    assert "Minas Gerais" in res
    assert "TCE" in res


@pytest.mark.asyncio
async def test_conectar_municipio_formatting() -> None:
    res = await tools.conectar_municipio("3550308", "SP")
    assert "Dossiê de Transparência: São Paulo (SP)" in res
    assert "3550308" in res
    assert "TCM-SP" in res
    assert "SICONFI" in res
    assert "PNCP" in res


@pytest.mark.asyncio
async def test_conectar_estado_formatting() -> None:
    res = await tools.conectar_estado("RJ")
    assert "Dossiê de Transparência Estadual: Rio de Janeiro (RJ)" in res
    assert "TCE-RJ" in res
    assert "ALERJ" in res
    assert "92" in res  # 92 municípios no RJ


@pytest.mark.asyncio
async def test_pesquisar_transparencia_municipio() -> None:
    res = await tools.pesquisar_transparencia("Curitiba", "PR")
    assert "Curitiba" in res
    assert "SICONFI" in res
    assert "PNCP" in res


@pytest.mark.asyncio
async def test_pesquisar_transparencia_estado() -> None:
    res = await tools.pesquisar_transparencia("BA")
    assert "Estado de Bahia" in res
    assert "transparencia.ba.gov.br" in res


@pytest.mark.asyncio
async def test_comparar_transparencia() -> None:
    res = await tools.comparar_transparencia(["SP", "RJ"])
    assert "Comparativo de Estrutura de Transparência" in res
    assert "São Paulo" in res
    assert "Rio de Janeiro" in res


@pytest.mark.asyncio
async def test_estatisticas_nacionais() -> None:
    res = await tools.estatisticas_nacionais()
    assert "5,571" in res or "5571" in res
    assert "27" in res
    assert "Sudeste" in res


@pytest.mark.asyncio
async def test_buscar_por_cnpj_formatting() -> None:
    res = await tools.buscar_por_cnpj("01613940000119")
    assert "Abadia de Goiás" in res
    assert "01613940000119" in res
    assert "TCM-GO" in res
    assert "PNCP" in res


@pytest.mark.asyncio
async def test_consultar_fiscal_formatting() -> None:
    res = await tools.consultar_fiscal("3550308", ano=2023, tipo="dca", limite=5)
    assert "SICONFI" in res
    assert "Balanço e Contas Anuais" in res
    assert "3550308" in res
    assert "R$" in res


@pytest.mark.asyncio
async def test_consultar_entregas_formatting() -> None:
    res = await tools.consultar_entregas("3550308", ano=2024)
    assert "Cumprimento da LRF" in res
    assert "3550308" in res
    assert "MSC" in res or "DCA" in res or "RREO" in res


@pytest.mark.asyncio
async def test_new_tools_formatting_mocked(monkeypatch: pytest.MonkeyPatch) -> None:
    from mcp_brasil.data.transparencia_municipios import client
    from mcp_brasil.data.transparencia_municipios.schemas import (
        ContratoItem,
        ContratosResult,
        DespesaItem,
        DespesasResult,
        EntregaItem,
        FiscalItem,
        LicitacaoItem,
        LicitacoesResult,
        Municipio,
        MunicipioDossier,
        PortalDiscoveryResult,
        ReceitaItem,
        ReceitasResult,
        ServidoresResult,
        ServidorItem,
        TransparenciaCompletaReport,
    )

    muni = Municipio(
        codigo_ibge="3166600",
        nome="Serra da Saudade",
        latitude=-19.43,
        longitude=-45.79,
        capital=False,
        codigo_uf=31,
        uf="MG",
        regiao="Sudeste",
        siafi_id="5331",
        ddd="37",
        fuso_horario="America/Sao_Paulo",
        slug="serra-da-saudade",
        cnpj="18500000000199",
        populacao=833,
    )
    disc = PortalDiscoveryResult(
        municipio=muni,
        url_resolvida="https://transparencia.serradasaudade.mg.gov.br",
        plataforma="quasar_publico",
        versao="v26.10.30",
        cod_administracao=2,
        entidade_nome="PREFEITURA MUNICIPAL",
        fonte_descoberta="dns_probe",
        servidores_suportado=True,
        candidatos_testados=["https://transparencia.serradasaudade.mg.gov.br"],
        tempo_ms=150,
    )
    serv = ServidoresResult(
        municipio=muni,
        fonte="Portal Municipal (Quasar)",
        plataforma="quasar_publico",
        portal_url="https://transparencia.serradasaudade.mg.gov.br",
        total_registros=409,
        disponivel=True,
        servidores=[
            ServidorItem(
                nome="AILTON JOSE SILVA",
                cpf_parcial="***.111.222-**",
                cargo="SERVICOS GERAIS",
                lotacao="PREFEITURA",
                competencia="Nov/2025",
                salario_bruto=2585.63,
                salario_liquido=2048.87,
            )
        ],
    )
    desp = DespesasResult(
        municipio=muni,
        fonte="Portal Municipal (Quasar)",
        plataforma="quasar_publico",
        total_registros=100,
        disponivel=True,
        despesas=[
            DespesaItem(
                numero_empenho="12/2025",
                data="2025-11-01",
                fornecedor="FORNECEDOR TESTE",
                historico="MATERIAL MEDICO",
                valor_empenhado=1500.0,
                valor_pago=1500.0,
            )
        ],
    )
    rec = ReceitasResult(
        municipio=muni,
        fonte="Portal Municipal (Quasar)",
        plataforma="quasar_publico",
        total_registros=50,
        disponivel=True,
        receitas=[
            ReceitaItem(
                codigo="1.7.1.8",
                descricao="COTA-PARTE FPM",
                categoria="ORDINARIOS",
                data="2025-11-01",
                valor_previsto=100000.0,
                valor_arrecadado=98000.0,
            )
        ],
    )
    lic = LicitacoesResult(
        municipio=muni,
        fonte="Portal Municipal (Quasar)",
        plataforma="quasar_publico",
        total_registros=10,
        disponivel=True,
        licitacoes=[
            LicitacaoItem(
                numero="05/2025",
                modalidade="PREGAO",
                situacao="HOMOLOGADA",
                data_abertura="2025-05-01",
                objeto="COMPRA DE VEICULO",
                valor_estimado=90000.0,
            )
        ],
    )
    cont = ContratosResult(
        municipio=muni,
        fonte="PNCP",
        plataforma="pncp",
        total_registros=5,
        disponivel=True,
        contratos=[
            ContratoItem(
                numero="02/2025",
                fornecedor="CONSTRUTORA ABC",
                data_assinatura="2025-06-01",
                situacao="VIGENTE",
                objeto="REFORMA ESCOLAR",
                valor=120000.0,
            )
        ],
    )
    dos = MunicipioDossier(
        municipio=muni,
        portal_transparencia_prefeitura="https://transparencia.serradasaudade.mg.gov.br",
        portal_prefeitura="https://www.serradasaudade.mg.gov.br",
        portal_camara="https://camaraserradasaudade.mg.leg.br",
        tribunal_contas_competente="TCE-MG",
        tribunal_contas_url="https://www.tce.mg.gov.br",
        radar_transparencia_url="https://radardatransparencia.atricon.org.br",
        siconfi_url="https://siconfi.tesouro.gov.br",
        siconfi_api="https://apidatalake.tesouro.gov.br",
        pncp_url="https://pncp.gov.br",
        transferegov_url="https://transferegov.sistema.gov.br",
        transparencia_federal_url="https://portaldatransparencia.gov.br",
    )
    report = TransparenciaCompletaReport(
        dossier=dos,
        discovery=disc,
        servidores=serv,
        despesas=desp,
        receitas=rec,
        licitacoes=lic,
        contratos=cont,
        siconfi_dca=[
            FiscalItem(
                exercicio=2024,
                instituicao="Pref",
                cod_ibge=3166600,
                conta="Ativo",
                valor=1000.0,
            )
        ],
        siconfi_rgf=[
            FiscalItem(
                exercicio=2024,
                instituicao="Pref",
                cod_ibge=3166600,
                conta="Despesa Pessoal",
                valor=500.0,
            )
        ],
        siconfi_entregas=[
            EntregaItem(
                exercicio=2024,
                cod_ibge=3166600,
                instituicao="Pref",
                entregavel="RGF",
                periodo=2,
                periodicidade="S",
                data_status="2025-01-30",
                forma_envio="M",
            )
        ],
        fontes_consultadas=["Portal Municipal (Quasar)", "PNCP", "SICONFI"],
        tempo_total_ms=800,
    )

    async def fake_disc(**_: object) -> PortalDiscoveryResult:
        return disc

    async def fake_serv(**_: object) -> ServidoresResult:
        return serv

    async def fake_desp(**_: object) -> DespesasResult:
        return desp

    async def fake_rec(**_: object) -> ReceitasResult:
        return rec

    async def fake_lic(**_: object) -> LicitacoesResult:
        return lic

    async def fake_cont(**_: object) -> ContratosResult:
        return cont

    async def fake_comp(**_: object) -> TransparenciaCompletaReport:
        return report

    monkeypatch.setattr(client, "descobrir_portal_municipio", fake_disc)
    monkeypatch.setattr(client, "consultar_servidores_municipio", fake_serv)
    monkeypatch.setattr(client, "consultar_despesas_municipio", fake_desp)
    monkeypatch.setattr(client, "consultar_receitas_municipio", fake_rec)
    monkeypatch.setattr(client, "consultar_licitacoes_municipio", fake_lic)
    monkeypatch.setattr(client, "consultar_contratos_municipio", fake_cont)
    monkeypatch.setattr(client, "consultar_transparencia_completa", fake_comp)

    out_disc = await tools.descobrir_portal_municipio("Serra da Saudade", "MG")
    assert "quasar_publico" in out_disc and "v26.10.30" in out_disc

    out_serv = await tools.consultar_servidores_municipio("Serra da Saudade", "MG")
    assert "AILTON JOSE SILVA" in out_serv and "2,585.63" in out_serv

    out_desp = await tools.consultar_despesas_municipio("Serra da Saudade", "MG")
    assert "FORNECEDOR TESTE" in out_desp and "1,500.00" in out_desp

    out_rec = await tools.consultar_receitas_municipio("Serra da Saudade", "MG")
    assert "COTA-PARTE FPM" in out_rec and "98,000.00" in out_rec

    out_lic = await tools.consultar_licitacoes_municipio("Serra da Saudade", "MG")
    assert "COMPRA DE VEICULO" in out_lic and "90,000.00" in out_lic

    out_cont = await tools.consultar_contratos_municipio("Serra da Saudade", "MG")
    assert "CONSTRUTORA ABC" in out_cont and "120,000.00" in out_cont

    out_comp = await tools.consultar_transparencia_completa("Serra da Saudade", "MG")
    assert "Auditoria Completa de Transparência" in out_comp
    assert "SICONFI — RGF" in out_comp
    assert "AILTON JOSE SILVA" in out_comp


@pytest.mark.asyncio
async def test_megasoft_tools_formatting(monkeypatch: pytest.MonkeyPatch) -> None:
    from mcp_brasil.data.transparencia_municipios.schemas import (
        ConcursoItem,
        ConcursosResult,
        ConvenioItem,
        ConveniosResult,
        DiariaItem,
        DiariasResult,
        LegislacaoItem,
        LegislacaoResult,
        MegasoftMunicipio,
        Municipio,
        ObraItem,
        ObrasResult,
        ServidorDetalheResult,
        VerbaItem,
    )

    muni = Municipio(
        codigo_ibge="5007950",
        nome="Tacuru",
        latitude=-23.636,
        longitude=-55.0141,
        capital=False,
        codigo_uf=50,
        uf="MS",
        regiao="Centro-Oeste",
        siafi_id="9815",
        ddd="67",
        fuso_horario="America/Porto_Velho",
        slug="tacuru",
        cnpj="03888989000100",
        populacao=11205,
    )

    async def fake_list_mega(**_: object) -> list[MegasoftMunicipio]:
        return [
            MegasoftMunicipio(
                codigo_ibge="5007950",
                nome="Tacuru",
                uf="MS",
                portal_url="https://tacuru.megasofttransparencia.com.br",
                portal_camara="https://camaratacuru.megasofttransparencia.com.br",
                codigo_orgao=10,
                site_prefeitura="https://tacuru.ms.gov.br",
            )
        ]

    async def fake_diarias(**_: object) -> DiariasResult:
        return DiariasResult(
            municipio=muni,
            fonte="Portal Municipal (Megasoft)",
            plataforma="megasoft",
            total_registros=1,
            disponivel=True,
            diarias=[
                DiariaItem(
                    beneficiario="JOAO MOTORISTA",
                    cargo="MOTORISTA",
                    destino="CAMPO GRANDE",
                    data="2025-05-10",
                    valor=350.0,
                    motivo="TRANSPORTE PACIENTE",
                    fonte="Portal Municipal (Megasoft)",
                )
            ],
        )

    async def fake_obras(**_: object) -> ObrasResult:
        return ObrasResult(
            municipio=muni,
            fonte="Portal Municipal (Megasoft)",
            plataforma="megasoft",
            total_registros=1,
            disponivel=True,
            obras=[
                ObraItem(
                    codigo="OB-01",
                    titulo="PAVIMENTACAO ASFALTICA",
                    situacao="EM ANDAMENTO",
                    data_inicial="2025-01-10",
                    valor=250000.0,
                    fonte="Portal Municipal (Megasoft)",
                )
            ],
        )

    async def fake_convenios(**_: object) -> ConveniosResult:
        return ConveniosResult(
            municipio=muni,
            fonte="Portal Municipal (Megasoft)",
            plataforma="megasoft",
            total_registros=1,
            disponivel=True,
            convenios=[
                ConvenioItem(
                    numero="CV-10",
                    concedente="FNDE",
                    convenente="PREFEITURA",
                    descricao="ONIBUS ESCOLAR",
                    valor=320000.0,
                    valor_liberado=320000.0,
                    fonte="Portal Municipal (Megasoft)",
                )
            ],
        )

    async def fake_leg(**_: object) -> LegislacaoResult:
        return LegislacaoResult(
            municipio=muni,
            fonte="Portal Municipal (Megasoft)",
            plataforma="megasoft",
            total_registros=1,
            disponivel=True,
            atos=[
                LegislacaoItem(
                    numero="1234",
                    ano=2025,
                    tipo_documento="LEI MUNICIPAL",
                    descricao="PLANO DIRETOR",
                    fonte="Portal Municipal (Megasoft)",
                )
            ],
        )

    async def fake_concursos(**_: object) -> ConcursosResult:
        return ConcursosResult(
            municipio=muni,
            fonte="Portal Municipal (Megasoft)",
            plataforma="megasoft",
            total_registros=1,
            disponivel=True,
            aprovados=[
                ConcursoItem(
                    nome="ANA CLARA",
                    cargo="MEDICA CLINICA",
                    classificacao=1,
                    situacao="CONVOCADO",
                    edital="01/2024",
                    fonte="Portal Municipal (Megasoft)",
                )
            ],
        )

    async def fake_detalhe(**_: object) -> ServidorDetalheResult:
        return ServidorDetalheResult(
            municipio=muni,
            nome="ANA CLARA",
            matricula="MAT-999",
            cpf_parcial="***.456.789-**",
            cargo="MEDICA CLINICA",
            orgao="SECRETARIA DE SAUDE",
            departamento="POSTO CENTRAL",
            vinculo="EFETIVO",
            situacao="ATIVO",
            competencia="10/2025",
            salario_bruto=12000.0,
            descontos=2500.0,
            vencimentos=[VerbaItem(descricao="SALARIO BASE", valor=12000.0)],
            verbas_desconto=[
                VerbaItem(descricao="PREVIDENCIA", valor=1500.0),
                VerbaItem(descricao="IRRF", valor=1000.0),
            ],
            fonte="Portal Municipal (Megasoft)",
        )

    monkeypatch.setattr(tools.client, "listar_municipios_megasoft", fake_list_mega)
    monkeypatch.setattr(tools.client, "consultar_diarias_municipio", fake_diarias)
    monkeypatch.setattr(tools.client, "consultar_obras_municipio", fake_obras)
    monkeypatch.setattr(tools.client, "consultar_convenios_municipio", fake_convenios)
    monkeypatch.setattr(tools.client, "consultar_legislacao_municipio", fake_leg)
    monkeypatch.setattr(tools.client, "consultar_concursos_municipio", fake_concursos)
    monkeypatch.setattr(tools.client, "consultar_detalhe_servidor_municipio", fake_detalhe)

    out_mega = await tools.listar_municipios_megasoft("MS")
    assert "Tacuru" in out_mega and "5007950" in out_mega

    out_diar = await tools.consultar_diarias_municipio("Tacuru", "MS")
    assert "JOAO MOTORISTA" in out_diar and "350.00" in out_diar

    out_obr = await tools.consultar_obras_municipio("Tacuru", "MS")
    assert "PAVIMENTACAO ASFALTICA" in out_obr and "250,000.00" in out_obr

    out_conv = await tools.consultar_convenios_municipio("Tacuru", "MS")
    assert "ONIBUS ESCOLAR" in out_conv and "320,000.00" in out_conv

    out_leg = await tools.consultar_legislacao_municipio("Tacuru", "MS")
    assert "PLANO DIRETOR" in out_leg and "1234" in out_leg

    out_conc = await tools.consultar_concursos_municipio("Tacuru", "MS")
    assert "ANA CLARA" in out_conc and "MEDICA CLINICA" in out_conc

    out_det = await tools.consultar_detalhe_servidor_municipio("Tacuru", "MAT-999", "MS")
    assert "Contracheque Detalhado" in out_det
    assert "12,000.00" in out_det
    assert "***.456.789-**" in out_det
    assert "PREVIDENCIA" in out_det
