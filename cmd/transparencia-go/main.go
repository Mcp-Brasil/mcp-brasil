package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	engine, err := transparencia.NewEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao inicializar motor de transparência: %v\n", err)
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "list-ufs":
		cmdListUFs(engine, os.Args[2:])
	case "search":
		cmdSearch(engine, os.Args[2:])
	case "get":
		cmdGet(engine, os.Args[2:])
	case "cnpj":
		cmdCNPJ(engine, os.Args[2:])
	case "connect-municipio":
		cmdConnectMunicipio(engine, os.Args[2:])
	case "connect-estado":
		cmdConnectEstado(engine, os.Args[2:])
	case "discover":
		cmdDiscover(engine, os.Args[2:])
	case "servidores":
		cmdServidores(engine, os.Args[2:])
	case "despesas":
		cmdDespesas(engine, os.Args[2:])
	case "receitas":
		cmdReceitas(engine, os.Args[2:])
	case "licitacoes":
		cmdLicitacoes(engine, os.Args[2:])
	case "contratos":
		cmdContratos(engine, os.Args[2:])
	case "diarias":
		cmdDiarias(engine, os.Args[2:])
	case "obras":
		cmdObras(engine, os.Args[2:])
	case "convenios":
		cmdConvenios(engine, os.Args[2:])
	case "legislacao":
		cmdLegislacao(engine, os.Args[2:])
	case "concursos":
		cmdConcursos(engine, os.Args[2:])
	case "servidor-detalhe":
		cmdServidorDetalhe(engine, os.Args[2:])
	case "megasoft-list":
		cmdMegasoftList(engine, os.Args[2:])
	case "transparencia":
		cmdTransparenciaCompleta(engine, os.Args[2:])
	case "fiscal":
		cmdFiscal(engine, os.Args[2:])
	case "entregas":
		cmdEntregas(engine, os.Args[2:])
	case "stats":
		cmdStats(engine, os.Args[2:])
	case "compare":
		cmdCompare(engine, os.Args[2:])
	case "probe":
		cmdProbe(engine, os.Args[2:])
	case "serve":
		cmdServe(engine, os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Comando desconhecido: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`transparencia-go — Motor nativo de alta velocidade para Transparência dos Municípios e Estados Brasileiros

Uso:
  transparencia-go <comando> [argumentos]

Comandos disponíveis:
  list-ufs [--json]
      Lista todas as 27 Unidades Federativas do Brasil com links oficiais.
  search <termo> [--uf <UF>] [--limit <N>] [--json]
      Busca rápida entre todos os 5.570 municípios (por nome, CNPJ, código IBGE ou SIAFI).
  get <codigo_ibge_ou_siafi> [--json]
      Obtém detalhes de um município pelo código IBGE ou SIAFI.
  cnpj <cnpj> [--json]
      Obtém detalhes e transparência de um município pelo CNPJ oficial da Prefeitura.
  connect-municipio <nome_ou_ibge> [--uf <UF>] [--json]
      Conecta a um município e retorna seu dossiê completo de transparência.
  connect-estado <uf> [--json]
      Conecta a uma UF e retorna seu dossiê completo de transparência e órgãos.
  discover <nome_ou_ibge> [--uf <UF>] [--url <url>] [--json]
      Descobre dinamicamente o portal da transparência real e a plataforma do município.
  megasoft-list [--uf <UF>] [--json]
      Lista todos os municípios brasileiros com portais Megasoft verificados.
  servidores <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--inicio YYYY-MM-DD] [--fim YYYY-MM-DD] [--ano YYYY] [--mes M] [--limit N] [--url <url>] [--json]
      Consulta servidores públicos e remuneração via portal municipal ou TCE estadual.
  servidor-detalhe <nome_ou_ibge> --matricula <mat> [--uf <UF>] [--ano YYYY] [--mes M] [--url <url>] [--json]
      Consulta o contracheque detalhado (verbas e descontos) de um servidor por matrícula.
  diarias <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--ano YYYY] [--limit N] [--url <url>] [--json]
      Consulta diárias e passagens pagas a servidores e agentes públicos.
  despesas <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--inicio YYYY-MM-DD] [--fim YYYY-MM-DD] [--ano YYYY] [--mes M] [--limit N] [--url <url>] [--json]
      Consulta despesas e empenhos municipais via portal municipal ou TCE estadual.
  receitas <nome_ou_ibge> [--uf <UF>] [--inicio YYYY-MM-DD] [--fim YYYY-MM-DD] [--ano YYYY] [--mes M] [--limit N] [--url <url>] [--json]
      Consulta receitas arrecadadas via portal municipal ou TCE estadual.
  licitacoes <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--inicio YYYY-MM-DD] [--fim YYYY-MM-DD] [--ano YYYY] [--limit N] [--url <url>] [--json]
      Consulta licitações e contratações via portal municipal ou PNCP.
  contratos <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--inicio YYYY-MM-DD] [--fim YYYY-MM-DD] [--ano YYYY] [--limit N] [--url <url>] [--json]
      Consulta contratos administrativos via portal municipal ou PNCP.
  obras <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--limit N] [--url <url>] [--json]
      Consulta obras públicas municipais (portais Megasoft).
  convenios <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--limit N] [--url <url>] [--json]
      Consulta convênios e transferências firmados pelo município (portais Megasoft).
  legislacao <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--ano YYYY] [--limit N] [--url <url>] [--json]
      Consulta legislação municipal, leis, decretos e portarias (portais Megasoft).
  concursos <nome_ou_ibge> [--uf <UF>] [--nome <filtro>] [--limit N] [--url <url>] [--json]
      Consulta aprovados em concursos e processos seletivos (portais Megasoft).
  transparencia <nome_ou_ibge> [--uf <UF>] [--ano YYYY] [--limit N] [--url <url>] [--json]
      Auditoria consolidada em paralelo (servidores, diárias, despesas, receitas, licitações, contratos, obras, convênios, legislação, concursos e SICONFI).
  fiscal <ibge_ou_nome> [--ano <ano>] [--tipo <dca|rreo|rgf>] [--periodo <p>] [--limit <N>] [--json]
      Consulta relatórios fiscais oficiais no SICONFI/Tesouro Nacional (DCA, RREO ou RGF).
  entregas <ibge_ou_nome> [--ano <ano>] [--json]
      Consulta cumprimento de entregas da Lei de Responsabilidade Fiscal no SICONFI.
  stats [--json]
      Estatísticas consolidadas nacionais sobre todos os municípios e estados.
  compare <id1> <id2>... [--json]
      Compara a estrutura de transparência entre múltiplos municípios ou estados.
  probe <nome_ou_ibge> [--uf <UF>] [--json]
      Testa conectividade e latência dos portais de transparência do alvo.
  serve [--port 8081]
      Inicia um microserviço HTTP nativo de alta performance com endpoints REST.`)
}

func parseFlexibleArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var flagArgs []string
	var posArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			name := strings.TrimLeft(arg, "-")
			if strings.Contains(name, "=") {
				continue
			}
			f := fs.Lookup(name)
			if f != nil {
				if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); !ok || !bf.IsBoolFlag() {
					if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
						i++
						flagArgs = append(flagArgs, args[i])
					}
				}
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}
	return posArgs, nil
}

func cmdListUFs(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("list-ufs", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	_, _ = parseFlexibleArgs(fs, args)

	ufs := engine.ListEstados()
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(ufs)
		return
	}

	fmt.Printf("%-4s %-25s %-12s %-6s %-35s\n", "UF", "NOME", "REGIÃO", "MUNIS", "TRIBUNAL DE CONTAS")
	fmt.Println(strings.Repeat("-", 95))
	for _, u := range ufs {
		fmt.Printf("%-4s %-25s %-12s %-6d %-35s\n", u.UF, u.Nome, u.Regiao, u.TotalMunicipios, u.TribunalContasNome)
	}
}

func cmdSearch(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	uf := fs.String("uf", "", "Filtrar por UF (ex: SP, MG, RJ)")
	limit := fs.Int("limit", 20, "Limite de resultados")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	query := strings.Join(posArgs, " ")

	start := time.Now()
	munis := engine.SearchMunicipios(query, *uf, *limit)
	elapsed := time.Since(start)

	if *asJSON {
		res := transparencia.SearchResult{
			Query:           query,
			UF:              *uf,
			TotalEncontrado: len(munis),
			Municipios:      munis,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}

	fmt.Printf("Encontrados %d municípios em %v:\n\n", len(munis), elapsed)
	fmt.Printf("%-8s %-30s %-4s %-12s %-6s %-6s %s\n", "IBGE", "MUNICÍPIO", "UF", "REGIÃO", "CAPITAL", "SIAFI", "PORTAL TRANSPARÊNCIA")
	fmt.Println(strings.Repeat("-", 105))
	for _, m := range munis {
		cap := "Não"
		if m.Capital {
			cap = "Sim"
		}
		portal := fmt.Sprintf("https://transparencia.%s.%s.gov.br", m.Slug, strings.ToLower(m.UF))
		fmt.Printf("%-8s %-30s %-4s %-12s %-6s %-6s %s\n", m.CodigoIBGE, m.Nome, m.UF, m.Regiao, cap, m.SiafiID, portal)
	}
}

func cmdGet(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Erro: especifique o código IBGE ou SIAFI")
		os.Exit(1)
	}

	code := posArgs[0]
	var m *transparencia.Municipio
	var err error

	if len(code) == 7 {
		m, err = engine.GetMunicipioByIBGE(code)
	} else {
		m, err = engine.GetMunicipioBySIAFI(code)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(m)
		return
	}

	fmt.Printf("Município: %s - %s\n", m.Nome, m.UF)
	fmt.Printf("Código IBGE: %s | SIAFI: %s | DDD: %s | Região: %s\n", m.CodigoIBGE, m.SiafiID, m.DDD, m.Regiao)
	fmt.Printf("Coordenadas: Lat %.4f, Long %.4f | Fuso: %s\n", m.Latitude, m.Longitude, m.FusoHorario)
}

