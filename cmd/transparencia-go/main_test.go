package main_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	// Build binary for tests
	tmpDir, err := filepath.Abs("../../bin")
	if err == nil {
		binPath = filepath.Join(tmpDir, "transparencia-go")
	}
	m.Run()
}

func runCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Comando falhou: %v\nStderr: %s", err, stderr.String())
	}
	return stdout.Bytes()
}

func TestCLIListUFs(t *testing.T) {
	out := runCLI(t, "list-ufs", "--json")
	var ufs []map[string]any
	if err := json.Unmarshal(out, &ufs); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if len(ufs) != 27 {
		t.Errorf("Esperava 27 UFs, obteve %d", len(ufs))
	}
}

func TestCLISearch(t *testing.T) {
	out := runCLI(t, "search", "Salvador", "--json")
	var res map[string]any
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	munis, ok := res["municipios"].([]any)
	if !ok || len(munis) == 0 {
		t.Fatalf("Esperava encontrar Salvador")
	}
	first := munis[0].(map[string]any)
	if first["nome"] != "Salvador" || first["uf"] != "BA" {
		t.Errorf("Primeiro resultado inesperado: %v", first)
	}
}

func TestCLIConnectMunicipio(t *testing.T) {
	out := runCLI(t, "connect-municipio", "3550308", "--json")
	var dossier map[string]any
	if err := json.Unmarshal(out, &dossier); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if dossier["portal_transparencia_prefeitura"] == nil {
		t.Errorf("Campo portal_transparencia_prefeitura ausente")
	}
	if dossier["tribunal_contas_competente"] == nil {
		t.Errorf("Campo tribunal_contas_competente ausente")
	}
}

func TestCLIConnectEstado(t *testing.T) {
	out := runCLI(t, "connect-estado", "MG", "--json")
	var dossier map[string]any
	if err := json.Unmarshal(out, &dossier); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	est := dossier["estado"].(map[string]any)
	if est["uf"] != "MG" || est["nome"] != "Minas Gerais" {
		t.Errorf("Dossiê de estado inesperado: %v", est)
	}
}

func TestCLIStats(t *testing.T) {
	out := runCLI(t, "stats", "--json")
	var stats map[string]any
	if err := json.Unmarshal(out, &stats); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if int(stats["total_municipios"].(float64)) != 5571 {
		t.Errorf("Total de municípios incorreto: %v", stats["total_municipios"])
	}
	if int(stats["total_ufs"].(float64)) != 27 {
		t.Errorf("Total de UFs incorreto: %v", stats["total_ufs"])
	}
}

func TestCLICompare(t *testing.T) {
	out := runCLI(t, "compare", "SP", "RJ", "3550308", "--json")
	var rel map[string]any
	if err := json.Unmarshal(out, &rel); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	itens := rel["itens"].([]any)
	if len(itens) != 3 {
		t.Errorf("Esperava 3 itens no comparativo, obteve %d", len(itens))
	}
}

func TestCLICNPJ(t *testing.T) {
	out := runCLI(t, "cnpj", "01613940000119", "--json")
	var dos map[string]any
	if err := json.Unmarshal(out, &dos); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	m, ok := dos["municipio"].(map[string]any)
	if !ok || m["nome"] != "Abadia de Goiás" {
		t.Fatalf("Esperava município Abadia de Goiás, obteve %v", m)
	}
}

func TestCLIDiscoverAndServidoresMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/publico/versao":
			_, _ = w.Write([]byte(`"v26.10.30"`))
		case "/publico/buscarEntidades":
			_, _ = w.Write([]byte(`[{"codAdministracao":2,"desAdministracao":"PREFEITURA"}]`))
		case "/publico/servidor":
			_, _ = w.Write([]byte(`{"content":[{"nomServidor":"TESTE CLI","numcpf":"12345678901","nomCargo":"TECNICO","vlrSalarioBruto":3200,"vlrSalarioLiquido":2700}],"totalElements":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	discOut := runCLI(t, "discover", "Serra da Saudade", "--uf", "MG", "--url", srv.URL, "--json")
	var disc map[string]any
	if err := json.Unmarshal(discOut, &disc); err != nil {
		t.Fatalf("JSON inválido em discover: %v", err)
	}
	if disc["plataforma"] != "quasar_publico" {
		t.Errorf("Esperava plataforma quasar_publico, obteve %v", disc["plataforma"])
	}

	servOut := runCLI(t, "servidores", "Serra da Saudade", "--uf", "MG", "--url", srv.URL, "--limit", "5", "--json")
	var serv map[string]any
	if err := json.Unmarshal(servOut, &serv); err != nil {
		t.Fatalf("JSON inválido em servidores: %v", err)
	}
	lista, ok := serv["servidores"].([]any)
	if !ok || len(lista) != 1 {
		t.Fatalf("Esperava 1 servidor, obteve %v", serv["servidores"])
	}
	first := lista[0].(map[string]any)
	if first["nome"] != "TESTE CLI" || first["cpf_parcial"] != "***.456.789-**" {
		t.Errorf("Servidor CLI inesperado: %v", first)
	}
}

func TestCLIMegasoftList(t *testing.T) {
	out := runCLI(t, "megasoft-list", "--uf", "MS", "--json")
	var munis []map[string]any
	if err := json.Unmarshal(out, &munis); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if len(munis) != 2 {
		t.Errorf("Esperava 2 municípios de MS no Megasoft, obteve %d", len(munis))
	}
	foundTacuru := false
	for _, m := range munis {
		if m["nome"] == "Tacuru" && m["codigo_ibge"] == "5007950" {
			foundTacuru = true
		}
	}
	if !foundTacuru {
		t.Errorf("Tacuru não encontrado na listagem CLI Megasoft: %v", munis)
	}
}

func TestCLIMegasoftModulesMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/configuracao/carregamento-inicial":
			_, _ = w.Write([]byte(`{
				"tokenPortal": "mock-cli-token",
				"entidades": [{"id": 1, "nome": "Prefeitura", "tipoEntidade": "PREFEITURA"}],
				"anos": [2025]
			}`))
		case "/api/orgaos-e-servidores/diaria/paginado":
			_, _ = w.Write([]byte(`{"total": 1, "registros": [{"nomeDoFuncionario": "BENEFICIARIO CLI", "valorDaDiaria": 250.0}]}`))
		case "/api/contratos-convenios-e-licitacoes/obra/paginado":
			_, _ = w.Write([]byte(`{"total": 1, "registros": [{"titulo": "ASFALTO CLI", "valor": 50000.0}]}`))
		case "/api/contratos-convenios-e-licitacoes/convenio/paginado":
			_, _ = w.Write([]byte(`{"total": 1, "registros": [{"numeroDoConvenio": "CV-99", "descricao": "CONVENIO CLI", "valor": 30000.0}]}`))
		case "/api/legislacao-e-publicacoes/legislacao-municipal/paginado":
			_, _ = w.Write([]byte(`{"total": 1, "registros": [{"numeroDoDocumento": "999", "ano": 2025, "descricao": "LEI CLI"}]}`))
		case "/api/orgaos-e-servidores/lista-de-aprovados-em-concursos-e-processos-seletivos/paginado":
			_, _ = w.Write([]byte(`{"total": 1, "registros": [{"nomeDaPessoa": "CANDIDATO CLI", "nomeDoCargo": "MEDICO"}]}`))
		case "/api/orgaos-e-servidores/servidor/detalhe":
			_, _ = w.Write([]byte(`{
				"matricula": "M100",
				"nome": "SERVIDOR DETALHE CLI",
				"cpf": "12345678901",
				"cargo": "ENGENHEIRO",
				"somatorioMovimentoVencimentos": {
					"somatorioProventos": 8000.0,
					"somatorioDescontos": 1500.0,
					"totalLiquido": 6500.0
				},
				"movimentoVencimentos": {
					"total": 1,
					"registros": [{"nome": "BASE", "proventos": 8000.0, "descontos": 0.0}]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// 1. Diárias
	diarOut := runCLI(t, "diarias", "Tacuru", "--uf", "MS", "--url", srv.URL, "--json")
	var diar map[string]any
	if err := json.Unmarshal(diarOut, &diar); err != nil || int(diar["total_registros"].(float64)) != 1 {
		t.Fatalf("CLI diarias falhou: %v, out=%s", err, diarOut)
	}

	// 2. Obras
	obrOut := runCLI(t, "obras", "Tacuru", "--uf", "MS", "--url", srv.URL, "--json")
	var obr map[string]any
	if err := json.Unmarshal(obrOut, &obr); err != nil || int(obr["total_registros"].(float64)) != 1 {
		t.Fatalf("CLI obras falhou: %v, out=%s", err, obrOut)
	}

	// 3. Convênios
	convOut := runCLI(t, "convenios", "Tacuru", "--uf", "MS", "--url", srv.URL, "--json")
	var conv map[string]any
	if err := json.Unmarshal(convOut, &conv); err != nil || int(conv["total_registros"].(float64)) != 1 {
		t.Fatalf("CLI convenios falhou: %v, out=%s", err, convOut)
	}

	// 4. Legislação
	legOut := runCLI(t, "legislacao", "Tacuru", "--uf", "MS", "--url", srv.URL, "--json")
	var leg map[string]any
	if err := json.Unmarshal(legOut, &leg); err != nil || int(leg["total_registros"].(float64)) != 1 {
		t.Fatalf("CLI legislacao falhou: %v, out=%s", err, legOut)
	}

	// 5. Concursos
	concOut := runCLI(t, "concursos", "Tacuru", "--uf", "MS", "--url", srv.URL, "--json")
	var conc map[string]any
	if err := json.Unmarshal(concOut, &conc); err != nil || int(conc["total_registros"].(float64)) != 1 {
		t.Fatalf("CLI concursos falhou: %v, out=%s", err, concOut)
	}

	// 6. Servidor Detalhe
	detOut := runCLI(t, "servidor-detalhe", "Tacuru", "--matricula", "M100", "--uf", "MS", "--url", srv.URL, "--json")
	var det map[string]any
	if err := json.Unmarshal(detOut, &det); err != nil || det["nome"] != "SERVIDOR DETALHE CLI" {
		t.Fatalf("CLI servidor-detalhe falhou: %v, out=%s", err, detOut)
	}
	if det["cpf_parcial"] != "***.456.789-**" {
		t.Errorf("CPF parcial no detalhe CLI incorreto: %v", det["cpf_parcial"])
	}
}
