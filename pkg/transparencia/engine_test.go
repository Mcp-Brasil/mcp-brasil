package transparencia_test

import (
	"strings"
	"testing"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestLoadAllMunicipiosAndEstados(t *testing.T) {
	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("Falha ao criar Engine: %v", err)
	}

	stats := engine.GetStats()
	if stats.TotalMunicipios != 5571 {
		t.Errorf("Esperava 5571 municípios e unidades territoriais, obteve %d", stats.TotalMunicipios)
	}

	if stats.TotalUFs != 27 {
		t.Errorf("Esperava 27 UFs, obteve %d", stats.TotalUFs)
	}

	if len(stats.Capitais) != 27 {
		t.Errorf("Esperava 27 capitais, obteve %d", len(stats.Capitais))
	}

	regioes := []string{"Norte", "Nordeste", "Sudeste", "Sul", "Centro-Oeste"}
	for _, r := range regioes {
		count, ok := stats.MunicipiosPorRegiao[r]
		if !ok || count == 0 {
			t.Errorf("Região %s sem municípios registrados", r)
		}
	}
}

func TestSearchMunicipios(t *testing.T) {
	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("Falha ao criar Engine: %v", err)
	}

	tests := []struct {
		name          string
		query         string
		uf            string
		limit         int
		expectMin     int
		expectTopName string
		expectTopUF   string
	}{
		{
			name:          "Busca exata São Paulo",
			query:         "São Paulo",
			limit:         5,
			expectMin:     1,
			expectTopName: "São Paulo",
			expectTopUF:   "SP",
		},
		{
			name:          "Busca sem acento sao paulo",
			query:         "sao paulo",
			limit:         5,
			expectMin:     1,
			expectTopName: "São Paulo",
			expectTopUF:   "SP",
		},
		{
			name:          "Busca Belem sem acento",
			query:         "belem",
			limit:         5,
			expectMin:     1,
			expectTopName: "Belém",
		},
		{
			name:          "Busca Florianopolis sem acento",
			query:         "florianopolis",
			limit:         5,
			expectMin:     1,
			expectTopName: "Florianópolis",
			expectTopUF:   "SC",
		},
		{
			name:          "Busca por código IBGE 3550308",
			query:         "3550308",
			limit:         5,
			expectMin:     1,
			expectTopName: "São Paulo",
			expectTopUF:   "SP",
		},
		{
			name:          "Busca por código SIAFI 7107",
			query:         "7107",
			limit:         5,
			expectMin:     1,
			expectTopName: "São Paulo",
			expectTopUF:   "SP",
		},
		{
			name:          "Filtro por UF MG",
			query:         "Belo",
			uf:            "MG",
			limit:         10,
			expectMin:     1,
			expectTopName: "Belo Horizonte",
			expectTopUF:   "MG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := engine.SearchMunicipios(tt.query, tt.uf, tt.limit)
			if len(results) < tt.expectMin {
				t.Fatalf("Esperava pelo menos %d resultados para '%s', obteve %d", tt.expectMin, tt.query, len(results))
			}
			if tt.expectTopName != "" && results[0].Nome != tt.expectTopName {
				t.Errorf("Esperava top resultado '%s', obteve '%s'", tt.expectTopName, results[0].Nome)
			}
			if tt.expectTopUF != "" && results[0].UF != tt.expectTopUF {
				t.Errorf("Esperava top resultado UF '%s', obteve '%s'", tt.expectTopUF, results[0].UF)
			}
		})
	}
}

func TestConnectMunicipioDossier(t *testing.T) {
	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("Falha ao criar Engine: %v", err)
	}

	// 1. São Paulo capital (deve ter TCM-SP)
	dosSP, err := engine.ConnectMunicipio("3550308", "SP")
	if err != nil {
		t.Fatalf("Erro ao conectar SP: %v", err)
	}
	if !strings.Contains(dosSP.TribunalContasCompetente, "TCM-SP") {
		t.Errorf("Esperava TCM-SP para a capital de SP, obteve %s", dosSP.TribunalContasCompetente)
	}
	if !strings.Contains(dosSP.PortalTransparenciaPrefeitura, "transparencia.sao-paulo.sp.gov.br") {
		t.Errorf("URL transparência incorreta: %s", dosSP.PortalTransparenciaPrefeitura)
	}
	if !strings.Contains(dosSP.SiconfiURL, "cod_ibge=3550308") {
		t.Errorf("URL Siconfi incorreta: %s", dosSP.SiconfiURL)
	}

	// 2. Rio de Janeiro capital (deve ter TCM-Rio)
	dosRJ, err := engine.ConnectMunicipio("3304557", "RJ")
	if err != nil {
		t.Fatalf("Erro ao conectar RJ: %v", err)
	}
	if !strings.Contains(dosRJ.TribunalContasCompetente, "TCM-Rio") {
		t.Errorf("Esperava TCM-Rio para o Rio, obteve %s", dosRJ.TribunalContasCompetente)
	}

	// 3. Salvador (município baiano fiscalizado pelo TCM-BA)
	dosSSA, err := engine.ConnectMunicipio("Salvador", "BA")
	if err != nil {
		t.Fatalf("Erro ao conectar Salvador: %v", err)
	}
	if !strings.Contains(dosSSA.TribunalContasCompetente, "TCM-BA") {
		t.Errorf("Esperava TCM-BA para Salvador, obteve %s", dosSSA.TribunalContasCompetente)
	}

	// 4. Campinas (município paulista do interior fiscalizado pelo TCE-SP)
	dosCampinas, err := engine.ConnectMunicipio("Campinas", "SP")
	if err != nil {
		t.Fatalf("Erro ao conectar Campinas: %v", err)
	}
	if !strings.Contains(dosCampinas.TribunalContasCompetente, "TCE-SP") {
		t.Errorf("Esperava TCE-SP para Campinas, obteve %s", dosCampinas.TribunalContasCompetente)
	}
}