func cmdConnectMunicipio(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("connect-municipio", flag.ExitOnError)
	uf := fs.String("uf", "", "Filtrar por UF")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Erro: especifique o nome ou código IBGE do município")
		os.Exit(1)
	}

	query := strings.Join(posArgs, " ")
	dossier, err := engine.ConnectMunicipio(query, *uf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(dossier)
		return
	}

	m := dossier.Municipio
	fmt.Printf("============================================================\n")
	fmt.Printf("  DOSSIÊ DE TRANSPARÊNCIA: %s (%s)\n", strings.ToUpper(m.Nome), m.UF)
	fmt.Printf("============================================================\n")
	fmt.Printf("• Código IBGE:                  %s\n", m.CodigoIBGE)
	fmt.Printf("• Código SIAFI:                 %s\n", m.SiafiID)
	fmt.Printf("• Região / DDD:                 %s / DDD %s\n", m.Regiao, m.DDD)
	fmt.Printf("• Portal da Transparência:      %s\n", dossier.PortalTransparenciaPrefeitura)
	fmt.Printf("• Site da Prefeitura:           %s\n", dossier.PortalPrefeitura)
	fmt.Printf("• Câmara Municipal:             %s\n", dossier.PortalCamara)
	fmt.Printf("• Tribunal de Contas:           %s\n", dossier.TribunalContasCompetente)
	fmt.Printf("  - URL do Tribunal:            %s\n", dossier.TribunalContasURL)
	fmt.Printf("• Radar Nacional Transparência: %s\n", dossier.RadarTransparenciaURL)
	fmt.Printf("• Dados Fiscais SICONFI:        %s\n", dossier.SiconfiURL)
	fmt.Printf("• Compras Públicas PNCP:        %s\n", dossier.PNCPURL)
	fmt.Printf("• Transferências Transferegov:  %s\n", dossier.TransferegovURL)
	fmt.Printf("• Transparência Federal (CGU):  %s\n", dossier.TransparenciaFederalURL)
	fmt.Printf("============================================================\n")
}

func cmdConnectEstado(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("connect-estado", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Erro: informe a sigla da UF (ex: SP, RJ, MG)")
		os.Exit(1)
	}

	uf := posArgs[0]
	dossier, err := engine.ConnectEstado(uf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(dossier)
		return
	}

	est := dossier.Estado
	fmt.Printf("============================================================\n")
	fmt.Printf("  DOSSIÊ DE TRANSPARÊNCIA ESTADUAL: %s (%s)\n", strings.ToUpper(est.Nome), est.UF)
	fmt.Printf("============================================================\n")
	fmt.Printf("• Código UF (IBGE):             %d\n", est.CodigoUF)
	fmt.Printf("• Região / Capital:             %s / %s\n", est.Regiao, est.Capital)
	fmt.Printf("• Total de Municípios:          %d\n", dossier.TotalMunicipios)
	fmt.Printf("• Portal da Transparência:      %s\n", dossier.PortalTransparencia)
	fmt.Printf("• Portal do Governo:            %s\n", dossier.PortalGoverno)
	fmt.Printf("• Tribunal de Contas (TCE):     %s\n", dossier.TribunalContasNome)
	fmt.Printf("  - URL do TCE:                 %s\n", dossier.TribunalContasURL)
	fmt.Printf("• Poder Legislativo:            %s\n", dossier.AssembleiaNome)
	fmt.Printf("  - URL da Assembleia:          %s\n", dossier.AssembleiaURL)
	fmt.Printf("• Radar Nacional Transparência: %s\n", dossier.RadarTransparenciaURL)
	fmt.Printf("• Dados Fiscais SICONFI:        %s\n", dossier.SiconfiURL)
	fmt.Printf("• Compras Públicas PNCP:        %s\n", dossier.PNCPURL)
	fmt.Printf("============================================================\n")
}

func cmdStats(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	_, _ = parseFlexibleArgs(fs, args)

	stats := engine.GetStats()
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(stats)
		return
	}

	fmt.Println("=== ESTATÍSTICAS NACIONAIS DE MUNICÍPIOS E UFs ===")
	fmt.Printf("Total de Municípios e Unidades Territoriais: %d\n", stats.TotalMunicipios)
	fmt.Printf("Total de Unidades Federativas (UFs):         %d\n\n", stats.TotalUFs)
	fmt.Println("Municípios por Região:")
	for reg, cnt := range stats.MunicipiosPorRegiao {
		fmt.Printf("  - %-15s %d municípios\n", reg, cnt)
	}
	fmt.Printf("\nTotal de Capitais Conectadas: %d\n", len(stats.Capitais))
}

func cmdCompare(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	if len(posArgs) < 2 {
		fmt.Fprintln(os.Stderr, "Erro: informe pelo menos dois municípios ou UFs para comparar")
		os.Exit(1)
	}

	rel, err := engine.CompareTransparencia(posArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rel)
		return
	}

	fmt.Println("=== COMPARATIVO DE ESTRUTURA DE TRANSPARÊNCIA ===")
	for i, item := range rel.Itens {
		fmt.Printf("\n[%d] %s: %s (%s) - Região %s\n", i+1, strings.ToUpper(item.Tipo), item.Nome, item.UF, item.Regiao)
		fmt.Printf("    • Código IBGE:           %s\n", item.CodigoIBGE)
		fmt.Printf("    • Transparência Direta:  %s\n", item.PortalTransparencia)
		fmt.Printf("    • Tribunal de Contas:    %s\n", item.TribunalContasNome)
		fmt.Printf("    • Compras PNCP:          %s\n", item.PNCPURL)
		fmt.Printf("    • Fiscais SICONFI:       %s\n", item.SiconfiURL)
	}
}

