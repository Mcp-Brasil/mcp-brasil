"""Demonstração mcp-brasil: testa tools reais via MCP Client com nomes namespaced."""

import asyncio
import json
import os
import time

os.environ["MCP_BRASIL_TOOL_SEARCH"] = "none"
results = []


async def test(client, feature, pergunta, tool_name, args=None):
    print(f"\n{'='*70}")
    print(f"[{feature}] {tool_name}")
    print(f"Pergunta: {pergunta}")
    print(f"{'='*70}")
    try:
        t0 = time.monotonic()
        result = await client.call_tool(tool_name, args or {})
        elapsed = time.monotonic() - t0
        content = result.content if hasattr(result, "content") else (result if isinstance(result, list) else [result])
        texts = [b.text for b in content if hasattr(b, "text")]
        result_str = "\n".join(texts) if texts else str(result)
        display = result_str[:1200] + "\n... [TRUNCADO]" if len(result_str) > 1200 else result_str
        print(f"✅ SUCESSO ({elapsed:.1f}s)\n{display}")
        results.append({
            "feature": feature, "pergunta": pergunta, "tool": tool_name,
            "status": "✅", "tempo": f"{elapsed:.1f}s",
            "resposta": result_str[:600],
        })
    except Exception as e:
        error_msg = f"{type(e).__name__}: {e}"
        print(f"❌ ERRO: {error_msg[:300]}")
        results.append({
            "feature": feature, "pergunta": pergunta, "tool": tool_name,
            "status": "❌", "tempo": "-",
            "resposta": error_msg[:300],
        })