func TestConnectEstadoDossierAllUFs(t *testing.T) {
	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("Falha ao criar Engine: %v", err)
	}

	ufs := []string{
		"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO",
		"MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI",
		"RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
	}

	for _, uf := range ufs {
		dos, err := engine.ConnectEstado(uf)
		if err != nil {
			t.Errorf("Falha ao conectar UF %s: %v", uf, err)
			continue
		}
		if dos.Estado.UF != uf {
			t.Errorf("UF divergente: esperava %s, obteve %s", uf, dos.Estado.UF)
		}
		if dos.PortalTransparencia == "" {
			t.Errorf("UF %s sem Portal da Transparência", uf)
		}
		if dos.TribunalContasNome == "" || dos.TribunalContasURL == "" {
			t.Errorf("UF %s sem Tribunal de Contas configurado", uf)
		}
		if dos.TotalMunicipios <= 0 && uf != "DF" {
			t.Errorf("UF %s com %d municípios (esperava > 0)", uf, dos.TotalMunicipios)
		}
	}
}

func TestCompareTransparencia(t *testing.T) {
	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("Falha ao criar Engine: %v", err)
	}

	alvos := []string{"SP", "RJ", "Belo Horizonte", "Curitiba"}
	rel, err := engine.CompareTransparencia(alvos)
	if err != nil {
		t.Fatalf("Erro ao comparar: %v", err)
	}

	if rel.TotalItens != 4 {
		t.Errorf("Esperava 4 itens no comparativo, obteve %d", rel.TotalItens)
	}

	// Verifica se os 2 primeiros são estados e os 2 últimos são municípios
	if rel.Itens[0].Tipo != "estado" || rel.Itens[1].Tipo != "estado" {
		t.Errorf("Esperava estados nos primeiros itens")
	}
	if rel.Itens[2].Tipo != "municipio" || rel.Itens[3].Tipo != "municipio" {
		t.Errorf("Esperava municípios nos últimos itens")
	}
}

func TestCNPJLookup(t *testing.T) {
	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("Falha ao criar Engine: %v", err)
	}

	// 1. Abadia de Goiás por CNPJ limpo
	m1, err := engine.GetMunicipioByCNPJ("01613940000119")
	if err != nil {
		t.Fatalf("Falha ao buscar município por CNPJ: %v", err)
	}
	if m1.Nome != "Abadia de Goiás" || m1.UF != "GO" {
		t.Errorf("Município incorreto retornado: %s (%s)", m1.Nome, m1.UF)
	}
	if m1.Populacao <= 0 {
		t.Errorf("População inválida: %d", m1.Populacao)
	}

	// 2. Busca por CNPJ com pontuação e formatação
	m2, err := engine.GetMunicipioByCNPJ("01.613.940/0001-19")
	if err != nil {
		t.Fatalf("Falha ao buscar município por CNPJ formatado: %v", err)
	}
	if m2.CodigoIBGE != m1.CodigoIBGE {
		t.Errorf("Divergência entre buscas por CNPJ formatado e não formatado")
	}

	// 3. Busca através do SearchMunicipios
	results := engine.SearchMunicipios("01613940000119", "", 5)
	if len(results) == 0 || results[0].CodigoIBGE != "5200050" {
		t.Errorf("SearchMunicipios falhou na busca direta por CNPJ")
	}

	// 4. Dossiê de transparência enriquecido com endpoints
	dos, err := engine.ConnectMunicipio("5200050", "GO")
	if err != nil {
		t.Fatalf("Erro ao gerar dossiê: %v", err)
	}
	if dos.CNPJ != "01613940000119" {
		t.Errorf("CNPJ incorreto no dossiê: %s", dos.CNPJ)
	}
	if dos.Populacao <= 0 {
		t.Errorf("População inválida no dossiê: %d", dos.Populacao)
	}
	if !strings.Contains(dos.PNCPCNPJURL, "01613940000119") {
		t.Errorf("PNCPCNPJURL incorreta: %s", dos.PNCPCNPJURL)
	}
	if !strings.Contains(dos.SiconfiDCAEndpoint, "5200050") {
		t.Errorf("SiconfiDCAEndpoint incorreto: %s", dos.SiconfiDCAEndpoint)
	}
}

func BenchmarkSearchSpeed(b *testing.B) {
	engine, err := transparencia.NewEngine()
	if err != nil {
		b.Fatalf("Falha ao criar Engine: %v", err)
	}

	queries := []string{"Sao Paulo", "Rio", "Curitiba", "Belo Horizonte", "Porto", "Salvador", "Recife", "Brasilia"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		_ = engine.SearchMunicipios(q, "", 10)
	}
}