func cmdProbe(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	uf := fs.String("uf", "", "Filtrar por UF")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Erro: informe o município ou UF para testar conectividade")
		os.Exit(1)
	}

	alvo := strings.Join(posArgs, " ")
	fetcher := transparencia.NewFetcher(4 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	alvos := make(map[string]string)

	if len(alvo) == 2 {
		if est, err := engine.GetEstado(alvo); err == nil {
			alvos["Portal Transparência Estadual"] = est.PortalTransparencia
			alvos["Portal do Governo Estadual"] = est.PortalGoverno
			alvos["Tribunal de Contas"] = est.TribunalContasURL
		}
	} else {
		dos, err := engine.ConnectMunicipio(alvo, *uf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
		alvos["Transparência Prefeitura"] = dos.PortalTransparenciaPrefeitura
		alvos["Site Prefeitura"] = dos.PortalPrefeitura
		alvos["Câmara Municipal"] = dos.PortalCamara
		alvos["Tribunal de Contas Competente"] = dos.TribunalContasURL
	}

	results := fetcher.ProbeMany(ctx, alvos, 5)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		return
	}

	fmt.Printf("Resultados do teste de conectividade para '%s':\n\n", alvo)
	for _, r := range results {
		status := "❌ OFFLINE"
		if r.Online {
			status = fmt.Sprintf("✅ ONLINE (HTTP %d)", r.StatusHTTP)
		}
		fmt.Printf("• %-30s: %s [%dms] -> %s\n", r.Alvo, status, r.TempoMs, r.URL)
		if r.Erro != "" {
			fmt.Printf("    Detalhe: %s\n", r.Erro)
		}
	}
}