async def run_all_tests():
    from fastmcp import Client
    from mcp_brasil.server import mcp

    async with Client(mcp) as client:
        tools = await client.list_tools()
        print(f"Total de tools: {len(tools)}")

        # IBGE
        await test(client, "IBGE", "Quais são os estados do Brasil?", "ibge_listar_estados")
        await test(client, "IBGE", "Quais municípios existem no Acre?", "ibge_buscar_municipios", {"uf": "AC"})
        await test(client, "IBGE", "Frequência do nome Maria?", "ibge_consultar_nome", {"nome": "Maria"})
        await test(client, "IBGE", "Nomes mais populares do Brasil?", "ibge_ranking_nomes")

        # BrasilAPI
        await test(client, "BrasilAPI", "Endereço do CEP 01001-000?", "brasilapi_consultar_cep", {"cep": "01001000"})
        await test(client, "BrasilAPI", "Dados da Petrobras?", "brasilapi_consultar_cnpj", {"cnpj": "33000167000101"})
        await test(client, "BrasilAPI", "Cidades do DDD 11?", "brasilapi_consultar_ddd", {"ddd": "11"})
        await test(client, "BrasilAPI", "Bancos do Brasil?", "brasilapi_listar_bancos")
        await test(client, "BrasilAPI", "Feriados de 2025?", "brasilapi_consultar_feriados", {"ano": 2025})
        await test(client, "BrasilAPI", "Domínio google.com.br?", "brasilapi_consultar_registro_br", {"dominio": "google.com.br"})

        # BACEN
        await test(client, "BACEN", "Taxa SELIC recente?", "bacen_consultar_serie", {"codigo": 432, "data_inicial": "01/01/2025", "data_final": "28/03/2025"})
        await test(client, "BACEN", "Séries populares?", "bacen_series_populares")
        await test(client, "BACEN", "Indicadores econômicos atuais?", "bacen_indicadores_atuais")

        # Câmara
        await test(client, "Câmara", "Deputados federais?", "camara_listar_deputados")

        # Senado
        await test(client, "Senado", "Senadores em exercício?", "senado_listar_senadores")
        await test(client, "Senado", "Partidos no Senado?", "senado_partidos_senado")

        # TSE
        await test(client, "TSE", "Anos com eleições?", "tse_anos_eleitorais")
        await test(client, "TSE", "Eleições de 2022?", "tse_listar_eleicoes")

        # Transparência
        await test(client, "Transparência", "Servidores 'Silva'?", "transparencia_buscar_servidores", {"nome": "Silva"})

        # TCU
        await test(client, "TCU", "Empresas inidôneas?", "tcu_consultar_inidoneos")

        # Dados Abertos
        await test(client, "Dados Abertos", "Datasets sobre educação?", "dados_abertos_buscar_conjuntos", {"texto": "educação"})

        # Tábua de Marés
        await test(client, "Tábua de Marés", "Estados com litoral?", "tabua_mares_listar_estados_costeiros")
        await test(client, "Tábua de Marés", "Portos do RJ?", "tabua_mares_listar_portos", {"estado": "RJ"})

        # INPE
        await test(client, "INPE", "Satélites de queimadas?", "inpe_dados_satelite")
        await test(client, "INPE", "Focos no Amazonas?", "inpe_buscar_focos_queimadas", {"estado": "AM"})

        # Farmácia Popular
        await test(client, "Farmácia Popular", "Medicamentos gratuitos?", "farmacia_popular_listar_medicamentos")

        # ANVISA
        await test(client, "ANVISA", "Paracetamol na ANVISA?", "anvisa_buscar_medicamento", {"nome": "paracetamol"})

        # ANA
        await test(client, "ANA", "Reservatórios de água?", "ana_monitorar_reservatorios")

        # Diário Oficial
        await test(client, "Diário Oficial", "Municípios SP com diários?", "diario_oficial_buscar_cidades", {"nome": "São Paulo"})

        # RENAME
        await test(client, "RENAME", "Grupos terapêuticos?", "rename_listar_grupos_terapeuticos")

        # BNDES
        await test(client, "BNDES", "Datasets financiamento?", "bndes_buscar_datasets_bndes", {"query": "financiamento"})

        # BPS
        await test(client, "BPS", "Preço amoxicilina governo?", "bps_buscar_medicamento_bps", {"descricao": "amoxicilina"})

        # OpenDataSUS
        await test(client, "OpenDataSUS", "Datasets saúde?", "opendatasus_listar_datasets_conhecidos")

        # TransfereGov
        await test(client, "TransfereGov", "Emendas pix 2024?", "transferegov_buscar_emendas_pix", {"ano": 2024})

        # TCE-RS
        await test(client, "TCE-RS", "Municípios RS?", "tce_rs_listar_municipios_rs")

        # Imunização
        await test(client, "Imunização", "Calendário vacinação?", "imunizacao_calendario_vacinacao")

        # Jurisprudência
        await test(client, "Jurisprudência", "STF liberdade expressão?", "jurisprudencia_buscar_jurisprudencia_stf", {"query": "liberdade de expressão"})

        # SINESP
        await test(client, "SINESP", "Datasets MJSP?", "sinesp_listar_datasets_mjsp")

        # Fórum Segurança
        await test(client, "Fórum Segurança", "Temas segurança?", "forum_seguranca_listar_temas_seguranca")

        # DENASUS
        await test(client, "DENASUS", "O que é o SNA?", "denasus_informacoes_sna")

        # Saúde/CNES
        await test(client, "Saúde/CNES", "Tipos estabelecimento saúde?", "saude_listar_tipos_estabelecimento")

        # Atlas Violência
        await test(client, "Atlas Violência", "Temas violência?", "atlas_violencia_listar_temas_violencia")

        # TCE-SP
        await test(client, "TCE-SP", "Municípios SP no TCE?", "tce_sp_listar_municipios_sp")

        # TCE-RJ
        await test(client, "TCE-RJ", "Obras paralisadas RJ?", "tce_rj_buscar_obras_paralisadas")

        # TCE-CE
        await test(client, "TCE-CE", "Municípios CE no TCE?", "tce_ce_listar_municipios_ce")

        # TCE-PE
        await test(client, "TCE-PE", "Unidades gestoras PE?", "tce_pe_buscar_unidades_pe")

        # DataJud
        await test(client, "DataJud", "Processos consumidor TJSP?", "datajud_buscar_processos", {"query": "direito do consumidor", "tribunal": "TJSP"})

        # Redator
        await test(client, "Redator", "Pronome para Ministro?", "redator_consultar_pronome_tratamento", {"cargo": "ministro"})
        await test(client, "Redator", "Tipos documento oficial?", "redator_listar_tipos_documento")
        await test(client, "Redator", "Data por extenso?", "redator_formatar_data_extenso")

        # Compras/PNCP
        await test(client, "Compras", "Fornecedor por CNPJ?", "compras_dadosabertos_consultar_fornecedor", {"cnpj": "33000167000101"})

    # RESUMO
    print("\n\n" + "=" * 70)
    print("RESUMO DOS TESTES")
    print("=" * 70)
    sucesso = [r for r in results if r["status"] == "✅"]
    erro = [r for r in results if r["status"] == "❌"]
    total = len(results)
    print(f"\nTotal: {total} | ✅ Sucesso: {len(sucesso)} | ❌ Erro: {len(erro)}")
    print(f"Taxa de sucesso: {len(sucesso)/total*100:.1f}%\n")
    if erro:
        print("Testes com erro:")
        for r in erro:
            print(f"  - [{r['feature']}] {r['tool']}: {r['resposta'][:100]}")
    with open("showcase_results.json", "w") as f:
        json.dump(results, f, ensure_ascii=False, indent=2)
    print(f"\nResultados salvos em showcase_results.json")
    return results

if __name__ == "__main__":
    asyncio.run(run_all_tests())