func cmdServe(engine *transparencia.Engine, args []string) {

	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8081, "Porta do servidor HTTP")
	_ = fs.Parse(args)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","engine":"go"}`))
	})

	mux.HandleFunc("/api/ufs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(engine.ListEstados())
	})

	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query().Get("q")
		uf := r.URL.Query().Get("uf")
		limStr := r.URL.Query().Get("limit")
		lim, _ := strconv.Atoi(limStr)
		if lim <= 0 {
			lim = 20
		}
		munis := engine.SearchMunicipios(q, uf, lim)
		_ = json.NewEncoder(w).Encode(transparencia.SearchResult{
			Query:           q,
			UF:              uf,
			TotalEncontrado: len(munis),
			Municipios:      munis,
		})
	})

	mux.HandleFunc("/api/municipio", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query().Get("q")
		uf := r.URL.Query().Get("uf")
		dossier, err := engine.ConnectMunicipio(q, uf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(dossier)
	})

	mux.HandleFunc("/api/estado", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		uf := r.URL.Query().Get("uf")
		dossier, err := engine.ConnectEstado(uf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(dossier)
	})

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(engine.GetStats())
	})

	mux.HandleFunc("/api/cnpj", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cnpj := r.URL.Query().Get("cnpj")
		m, err := engine.GetMunicipioByCNPJ(cnpj)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		est, _ := engine.GetEstado(m.UF)
		_ = json.NewEncoder(w).Encode(transparencia.BuildMunicipioDossier(m, est))
	})

	mux.HandleFunc("/api/fiscal", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ibge := r.URL.Query().Get("ibge")
		tipo := r.URL.Query().Get("tipo")
		anoStr := r.URL.Query().Get("ano")
		ano, _ := strconv.Atoi(anoStr)
		periodoStr := r.URL.Query().Get("periodo")
		periodo, _ := strconv.Atoi(periodoStr)

		client := transparencia.NewSiconfiClient()
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var items []transparencia.FiscalItem
		var err error
		switch strings.ToLower(tipo) {
		case "rreo":
			items, err = client.FetchRREO(ctx, ibge, ano, periodo)
		case "rgf":
			items, err = client.FetchRGF(ctx, ibge, ano, "", periodo)
		default:
			items, err = client.FetchDCA(ctx, ibge, ano)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(items)
	})

	mux.HandleFunc("/api/entregas", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ibge := r.URL.Query().Get("ibge")
		anoStr := r.URL.Query().Get("ano")
		ano, _ := strconv.Atoi(anoStr)

		client := transparencia.NewSiconfiClient()
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		items, err := client.FetchEntregas(ctx, ibge, ano)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(items)
	})

	hub := transparencia.NewHub(engine)

	parseHTTPOpts := func(r *http.Request) (string, transparencia.QueryOptions) {
		q := r.URL.Query().Get("q")
		if q == "" {
			q = r.URL.Query().Get("municipio")
		}
		ano, _ := strconv.Atoi(r.URL.Query().Get("ano"))
		mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))
		lim, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		return q, transparencia.QueryOptions{
			UF:          r.URL.Query().Get("uf"),
			OverrideURL: r.URL.Query().Get("url"),
			NomeFiltro:  r.URL.Query().Get("nome"),
			Inicio:      r.URL.Query().Get("inicio"),
			Fim:         r.URL.Query().Get("fim"),
			Ano:         ano,
			Mes:         mes,
			Limit:       lim,
		}
	}

	mux.HandleFunc("/api/discover", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		res, err := hub.DiscoverMunicipioPortal(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/servidores", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryServidores(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/despesas", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryDespesas(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/receitas", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryReceitas(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/licitacoes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryLicitacoes(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/contratos", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryContratos(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/megasoft", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		uf := r.URL.Query().Get("uf")
		_ = json.NewEncoder(w).Encode(engine.ListMegasoftMunicipios(uf))
	})

	mux.HandleFunc("/api/diarias", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryDiarias(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/obras", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryObras(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/convenios", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryConvenios(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/legislacao", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryLegislacao(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/concursos", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryConcursos(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/servidor-detalhe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		mat := r.URL.Query().Get("matricula")
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		res, err := hub.QueryServidorDetalhe(ctx, q, mat, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/transparencia", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q, opts := parseHTTPOpts(r)
		ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
		defer cancel()
		res, err := hub.QueryTransparenciaCompleta(ctx, q, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Iniciando servidor nativo de transparência em Go em http://localhost%s...\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao rodar servidor: %v\n", err)
		os.Exit(1)
	}
}

func cmdCNPJ(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("cnpj", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Erro: informe o CNPJ a consultar")
		os.Exit(1)
	}

	cnpj := posArgs[0]
	m, err := engine.GetMunicipioByCNPJ(cnpj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	est, _ := engine.GetEstado(m.UF)
	dossier := transparencia.BuildMunicipioDossier(m, est)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(dossier)
		return
	}

	fmt.Printf("Município encontrado para o CNPJ %s:\n\n", cnpj)
	fmt.Printf("• Município: %s (%s)\n", m.Nome, m.UF)
	fmt.Printf("• Código IBGE: %s | SIAFI: %s\n", m.CodigoIBGE, m.SiafiID)
	fmt.Printf("• População: %d habitantes\n", m.Populacao)
	fmt.Printf("• Portal Transparência: %s\n", dossier.PortalTransparenciaPrefeitura)
	fmt.Printf("• Tribunal de Contas: %s (%s)\n", dossier.TribunalContasCompetente, dossier.TribunalContasURL)
	if dossier.PNCPCNPJURL != "" {
		fmt.Printf("• Editais e Compras PNCP: %s\n", dossier.PNCPCNPJURL)
	}
	fmt.Printf("• Balanço SICONFI: %s\n", dossier.SiconfiURL)
}

func cmdFiscal(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("fiscal", flag.ExitOnError)
	ibge := fs.String("ibge", "", "Código IBGE do ente municipal ou estadual")
	ano := fs.Int("ano", 0, "Ano de exercício fiscal (padrão: ano anterior)")
	tipo := fs.String("tipo", "dca", "Tipo de relatório: dca, rreo ou rgf")
	periodo := fs.Int("periodo", 1, "Período/Bimestre")
	limit := fs.Int("limit", 25, "Limite de linhas a exibir na tabela")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	target := *ibge
	if target == "" && len(posArgs) > 0 {
		target = posArgs[0]
	}

	if target == "" {
		fmt.Fprintln(os.Stderr, "Erro: informe o código IBGE com --ibge <cod> ou posicional")
		os.Exit(1)
	}

	// Se target não for puramente numérico, pesquisa o município
	if _, err := strconv.Atoi(target); err != nil {
		munis := engine.SearchMunicipios(target, "", 1)
		if len(munis) == 0 {
			fmt.Fprintf(os.Stderr, "Município não encontrado: %s\n", target)
			os.Exit(1)
		}
		target = munis[0].CodigoIBGE
	}

	client := transparencia.NewSiconfiClient()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var items []transparencia.FiscalItem
	var err error
	switch strings.ToLower(*tipo) {
	case "rreo":
		items, err = client.FetchRREO(ctx, target, *ano, *periodo)
	case "rgf":
		items, err = client.FetchRGF(ctx, target, *ano, "", 0)
	default:
		items, err = client.FetchDCA(ctx, target, *ano)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao consultar dados fiscais: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(items)
		return
	}

	tipoUpper := strings.ToUpper(*tipo)
	fmt.Printf("Dados fiscais SICONFI (%s) para IBGE %s (%d itens encontrados):\n\n", tipoUpper, target, len(items))
	fmt.Printf("%-6s %-18s %-42s %-25s %s\n", "ANO", "CÓDIGO CONTA", "CONTA / DESCRIÇÃO", "COLUNA", "VALOR (R$)")
	fmt.Println(strings.Repeat("-", 120))

	showCount := len(items)
	if *limit > 0 && showCount > *limit {
		showCount = *limit
	}

	for i := 0; i < showCount; i++ {
		it := items[i]
		fmt.Printf("%-6d %-18s %-42s %-25s R$ %15.2f\n",
			it.Exercicio, it.CodConta, truncateStr(it.Conta, 42), truncateStr(it.Coluna, 25), it.Valor)
	}
	if len(items) > showCount {
		fmt.Printf("\n... e mais %d contas fiscais (use --limit para ver mais ou --json para dump completo).\n", len(items)-showCount)
	}
}

func cmdEntregas(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("entregas", flag.ExitOnError)
	ibge := fs.String("ibge", "", "Código IBGE do município")
	ano := fs.Int("ano", 0, "Ano de referência (padrão: ano corrente)")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	target := *ibge
	if target == "" && len(posArgs) > 0 {
		target = posArgs[0]
	}

	if target == "" {
		fmt.Fprintln(os.Stderr, "Erro: informe o código IBGE com --ibge <cod> ou posicional")
		os.Exit(1)
	}

	if _, err := strconv.Atoi(target); err != nil {
		munis := engine.SearchMunicipios(target, "", 1)
		if len(munis) == 0 {
			fmt.Fprintf(os.Stderr, "Município não encontrado: %s\n", target)
			os.Exit(1)
		}
		target = munis[0].CodigoIBGE
	}

	client := transparencia.NewSiconfiClient()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	items, err := client.FetchEntregas(ctx, target, *ano)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao consultar extrato de entregas: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(items)
		return
	}

	fmt.Printf("Extrato de Entregas da LRF no SICONFI para IBGE %s (%d entregas registradas):\n\n", target, len(items))
	fmt.Printf("%-6s %-25s %-8s %-12s %-10s %s\n", "ANO", "ENTREGÁVEL", "PERÍODO", "DATA STATUS", "FORMA", "INSTITUIÇÃO")
	fmt.Println(strings.Repeat("-", 100))

	for _, it := range items {
		periodoStr := fmt.Sprintf("%d (%s)", it.Periodo, it.Periodicidade)
		dataStr := it.DataStatus
		if len(dataStr) > 10 {
			dataStr = dataStr[:10]
		}
		fmt.Printf("%-6d %-25s %-8s %-12s %-10s %s\n",
			it.Exercicio, truncateStr(it.Entregavel, 25), periodoStr, dataStr, it.FormaEnvio, truncateStr(it.Instituicao, 30))
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func parseHubCommandFlags(name string, args []string) (string, transparencia.QueryOptions, bool) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	uf := fs.String("uf", "", "Sigla da UF")
	overrideURL := fs.String("url", "", "URL base explícita do portal municipal")
	nome := fs.String("nome", "", "Filtro de texto/nome")
	inicio := fs.String("inicio", "", "Data inicial YYYY-MM-DD")
	fim := fs.String("fim", "", "Data final YYYY-MM-DD")
	ano := fs.Int("ano", 0, "Ano de referência")
	mes := fs.Int("mes", 0, "Mês de referência (1-12)")
	limit := fs.Int("limit", 20, "Limite de registros")
	ordemSalarioDesc := fs.Bool("ordem-salario-desc", false, "Ordenar servidores pela maior remuneração bruta")
	ordemValorDesc := fs.Bool("ordem-valor-desc", false, "Ordenar despesas pelo maior valor")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	target := strings.Join(posArgs, " ")
	if target == "" {
		fmt.Fprintf(os.Stderr, "Erro: especifique o nome ou código IBGE do município para '%s'\n", name)
		os.Exit(1)
	}
	return target, transparencia.QueryOptions{
		UF:               *uf,
		OverrideURL:      *overrideURL,
		NomeFiltro:       *nome,
		Inicio:           *inicio,
		Fim:              *fim,
		Ano:              *ano,
		Mes:              *mes,
		Limit:            *limit,
		OrdemSalarioDesc: *ordemSalarioDesc,
		OrdemValorDesc:   *ordemValorDesc,
	}, *asJSON
}

func cmdDiscover(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("discover", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res, err := hub.DiscoverMunicipioPortal(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Descoberta de Portal: %s (%s) [%dms]\n", res.Municipio.Nome, res.Municipio.UF, res.TempoMs)
	fmt.Printf("• URL Resolvida:        %s\n", res.URLResolvida)
	fmt.Printf("• Plataforma Detectada: %s (versão: %s)\n", res.Plataforma, res.Versao)
	fmt.Printf("• Servidores API:       %v\n", res.ServidoresSuportado)
	if res.EntidadeNome != "" {
		fmt.Printf("• Entidade / Cód Adm:   %s (cod=%d)\n", res.EntidadeNome, res.CodAdministracao)
	}
	if res.Observacoes != "" {
		fmt.Printf("• Observações:          %s\n", res.Observacoes)
	}
}

func cmdServidores(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("servidores", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryServidores(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Servidores Públicos: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Servidores) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-32s %-25s %-22s %-12s %-12s %s\n", "NOME", "CARGO", "LOTAÇÃO", "BRUTO (R$)", "LÍQUIDO (R$)", "COMP.")
	fmt.Println(strings.Repeat("-", 118))
	for _, s := range res.Servidores {
		fmt.Printf("%-32s %-25s %-22s %12.2f %12.2f %s\n",
			truncateStr(s.Nome, 32), truncateStr(s.Cargo, 25), truncateStr(s.Lotacao, 22),
			s.SalarioBruto, s.SalarioLiquido, s.Competencia)
	}
}

func cmdDespesas(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("despesas", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryDespesas(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Despesas: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Despesas) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-14s %-34s %-24s %14s %14s\n", "EMPENHO", "FORNECEDOR", "UNIDADE/FUNÇÃO", "EMPENHADO", "PAGO")
	fmt.Println(strings.Repeat("-", 106))
	for _, d := range res.Despesas {
		unid := d.Unidade
		if unid == "" {
			unid = d.Funcao
		}
		fmt.Printf("%-14s %-34s %-24s %14.2f %14.2f\n",
			truncateStr(d.NumeroEmpenho, 14), truncateStr(d.Fornecedor, 34), truncateStr(unid, 24),
			d.ValorEmpenhado, d.ValorPago)
	}
}

func cmdReceitas(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("receitas", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryReceitas(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Receitas: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Receitas) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-16s %-45s %16s %16s\n", "CÓDIGO", "DESCRIÇÃO", "PREVISTO (R$)", "ARRECADADO (R$)")
	fmt.Println(strings.Repeat("-", 98))
	for _, r := range res.Receitas {
		fmt.Printf("%-16s %-45s %16.2f %16.2f\n",
			truncateStr(r.Codigo, 16), truncateStr(r.Descricao, 45), r.ValorPrevisto, r.ValorArrecadado)
	}
}

func cmdLicitacoes(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("licitacoes", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryLicitacoes(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Licitações: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if len(res.Licitacoes) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-14s %-22s %-12s %-48s %14s\n", "NÚMERO", "MODALIDADE", "ABERTURA", "OBJETO", "ESTIMADO (R$)")
	fmt.Println(strings.Repeat("-", 116))
	for _, l := range res.Licitacoes {
		fmt.Printf("%-14s %-22s %-12s %-48s %14.2f\n",
			truncateStr(l.Numero, 14), truncateStr(l.Modalidade, 22), truncateStr(l.DataAbertura, 12),
			truncateStr(l.Objeto, 48), l.ValorEstimado)
	}
}

func cmdContratos(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("contratos", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryContratos(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Contratos: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if len(res.Contratos) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-14s %-30s %-12s %-42s %14s\n", "NÚMERO", "FORNECEDOR", "ASSINATURA", "OBJETO", "VALOR (R$)")
	fmt.Println(strings.Repeat("-", 118))
	for _, c := range res.Contratos {
		fmt.Printf("%-14s %-30s %-12s %-42s %14.2f\n",
			truncateStr(c.Numero, 14), truncateStr(c.Fornecedor, 30), truncateStr(c.DataAssinatura, 12),
			truncateStr(c.Objeto, 42), c.Valor)
	}
}

func cmdMegasoftList(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("megasoft-list", flag.ExitOnError)
	uf := fs.String("uf", "", "Filtrar por UF (ex: GO, TO, MS)")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	_, _ = parseFlexibleArgs(fs, args)

	items := engine.ListMegasoftMunicipios(*uf)
	if *asJSON {
		printJSON(items)
		return
	}
	fmt.Printf("Municípios Megasoft Verificados (%d encontrados):\n\n", len(items))
	fmt.Printf("%-8s %-30s %-4s %-45s %s\n", "IBGE", "MUNICÍPIO", "UF", "PORTAL TRANSPARÊNCIA", "CÂMARA")
	fmt.Println(strings.Repeat("-", 125))
	for _, it := range items {
		fmt.Printf("%-8s %-30s %-4s %-45s %s\n",
			it.CodigoIBGE, truncateStr(it.Nome, 30), it.UF, truncateStr(it.PortalURL, 45), truncateStr(it.PortalCamara, 40))
	}
}

func cmdDiarias(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("diarias", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryDiarias(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Diárias: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Diarias) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-28s %-22s %-22s %-12s %12s\n", "BENEFICIÁRIO", "CARGO", "DESTINO", "SAÍDA", "VALOR (R$)")
	fmt.Println(strings.Repeat("-", 102))
	for _, d := range res.Diarias {
		fmt.Printf("%-28s %-22s %-22s %-12s %12.2f\n",
			truncateStr(d.Beneficiario, 28), truncateStr(d.Cargo, 22), truncateStr(d.Destino, 22),
			truncateStr(d.DataSaida, 12), d.Valor)
	}
}

func cmdObras(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("obras", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryObras(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Obras Públicas: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Obras) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-8s %-42s %-20s %-14s %14s\n", "CÓDIGO", "TÍTULO / OBJETO", "SITUAÇÃO", "INÍCIO", "VALOR (R$)")
	fmt.Println(strings.Repeat("-", 104))
	for _, o := range res.Obras {
		fmt.Printf("%-8s %-42s %-20s %-14s %14.2f\n",
			truncateStr(o.Codigo, 8), truncateStr(o.Titulo, 42), truncateStr(o.Situacao, 20),
			truncateStr(o.DataInicial, 14), o.Valor)
	}
}

func cmdConvenios(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("convenios", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryConvenios(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Convênios: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Convenios) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-14s %-26s %-36s %-16s %14s\n", "NÚMERO", "CONCEDENTE", "DESCRIÇÃO", "SITUAÇÃO", "VALOR (R$)")
	fmt.Println(strings.Repeat("-", 112))
	for _, c := range res.Convenios {
		fmt.Printf("%-14s %-26s %-36s %-16s %14.2f\n",
			truncateStr(c.Numero, 14), truncateStr(c.Concedente, 26), truncateStr(c.Descricao, 36),
			truncateStr(c.Situacao, 16), c.Valor)
	}
}

func cmdLegislacao(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("legislacao", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryLegislacao(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Legislação Municipal: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Atos) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-18s %-12s %-6s %-12s %s\n", "TIPO", "NÚMERO", "ANO", "PUBLICAÇÃO", "DESCRIÇÃO")
	fmt.Println(strings.Repeat("-", 105))
	for _, a := range res.Atos {
		fmt.Printf("%-18s %-12s %-6d %-12s %s\n",
			truncateStr(a.TipoDocumento, 18), truncateStr(a.Numero, 12), a.Ano,
			truncateStr(a.DataPublicacao, 12), truncateStr(a.Descricao, 50))
	}
}

func cmdConcursos(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("concursos", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryConcursos(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Concursos e Seletivos: %s (%s) | Fonte: %s | Total: %d\n",
		res.Municipio.Nome, res.Municipio.UF, res.Fonte, res.TotalRegistros)
	if !res.Disponivel || len(res.Aprovados) == 0 {
		fmt.Println("Info:", res.Mensagem)
		return
	}
	fmt.Printf("%-10s %-30s %-26s %-6s %s\n", "EDITAL", "CANDIDATO", "CARGO", "CLASS.", "SITUAÇÃO")
	fmt.Println(strings.Repeat("-", 96))
	for _, c := range res.Aprovados {
		fmt.Printf("%-10s %-30s %-26s %-6d %s\n",
			truncateStr(c.Edital, 10), truncateStr(c.Nome, 30), truncateStr(c.Cargo, 26),
			c.Classificacao, truncateStr(c.Situacao, 20))
	}
}

func cmdServidorDetalhe(engine *transparencia.Engine, args []string) {
	fs := flag.NewFlagSet("servidor-detalhe", flag.ExitOnError)
	uf := fs.String("uf", "", "Sigla da UF")
	overrideURL := fs.String("url", "", "URL base explícita do portal municipal")
	matricula := fs.String("matricula", "", "Matrícula do servidor")
	ano := fs.Int("ano", 0, "Ano de referência")
	mes := fs.Int("mes", 0, "Mês de referência (1-12)")
	asJSON := fs.Bool("json", false, "Exibir em formato JSON")
	posArgs, _ := parseFlexibleArgs(fs, args)

	target := strings.Join(posArgs, " ")
	if target == "" || strings.TrimSpace(*matricula) == "" {
		fmt.Fprintln(os.Stderr, "Erro: especifique o município e --matricula <mat>")
		os.Exit(1)
	}
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := hub.QueryServidorDetalhe(ctx, target, *matricula, transparencia.QueryOptions{
		UF:          *uf,
		OverrideURL: *overrideURL,
		Ano:         *ano,
		Mes:         *mes,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if *asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Contracheque Detalhado: %s (%s) — Matrícula %s | Comp: %s\n",
		res.Nome, res.Cargo, res.Matricula, res.Competencia)
	fmt.Printf("Órgão: %s | Departamento: %s | Vínculo: %s\n", res.Orgao, res.Departamento, res.Vinculo)
	fmt.Printf("Bruto: R$ %.2f | Descontos: R$ %.2f | Líquido: R$ %.2f\n",
		res.SalarioBruto, res.Descontos, res.SalarioLiquido)
	if len(res.Vencimentos) > 0 {
		fmt.Println("\nVencimentos:")
		for _, v := range res.Vencimentos {
			fmt.Printf("  + %-45s R$ %10.2f\n", truncateStr(v.Descricao, 45), v.Valor)
		}
	}
	if len(res.VerbasDesconto) > 0 {
		fmt.Println("\nDescontos:")
		for _, v := range res.VerbasDesconto {
			fmt.Printf("  - %-45s R$ %10.2f\n", truncateStr(v.Descricao, 45), v.Valor)
		}
	}
}

func cmdTransparenciaCompleta(engine *transparencia.Engine, args []string) {
	target, opts, asJSON := parseHubCommandFlags("transparencia", args)
	hub := transparencia.NewHub(engine)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	rep, err := hub.QueryTransparenciaCompleta(ctx, target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		printJSON(rep)
		return
	}
	m := rep.Dossier.Municipio
	fmt.Printf("=== AUDITORIA COMPLETA DE TRANSPARÊNCIA: %s (%s) [%dms] ===\n", m.Nome, m.UF, rep.TempoTotalMs)
	fmt.Printf("IBGE: %s | CNPJ: %s | População: %d\n", m.CodigoIBGE, m.CNPJ, m.Populacao)
	if rep.Discovery != nil {
		fmt.Printf("Portal Resolvido: %s (plataforma: %s)\n", rep.Discovery.URLResolvida, rep.Discovery.Plataforma)
	}
	fmt.Printf("Fontes Consultadas (%d): %s\n", len(rep.FontesConsultadas), strings.Join(rep.FontesConsultadas, "; "))
	if rep.Servidores != nil {
		fmt.Printf("• Servidores:   %d registros retornados (total=%d, disponível=%v)\n",
			len(rep.Servidores.Servidores), rep.Servidores.TotalRegistros, rep.Servidores.Disponivel)
	}
	if rep.Diarias != nil {
		fmt.Printf("• Diárias:      %d registros retornados (total=%d)\n",
			len(rep.Diarias.Diarias), rep.Diarias.TotalRegistros)
	}
	if rep.Despesas != nil {
		fmt.Printf("• Despesas:     %d registros retornados (total=%d, disponível=%v)\n",
			len(rep.Despesas.Despesas), rep.Despesas.TotalRegistros, rep.Despesas.Disponivel)
	}
	if rep.Receitas != nil {
		fmt.Printf("• Receitas:     %d registros retornados (total=%d, disponível=%v)\n",
			len(rep.Receitas.Receitas), rep.Receitas.TotalRegistros, rep.Receitas.Disponivel)
	}
	if rep.Licitacoes != nil {
		fmt.Printf("• Licitações:   %d registros retornados (total=%d)\n",
			len(rep.Licitacoes.Licitacoes), rep.Licitacoes.TotalRegistros)
	}
	if rep.Contratos != nil {
		fmt.Printf("• Contratos:    %d registros retornados (total=%d)\n",
			len(rep.Contratos.Contratos), rep.Contratos.TotalRegistros)
	}
	if rep.Obras != nil {
		fmt.Printf("• Obras:        %d registros retornados (total=%d)\n",
			len(rep.Obras.Obras), rep.Obras.TotalRegistros)
	}
	if rep.Convenios != nil {
		fmt.Printf("• Convênios:    %d registros retornados (total=%d)\n",
			len(rep.Convenios.Convenios), rep.Convenios.TotalRegistros)
	}
	if rep.Legislacao != nil {
		fmt.Printf("• Legislação:   %d registros retornados (total=%d)\n",
			len(rep.Legislacao.Atos), rep.Legislacao.TotalRegistros)
	}
	if rep.Concursos != nil {
		fmt.Printf("• Concursos:    %d registros retornados (total=%d)\n",
			len(rep.Concursos.Aprovados), rep.Concursos.TotalRegistros)
	}
	fmt.Printf("• SICONFI:      DCA=%d contas | RGF(LRF Pessoal)=%d contas | Entregas=%d\n",
		len(rep.SiconfiDCA), len(rep.SiconfiRGF), len(rep.SiconfiEntregas))
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
