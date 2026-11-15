package transparencia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Sonner Sistemas (/GRP/pic/api) Connector
// CSRF handshake (/auth/csrf) + Funcionalidade tree + Service APIs
// ---------------------------------------------------------------------------

const (
	sonnerUserAgent  = "Mozilla/5.0 (compatible; mcp-brasil/1.0)"
	sonnerFonte      = "Portal Municipal (Sonner Sistemas)"
	sonnerPlataforma = "sonner"
)

var sonnerTargetRoutes = []string{
	"consulta-licitacao",
	"consulta-contrato",
	"listagem-obra",
	"cbp-convenios",
	"visualiza-salario",
}

func normalizeSonnerRoot(baseURL string) string {
	raw := strings.TrimSpace(baseURL)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}
	u.Fragment = ""
	u.RawQuery = ""
	path := strings.TrimRight(u.Path, "/")
	suffixes := []string{"/grp/pic/api", "/grp/pic", "/grp", "/portalcidadao", "/portal"}
	changed := true
	for changed {
		changed = false
		low := strings.ToLower(path)
		for _, suf := range suffixes {
			if strings.HasSuffix(low, suf) {
				path = strings.TrimRight(path[:len(path)-len(suf)], "/")
				changed = true
				break
			}
		}
	}
	u.Path = path
	return strings.TrimRight(u.String(), "/")
}

// ProbeSonner verifica se a URL base responde como um portal Sonner Sistemas (/GRP/pic/api/config/contexto).
func ProbeSonner(ctx context.Context, client *http.Client, baseURL string) (entidadeNome string, ok bool) {
	root := normalizeSonnerRoot(baseURL)
	if root == "" {
		return "", false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+"/GRP/pic/api/config/contexto", nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", sonnerUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := ensureHTTPClient(client).Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", false
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil || len(obj) == 0 {
		return "", false
	}

	_, hasDadosOrgao := obj["dadosOrgao"]
	_, hasMunicipio := obj["municipio"]
	_, hasExercicio := obj["exercicioAtual"]
	if !hasDadosOrgao && !hasMunicipio && !hasExercicio {
		return "", false
	}

	if orgaoMap, okMap := obj["dadosOrgao"].(map[string]any); okMap {
		entidadeNome = mapGetStr(orgaoMap, "nome", "nomeOrgao", "razaoSocial", "descricao", "municipio", "entidade")
	}
	if entidadeNome == "" {
		entidadeNome = mapGetStr(obj, "municipio", "nomeOrgao", "entidade", "razaoSocial", "nome")
	}
	return entidadeNome, true
}

// sonnerSession encapsula sessão CSRF, cookies e resolução de chaves X-Portal-Funcionalidade.
type sonnerSession struct {
	root      string
	client    *http.Client
	tenantID  string
	mu        sync.Mutex
	cookies   map[string]*http.Cookie
	xsrfToken string
	funcKeys  map[string]string
}

func newSonnerSession(ctx context.Context, client *http.Client, baseURL, tenantID string) (*sonnerSession, error) {
	s := &sonnerSession{
		root:     normalizeSonnerRoot(baseURL),
		client:   ensureHTTPClient(client),
		tenantID: strings.TrimSpace(tenantID),
		cookies:  make(map[string]*http.Cookie),
		funcKeys: make(map[string]string),
	}

	// 1. Handshake CSRF: GET /GRP/pic/api/auth/csrf
	if err := s.fetchCSRF(ctx); err != nil {
		return nil, err
	}

	// 2. Descoberta da árvore de funcionalidades: GET /GRP/pic/api/funcionalidade
	_ = s.resolveFuncionalidades(ctx)

	return s, nil
}

func (s *sonnerSession) captureResponseCookies(resp *http.Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range resp.Cookies() {
		if c == nil || c.Name == "" {
			continue
		}
		s.cookies[c.Name] = c
		if c.Name == "PORTAL-XSRF-TOKEN" && c.Value != "" {
			s.xsrfToken = c.Value
		} else if c.Name == "XSRF-TOKEN" && s.xsrfToken == "" && c.Value != "" {
			s.xsrfToken = c.Value
		}
	}
}

func (s *sonnerSession) attachRequestHeaders(req *http.Request, targetRoute string, isPost bool) {
	req.Header.Set("User-Agent", sonnerUserAgent)
	req.Header.Set("Accept", "application/json")
	s.mu.Lock()
	for _, c := range s.cookies {
		req.AddCookie(c)
	}
	if isPost && s.xsrfToken != "" {
		req.Header.Set("X-PORTAL-XSRF-TOKEN", s.xsrfToken)
	}
	if targetRoute != "" {
		if key, ok := s.funcKeys[targetRoute]; ok && key != "" {
			req.Header.Set("X-Portal-Funcionalidade", key)
		}
	}
	s.mu.Unlock()
	if s.tenantID != "" {
		req.Header.Set("X-Tenant-Id", s.tenantID)
	}
}

func (s *sonnerSession) doGet(ctx context.Context, path, targetRoute string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.root+path, nil)
	if err != nil {
		return nil, 0, err
	}
	s.attachRequestHeaders(req, targetRoute, false)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	s.captureResponseCookies(resp)

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func (s *sonnerSession) doPostJSON(ctx context.Context, path, targetRoute string, payload any) ([]byte, int, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.root+path, bytes.NewReader(rawPayload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	s.attachRequestHeaders(req, targetRoute, true)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	s.captureResponseCookies(resp)

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func (s *sonnerSession) fetchCSRF(ctx context.Context) error {
	body, status, err := s.doGet(ctx, "/GRP/pic/api/auth/csrf", "")
	if err != nil {
		return err
	}
	if status == http.StatusOK && len(bytes.TrimSpace(body)) > 0 {
		s.mu.Lock()
		hasTok := s.xsrfToken != ""
		s.mu.Unlock()
		if !hasTok {
			var obj map[string]any
			if json.Unmarshal(body, &obj) == nil {
				if tok := mapGetStr(obj, "token", "csrfToken", "xsrfToken"); tok != "" {
					s.mu.Lock()
					if s.xsrfToken == "" {
						s.xsrfToken = tok
					}
					s.mu.Unlock()
				}
			}
		}
	}
	return nil
}

type funcQueueItem struct {
	nodeID string
	depth  int
}

func (s *sonnerSession) funcKeysCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.funcKeys)
}

func (s *sonnerSession) resolveFuncionalidades(ctx context.Context) error {
	const (
		maxDepth = 4
		maxCalls = 30
	)
	visited := make(map[string]struct{})
	queue := []funcQueueItem{{nodeID: "", depth: 0}}
	calls := 0

	for len(queue) > 0 && calls < maxCalls && s.funcKeysCount() < len(sonnerTargetRoutes) {
		cur := queue[0]
		queue = queue[1:]

		path := "/GRP/pic/api/funcionalidade"
		if cur.nodeID != "" {
			if _, seen := visited[cur.nodeID]; seen {
				continue
			}
			visited[cur.nodeID] = struct{}{}
			path += "?node=" + url.QueryEscape(cur.nodeID)
		}

		calls++
		body, status, err := s.doGet(ctx, path, "")
		if err != nil || status != http.StatusOK {
			continue
		}

		nodes := parseSonnerNodes(body)
		s.walkFuncionalidadeNodes(nodes, cur.depth, maxDepth, visited, &queue)
	}
	return nil
}

func parseSonnerNodes(body []byte) []map[string]any {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var nodes []map[string]any
		if json.Unmarshal([]byte(trimmed), &nodes) == nil {
			return nodes
		}
		return nil
	}
	var obj map[string]any
	if json.Unmarshal([]byte(trimmed), &obj) != nil {
		return nil
	}
	for _, k := range []string{"children", "items", "data", "registros", "funcionalidades"} {
		if arr, ok := obj[k].([]any); ok {
			out := make([]map[string]any, 0, len(arr))
			for _, el := range arr {
				if m, ok := el.(map[string]any); ok {
					out = append(out, m)
				}
			}
			return out
		}
	}
	return []map[string]any{obj}
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func extractSonnerFuncKey(node map[string]any) string {
	keys := []string{"key", "funcionalidadeId", "idFuncionalidade", "codigo", "id"}
	for _, k := range keys {
		val := mapGetStr(node, k)
		if isAllDigits(val) {
			return val
		}
	}
	for _, k := range keys {
		if val := mapGetStr(node, k); val != "" {
			return val
		}
	}
	return ""
}

func (s *sonnerSession) walkFuncionalidadeNodes(nodes []map[string]any, depth, maxDepth int, visited map[string]struct{}, queue *[]funcQueueItem) {
	for _, node := range nodes {
		routeVal := strings.ToLower(strings.Join([]string{
			mapGetStr(node, "url"),
			mapGetStr(node, "rota"),
			mapGetStr(node, "route"),
			mapGetStr(node, "caminho"),
			mapGetStr(node, "path"),
			mapGetStr(node, "link"),
			mapGetStr(node, "servico"),
			mapGetStr(node, "acao"),
			mapGetStr(node, "componente"),
			mapGetStr(node, "codigoRota"),
		}, " "))

		funcKey := extractSonnerFuncKey(node)
		if funcKey != "" && routeVal != "" {
			s.mu.Lock()
			for _, target := range sonnerTargetRoutes {
				if strings.Contains(routeVal, target) {
					if _, exists := s.funcKeys[target]; !exists {
						s.funcKeys[target] = funcKey
					}
				}
			}
			s.mu.Unlock()
		}

		// Verifica filhos inline
		var inlineChildren []map[string]any
		for _, ck := range []string{"children", "filhos", "items", "submenus"} {
			if rawArr, ok := node[ck].([]any); ok && len(rawArr) > 0 {
				for _, el := range rawArr {
					if cm, ok := el.(map[string]any); ok {
						inlineChildren = append(inlineChildren, cm)
					}
				}
				break
			}
		}
		if len(inlineChildren) > 0 {
			s.walkFuncionalidadeNodes(inlineChildren, depth+1, maxDepth, visited, queue)
			continue
		}

		// Se não for folha e tiver ID, enfileira busca lazy ?node=<id>
		isLeaf, hasLeaf := node["leaf"].(bool)
		if hasLeaf && isLeaf {
			continue
		}
		if depth+1 <= maxDepth {
			nodeID := mapGetStr(node, "id", "key", "codigo")
			if nodeID != "" {
				if _, seen := visited[nodeID]; !seen {
					*queue = append(*queue, funcQueueItem{nodeID: nodeID, depth: depth + 1})
				}
			}
		}
	}
}

func resolveSonnerYears(p PortalQueryParams) []int {
	if p.Ano > 0 {
		return []int{p.Ano}
	}
	if len(p.Inicio) >= 4 {
		if y, err := strconv.Atoi(p.Inicio[:4]); err == nil && y > 1900 {
			return []int{y}
		}
	}
	nowY := time.Now().Year()
	return []int{nowY, nowY - 1, nowY - 2}
}

func decodeSonnerObjectOrArray(body []byte) (rows []map[string]any, obj map[string]any, total int, err error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return nil, nil, 0, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, nil, 0, err
		}
		return rows, nil, len(rows), nil
	}
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
			return nil, nil, 0, err
		}
		total = mapGetInt(obj, "total", "totalElements", "totalRegistros", "count")
		for _, key := range []string{"registros", "Registros", "content", "dados", "data", "items", "lista", "funcionarios", "despesas", "receitas", "licitacoes", "contratos", "obras", "convenios", "series", "categorias"} {
			if raw, ok := obj[key]; ok {
				if arr, ok := raw.([]any); ok {
					out := make([]map[string]any, 0, len(arr))
					for _, el := range arr {
						if item, ok := el.(map[string]any); ok {
							out = append(out, item)
						}
					}
					if total == 0 {
						total = len(out)
					}
					return out, obj, total, nil
				}
			}
		}
		return nil, obj, total, nil
	}
	return nil, nil, 0, fmt.Errorf("resposta Sonner inválida: %.64s", trimmed)
}

func stripSonnerSensitiveFields(m map[string]any) {
	for _, k := range []string{
		"cpf", "numCpf", "cpfFormatado",
		"endereco", "logradouro", "numero", "complemento", "bairro", "cep", "cidade", "uf",
		"banco", "agencia", "conta", "contaCorrente", "digito", "pix", "dadosBancarios",
	} {
		delete(m, k)
	}
}

// QuerySonnerServidores consulta servidores no endpoint Sonner /GRP/pic/api/visualiza-salario/funcionario,
// enriquecendo registros sem remuneração com /anos-folha e /matriculafolhas (até 10 sequencialmente ou 100 em pool quando OrdemSalarioDesc=true) e removendo/mascarando dados sensíveis (LGPD).
func QuerySonnerServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	sess, err := newSonnerSession(ctx, client, p.BaseURL, p.TenantID)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(p.Limit)
	filtro := strings.TrimSpace(p.NomeFiltro)

	tokens := []string{filtro}
	if filtro == "" {
		if p.OrdemSalarioDesc {
			tokens = []string{" ", "A", "E", "I", "O", "U"}
		} else {
			tokens = []string{"A", " "}
		}
	}

	var rows []map[string]any
	var total int
	var lastStatus int
	seenRows := make(map[string]struct{})

	for _, tok := range tokens {
		path := "/GRP/pic/api/visualiza-salario/funcionario?token=" + url.QueryEscape(tok)
		body, status, err := sess.doGet(ctx, path, "visualiza-salario")
		if err != nil {
			return nil, err
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}
		parsed, _, tot, decErr := decodeSonnerObjectOrArray(body)
		if decErr != nil {
			return nil, fmt.Errorf("erro ao decodificar Sonner /visualiza-salario/funcionario: %w", decErr)
		}
		if len(parsed) == 0 {
			continue
		}
		if !p.OrdemSalarioDesc {
			rows = parsed
			total = tot
			break
		}
		if tot > total {
			total = tot
		}
		for _, r := range parsed {
			funcID := mapGetStr(r, "id", "funcionarioId", "idFuncionario")
			mat := mapGetStr(r, "matricula", "numMatricula", "codigo", "registro")
			nome := strings.ToUpper(strings.TrimSpace(mapGetStr(r, "nome", "nomeFuncionario", "servidor", "funcionario")))
			key := funcID + "|" + mat + "|" + nome
			if key != "||" {
				if _, dup := seenRows[key]; dup {
					continue
				}
				seenRows[key] = struct{}{}
			}
			rows = append(rows, r)
			if len(rows) >= limit {
				break
			}
		}
		if len(rows) >= limit {
			break
		}
	}

	if len(rows) == 0 {
		if lastStatus != 0 && lastStatus != http.StatusOK {
			return nil, fmt.Errorf("Sonner /visualiza-salario/funcionario retornou HTTP %d", lastStatus)
		}
		return &ServidoresResult{
			Municipio:  m,
			Fonte:      sonnerFonte,
			Plataforma: sonnerPlataforma,
			PortalURL:  sess.root,
			Servidores: []ServidorItem{},
			Disponivel: true,
			Mensagem:   "Nenhum servidor encontrado para os filtros consultados.",
		}, nil
	}

	capped := minInt(limit, len(rows))
	items := make([]ServidorItem, 0, capped)
	rawIDs := make([]string, 0, capped)

	for i := 0; i < capped; i++ {
		r := rows[i]
		cpfMasked := maskCPF("", mapGetStr(r, "cpf", "numCpf", "cpfCnpj", "documento"))
		funcID := mapGetStr(r, "id", "funcionarioId", "idFuncionario")
		stripSonnerSensitiveFields(r)

		items = append(items, ServidorItem{
			Nome:           mapGetStr(r, "nome", "nomeFuncionario", "servidor", "funcionario"),
			Matricula:      mapGetStr(r, "matricula", "numMatricula", "codigo", "registro"),
			CPFParcial:     cpfMasked,
			Cargo:          mapGetStr(r, "cargo", "descricaoCargo", "nomeCargo"),
			Funcao:         mapGetStr(r, "funcao", "descricaoFuncao"),
			Lotacao:        mapGetStr(r, "lotacao", "descricaoLotacao", "departamento", "setor", "unidade"),
			Orgao:          mapGetStr(r, "orgao", "nomeOrgao", "entidade"),
			Vinculo:        mapGetStr(r, "vinculo", "regime", "tipoVinculo", "categoria"),
			Situacao:       mapGetStr(r, "situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salarioBase", "vencimentoBase", "base"),
			SalarioBruto:   mapGetFloat(r, "salarioBruto", "totalProventos", "proventos", "bruto", "remuneracao", "vencimentos"),
			Descontos:      mapGetFloat(r, "descontos", "totalDescontos"),
			SalarioLiquido: mapGetFloat(r, "salarioLiquido", "liquido", "totalLiquido"),
			Competencia:    mapGetStr(r, "competencia"),
			CargaHoraria:   mapGetStr(r, "cargaHoraria", "horasSemanais"),
			DataAdmissao:   mapGetStr(r, "dataAdmissao", "admissao"),
			Fonte:          sonnerFonte,
			PortalURL:      sess.root,
		})
		rawIDs = append(rawIDs, funcID)
	}

	if !p.OrdemSalarioDesc {
		// Enriquecimento opcional para os primeiros N <= 10 registros sem remuneração
		enrichLimit := minInt(10, len(items))
		defaultAno := p.Ano
		for i := 0; i < enrichLimit; i++ {
			if items[i].SalarioBruto > 0 {
				continue
			}
			if ctx.Err() != nil {
				break
			}
			ano := defaultAno
			if ano <= 0 {
				ano = fetchSonnerAnoFolha(ctx, sess, rawIDs[i], items[i].Matricula)
				if ano <= 0 {
					ano = time.Now().Year()
				} else {
					defaultAno = ano
				}
			}
			enrichSonnerFolha(ctx, sess, &items[i], rawIDs[i], ano, p.Mes)
		}
	} else {
		enrichLimit := minInt(100, len(items))
		var idxs []int
		for i := 0; i < enrichLimit; i++ {
			if items[i].SalarioBruto <= 0 {
				idxs = append(idxs, i)
			}
		}
		if len(idxs) > 0 && ctx.Err() == nil {
			defaultAno := p.Ano
			if defaultAno <= 0 {
				defaultAno = fetchSonnerAnoFolha(ctx, sess, rawIDs[idxs[0]], items[idxs[0]].Matricula)
				if defaultAno <= 0 {
					defaultAno = time.Now().Year()
				}
			}
			workers := minInt(5, len(idxs))
			ch := make(chan int, len(idxs))
			for _, idx := range idxs {
				ch <- idx
			}
			close(ch)
			var wg sync.WaitGroup
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for idx := range ch {
						if ctx.Err() != nil {
							return
						}
						enrichSonnerFolha(ctx, sess, &items[idx], rawIDs[idx], defaultAno, p.Mes)
					}
				}()
			}
			wg.Wait()
		}
	}

	if total == 0 || (p.OrdemSalarioDesc && len(rows) > total) {
		total = len(rows)
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          sonnerFonte,
		Plataforma:     sonnerPlataforma,
		PortalURL:      sess.root,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
	}, nil
}

func fetchSonnerAnoFolha(ctx context.Context, sess *sonnerSession, funcID, matricula string) int {
	q := url.Values{}
	if funcID != "" {
		q.Set("funcionarioId", funcID)
	}
	if matricula != "" {
		q.Set("matricula", matricula)
	}
	path := "/GRP/pic/api/visualiza-salario/anos-folha"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	body, status, err := sess.doGet(ctx, path, "visualiza-salario")
	if err != nil || status != http.StatusOK {
		return 0
	}

	trimmed := strings.TrimSpace(string(body))
	var intArr []int
	if json.Unmarshal([]byte(trimmed), &intArr) == nil && len(intArr) > 0 {
		best := intArr[0]
		for _, y := range intArr {
			if y > best {
				best = y
			}
		}
		return best
	}

	rows, _, _, decErr := decodeSonnerObjectOrArray(body)
	if decErr != nil || len(rows) == 0 {
		return 0
	}
	best := 0
	for _, r := range rows {
		if y := mapGetInt(r, "ano", "exercicio", "value", "id"); y > best {
			best = y
		}
	}
	return best
}

func enrichSonnerFolha(ctx context.Context, sess *sonnerSession, item *ServidorItem, funcID string, ano, mes int) {
	q := url.Values{}
	q.Set("ano", strconv.Itoa(ano))
	if item.Matricula != "" {
		q.Set("matricula", item.Matricula)
	}
	if funcID != "" {
		q.Set("funcionarioId", funcID)
	}
	if mes > 0 {
		q.Set("mes", strconv.Itoa(mes))
	}
	path := "/GRP/pic/api/visualiza-salario/matriculafolhas?" + q.Encode()
	body, status, err := sess.doGet(ctx, path, "visualiza-salario")
	if err != nil || status != http.StatusOK {
		return
	}
	rows, obj, _, decErr := decodeSonnerObjectOrArray(body)
	if decErr != nil {
		return
	}
	var target map[string]any
	if len(rows) > 0 {
		target = rows[len(rows)-1]
		if mes > 0 {
			for _, r := range rows {
				if mapGetInt(r, "mes", "mesCompetencia") == mes {
					target = r
					break
				}
			}
		}
	} else if obj != nil {
		target = obj
	}
	if target == nil {
		return
	}

	// Nunca armazena CPF bruto, endereço ou dados bancários
	if item.CPFParcial == "" {
		item.CPFParcial = maskCPF("", mapGetStr(target, "cpf", "numCpf"))
	}
	stripSonnerSensitiveFields(target)

	if sb := mapGetFloat(target, "salarioBase", "vencimentoBase", "base"); sb > 0 {
		item.SalarioBase = sb
	}
	if bruto := mapGetFloat(target, "salarioBruto", "totalProventos", "proventos", "bruto", "valorBruto", "vencimentos"); bruto > 0 {
		item.SalarioBruto = bruto
	}
	if desc := mapGetFloat(target, "descontos", "totalDescontos", "valorDescontos"); desc > 0 {
		item.Descontos = desc
	}
	if liq := mapGetFloat(target, "salarioLiquido", "totalLiquido", "liquido", "valorLiquido"); liq > 0 {
		item.SalarioLiquido = liq
	} else if item.SalarioBruto > 0 && item.Descontos > 0 {
		item.SalarioLiquido = item.SalarioBruto - item.Descontos
	}
	if comp := mapGetStr(target, "competencia", "mesAno"); comp != "" {
		item.Competencia = comp
	} else {
		mFolha := mapGetInt(target, "mes", "mesCompetencia")
		if mFolha <= 0 {
			mFolha = mes
		}
		if mFolha > 0 {
			item.Competencia = fmt.Sprintf("%02d/%04d", mFolha, ano)
		} else {
			item.Competencia = strconv.Itoa(ano)
		}
	}
	if item.Cargo == "" {
		item.Cargo = mapGetStr(target, "cargo", "descricaoCargo")
	}
	if item.Lotacao == "" {
		item.Lotacao = mapGetStr(target, "lotacao", "descricaoLotacao", "departamento")
	}
}

// QuerySonnerDespesas consulta despesas em Sonner (/GRP/pic/api/dashboard/data/despesas ou /dashboard/drill/despesas).
func QuerySonnerDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	sess, err := newSonnerSession(ctx, client, p.BaseURL, p.TenantID)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	var lastStatus int
	for _, ano := range resolveSonnerYears(p) {
		payload := map[string]any{"ano": ano}
		body, status, err := sess.doPostJSON(ctx, "/GRP/pic/api/dashboard/data/despesas", "", payload)
		if err != nil {
			return nil, err
		}
		lastStatus = status

		var rows []map[string]any
		var obj map[string]any
		var total int
		if status == http.StatusOK {
			rows, obj, total, _ = decodeSonnerObjectOrArray(body)
		}

		// Se /dashboard/data/despesas falhou ou só retornou resumo sem itens, tenta /dashboard/drill/despesas
		if len(rows) == 0 {
			bodyDrill, statusDrill, errDrill := sess.doPostJSON(ctx, "/GRP/pic/api/dashboard/drill/despesas", "", payload)
			if errDrill == nil && statusDrill == http.StatusOK {
				drillRows, drillObj, drillTotal, _ := decodeSonnerObjectOrArray(bodyDrill)
				if len(drillRows) > 0 {
					rows = drillRows
					total = drillTotal
				} else if obj == nil && drillObj != nil {
					obj = drillObj
				}
				lastStatus = statusDrill
			} else if status != http.StatusOK && errDrill == nil {
				lastStatus = statusDrill
			}
		}

		if len(rows) > 0 {
			items := make([]DespesaItem, 0, minInt(limit, len(rows)))
			matchedTotal := 0
			for _, r := range rows {
				fornecedor := mapGetStr(r, "fornecedor", "favorecido", "credor", "nome", "descricao")
				historico := mapGetStr(r, "historico", "descricao", "especificacao", "objeto")
				if filtro != "" && !strings.Contains(strings.ToLower(fornecedor), filtro) && !strings.Contains(strings.ToLower(historico), filtro) {
					continue
				}
				matchedTotal++
				if len(items) >= limit {
					continue
				}
				itemAno := mapGetInt(r, "ano", "exercicio")
				if itemAno == 0 {
					itemAno = ano
				}
				doc := maskPortalTPDoc(mapGetStr(r, "documentoFornecedor", "cpfCnpj", "documento", "cnpj"))
				items = append(items, DespesaItem{
					NumeroEmpenho:       mapGetStr(r, "numeroEmpenho", "empenho", "numero"),
					Ano:                 itemAno,
					Data:                mapGetStr(r, "data", "dataEmpenho"),
					Fornecedor:          fornecedor,
					DocumentoFornecedor: doc,
					Orgao:               mapGetStr(r, "orgao", "nomeOrgao", "entidade"),
					Unidade:             mapGetStr(r, "unidade", "nomeUnidade"),
					Funcao:              mapGetStr(r, "funcao", "nomeFuncao"),
					Elemento:            mapGetStr(r, "elemento", "elementoDespesa", "natureza"),
					Historico:           historico,
					ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "empenhado", "valor"),
					ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "liquidado"),
					ValorPago:           mapGetFloat(r, "valorPago", "pago"),
					Fonte:               sonnerFonte,
				})
			}
			if len(items) == 0 && filtro != "" {
				continue
			}
			if filtro != "" || total == 0 {
				total = matchedTotal
			}
			return &DespesasResult{
				Municipio:      m,
				Fonte:          sonnerFonte,
				Plataforma:     sonnerPlataforma,
				PortalURL:      sess.root,
				TotalRegistros: total,
				Despesas:       items,
				Disponivel:     true,
			}, nil
		}

		// Retorno agregado (objeto único com totais de dashboard)
		if obj != nil {
			emp := mapGetFloat(obj, "empenhado", "valorEmpenhado", "totalEmpenhado")
			liq := mapGetFloat(obj, "liquidado", "valorLiquidado", "totalLiquidado")
			pag := mapGetFloat(obj, "pago", "valorPago", "totalPago")
			if emp != 0 || liq != 0 || pag != 0 {
				items := []DespesaItem{{
					Ano:            ano,
					Orgao:          mapGetStr(obj, "orgao", "entidade"),
					Historico:      fmt.Sprintf("Resumo anual de despesas (%d)", ano),
					ValorEmpenhado: emp,
					ValorLiquidado: liq,
					ValorPago:      pag,
					Fonte:          sonnerFonte,
				}}
				return &DespesasResult{
					Municipio:      m,
					Fonte:          sonnerFonte,
					Plataforma:     sonnerPlataforma,
					PortalURL:      sess.root,
					TotalRegistros: 1,
					Despesas:       items,
					Disponivel:     true,
				}, nil
			}
		}
	}

	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("Sonner /dashboard/data/despesas retornou HTTP %d", lastStatus)
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      sonnerFonte,
		Plataforma: sonnerPlataforma,
		PortalURL:  sess.root,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa encontrada para o período informado.",
	}, nil
}

// QuerySonnerReceitas consulta receitas em Sonner (/GRP/pic/api/dashboard/data/receitas).
func QuerySonnerReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	sess, err := newSonnerSession(ctx, client, p.BaseURL, p.TenantID)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	var lastStatus int
	for _, ano := range resolveSonnerYears(p) {
		body, status, err := sess.doPostJSON(ctx, "/GRP/pic/api/dashboard/data/receitas", "", map[string]any{"ano": ano})
		if err != nil {
			return nil, err
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}

		rows, obj, total, decErr := decodeSonnerObjectOrArray(body)
		if decErr != nil {
			return nil, fmt.Errorf("erro ao decodificar Sonner /dashboard/data/receitas: %w", decErr)
		}

		if len(rows) > 0 {
			items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
			matchedTotal := 0
			for _, r := range rows {
				desc := mapGetStr(r, "descricao", "especificacao", "nome", "receita", "categoria")
				if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
					continue
				}
				matchedTotal++
				if len(items) >= limit {
					continue
				}
				items = append(items, ReceitaItem{
					Codigo:          mapGetStr(r, "codigo", "conta", "classificacao"),
					Descricao:       desc,
					Categoria:       mapGetStr(r, "categoria", "fonte", "origem"),
					Data:            mapGetStr(r, "data", "competencia"),
					ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "previsto", "orcado", "valorOrcado"),
					ValorArrecadado: mapGetFloat(r, "valorArrecadado", "arrecadado", "realizado", "valorRealizado", "valor"),
					Fonte:           sonnerFonte,
				})
			}
			if len(items) == 0 && filtro != "" {
				continue
			}
			if filtro != "" || total == 0 {
				total = matchedTotal
			}
			return &ReceitasResult{
				Municipio:      m,
				Fonte:          sonnerFonte,
				Plataforma:     sonnerPlataforma,
				PortalURL:      sess.root,
				TotalRegistros: total,
				Receitas:       items,
				Disponivel:     true,
			}, nil
		}

		if obj != nil {
			prev := mapGetFloat(obj, "previsto", "valorPrevisto", "orcado", "valorOrcado", "totalPrevisto")
			arr := mapGetFloat(obj, "arrecadado", "valorArrecadado", "realizado", "valorRealizado", "totalArrecadado")
			if prev != 0 || arr != 0 {
				items := []ReceitaItem{{
					Descricao:       fmt.Sprintf("Resumo anual de receitas (%d)", ano),
					ValorPrevisto:   prev,
					ValorArrecadado: arr,
					Fonte:           sonnerFonte,
				}}
				return &ReceitasResult{
					Municipio:      m,
					Fonte:          sonnerFonte,
					Plataforma:     sonnerPlataforma,
					PortalURL:      sess.root,
					TotalRegistros: 1,
					Receitas:       items,
					Disponivel:     true,
				}, nil
			}
		}
	}

	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("Sonner /dashboard/data/receitas retornou HTTP %d", lastStatus)
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      sonnerFonte,
		Plataforma: sonnerPlataforma,
		PortalURL:  sess.root,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita encontrada para o período informado.",
	}, nil
}

// QuerySonnerLicitacoes consulta licitações em Sonner (/GRP/pic/api/consulta-licitacao).
func QuerySonnerLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	sess, err := newSonnerSession(ctx, client, p.BaseURL, p.TenantID)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	var lastStatus int
	for _, ano := range resolveSonnerYears(p) {
		body, status, err := sess.doPostJSON(ctx, "/GRP/pic/api/consulta-licitacao", "consulta-licitacao", map[string]any{"ano": ano})
		if err != nil {
			return nil, err
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}

		rows, _, total, decErr := decodeSonnerObjectOrArray(body)
		if decErr != nil {
			return nil, fmt.Errorf("erro ao decodificar Sonner /consulta-licitacao: %w", decErr)
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			objeto := mapGetStr(r, "objeto", "descricao")
			if filtro != "" && !strings.Contains(strings.ToLower(objeto), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			itemAno := mapGetInt(r, "ano", "anoLicitacao", "exercicio")
			if itemAno == 0 {
				itemAno = ano
			}
			link := mapGetStr(r, "link", "url")
			if link == "" {
				link = sess.root
			}
			items = append(items, LicitacaoItem{
				Numero:          mapGetStr(r, "numero", "numeroLicitacao", "numeroProcesso", "processo", "edital"),
				Ano:             itemAno,
				Modalidade:      mapGetStr(r, "modalidade", "descricaoModalidade", "tipo"),
				Situacao:        mapGetStr(r, "situacao", "status", "descricaoSituacao"),
				DataAbertura:    mapGetStr(r, "dataAbertura", "dataPublicacao", "data"),
				Objeto:          objeto,
				ValorEstimado:   mapGetFloat(r, "valorEstimado", "valorMaximo", "valorPrevisto", "valor"),
				ValorHomologado: mapGetFloat(r, "valorHomologado", "valorAdjudicado", "valorVencedor", "valorTotal"),
				Link:            link,
				Fonte:           sonnerFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &LicitacoesResult{
			Municipio:      m,
			Fonte:          sonnerFonte,
			Plataforma:     sonnerPlataforma,
			PortalURL:      sess.root,
			TotalRegistros: total,
			Licitacoes:     items,
			Disponivel:     true,
		}, nil
	}

	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("Sonner /consulta-licitacao retornou HTTP %d", lastStatus)
	}

	return &LicitacoesResult{
		Municipio:  m,
		Fonte:      sonnerFonte,
		Plataforma: sonnerPlataforma,
		PortalURL:  sess.root,
		Licitacoes: []LicitacaoItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma licitação encontrada para o período informado.",
	}, nil
}

// QuerySonnerContratos consulta contratos em Sonner (/GRP/pic/api/consulta-contrato).
func QuerySonnerContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	sess, err := newSonnerSession(ctx, client, p.BaseURL, p.TenantID)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	var lastStatus int
	for _, ano := range resolveSonnerYears(p) {
		body, status, err := sess.doPostJSON(ctx, "/GRP/pic/api/consulta-contrato", "consulta-contrato", map[string]any{"ano": ano})
		if err != nil {
			return nil, err
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}

		rows, _, total, decErr := decodeSonnerObjectOrArray(body)
		if decErr != nil {
			return nil, fmt.Errorf("erro ao decodificar Sonner /consulta-contrato: %w", decErr)
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]ContratoItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			fornecedor := mapGetStr(r, "fornecedor", "contratado", "nomeContratado", "favorecido", "credor", "razaoSocial")
			objeto := mapGetStr(r, "objeto", "descricao")
			if filtro != "" && !strings.Contains(strings.ToLower(fornecedor), filtro) && !strings.Contains(strings.ToLower(objeto), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			itemAno := mapGetInt(r, "ano", "anoContrato", "exercicio")
			if itemAno == 0 {
				itemAno = ano
			}
			doc := maskPortalTPDoc(mapGetStr(r, "documentoFornecedor", "cpfCnpj", "documento", "cnpj", "cpf"))

			items = append(items, ContratoItem{
				Numero:              mapGetStr(r, "numero", "numeroContrato", "contrato"),
				Ano:                 itemAno,
				Fornecedor:          fornecedor,
				DocumentoFornecedor: doc,
				Objeto:              objeto,
				Situacao:            mapGetStr(r, "situacao", "status"),
				DataAssinatura:      mapGetStr(r, "dataAssinatura", "data"),
				VigenciaInicio:      mapGetStr(r, "vigenciaInicio", "dataInicio", "inicioVigencia"),
				VigenciaFim:         mapGetStr(r, "vigenciaFim", "dataFim", "fimVigencia", "dataTermino"),
				Valor:               mapGetFloat(r, "valor", "valorContrato", "valorGlobal", "valorInicial"),
				Fonte:               sonnerFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &ContratosResult{
			Municipio:      m,
			Fonte:          sonnerFonte,
			Plataforma:     sonnerPlataforma,
			PortalURL:      sess.root,
			TotalRegistros: total,
			Contratos:      items,
			Disponivel:     true,
		}, nil
	}

	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("Sonner /consulta-contrato retornou HTTP %d", lastStatus)
	}

	return &ContratosResult{
		Municipio:  m,
		Fonte:      sonnerFonte,
		Plataforma: sonnerPlataforma,
		PortalURL:  sess.root,
		Contratos:  []ContratoItem{},
		Disponivel: true,
		Mensagem:   "Nenhum contrato encontrado para o período informado.",
	}, nil
}

// QuerySonnerObras consulta obras públicas em Sonner (/GRP/pic/api/listagem-obra/consultar).
func QuerySonnerObras(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ObrasResult, error) {
	sess, err := newSonnerSession(ctx, client, p.BaseURL, p.TenantID)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	var lastStatus int
	for _, ano := range resolveSonnerYears(p) {
		body, status, err := sess.doPostJSON(ctx, "/GRP/pic/api/listagem-obra/consultar", "listagem-obra", map[string]any{"ano": ano})
		if err != nil {
			return nil, err
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}

		rows, _, total, decErr := decodeSonnerObjectOrArray(body)
		if decErr != nil {
			return nil, fmt.Errorf("erro ao decodificar Sonner /listagem-obra/consultar: %w", decErr)
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]ObraItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			titulo := mapGetStr(r, "titulo", "nome", "descricao", "objeto")
			objeto := mapGetStr(r, "objeto", "descricao", "titulo")
			if filtro != "" && !strings.Contains(strings.ToLower(titulo), filtro) && !strings.Contains(strings.ToLower(objeto), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			items = append(items, ObraItem{
				Codigo:      mapGetStr(r, "codigo", "codigoObra", "id", "numero"),
				Titulo:      titulo,
				Objeto:      objeto,
				Orgao:       mapGetStr(r, "orgao", "secretaria", "unidade"),
				Endereco:    mapGetStr(r, "endereco", "localizacao", "local"),
				Bairro:      mapGetStr(r, "bairro"),
				DataInicial: mapGetStr(r, "dataInicial", "dataInicio"),
				DataFinal:   mapGetStr(r, "dataFinal", "dataFim", "dataTermino", "previsaoTermino"),
				Situacao:    mapGetStr(r, "situacao", "status"),
				Valor:       mapGetFloat(r, "valor", "valorObra", "valorContrato", "valorPrevisto"),
				Fonte:       sonnerFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &ObrasResult{
			Municipio:      m,
			Fonte:          sonnerFonte,
			Plataforma:     sonnerPlataforma,
			PortalURL:      sess.root,
			TotalRegistros: total,
			Obras:          items,
			Disponivel:     true,
		}, nil
	}

	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("Sonner /listagem-obra/consultar retornou HTTP %d", lastStatus)
	}

	return &ObrasResult{
		Municipio:  m,
		Fonte:      sonnerFonte,
		Plataforma: sonnerPlataforma,
		PortalURL:  sess.root,
		Obras:      []ObraItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma obra encontrada para o período consultado.",
	}, nil
}

// QuerySonnerConvenios consulta convênios em Sonner (/GRP/pic/api/cbp-convenios/consultar).
func QuerySonnerConvenios(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ConveniosResult, error) {
	sess, err := newSonnerSession(ctx, client, p.BaseURL, p.TenantID)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	var lastStatus int
	for _, ano := range resolveSonnerYears(p) {
		body, status, err := sess.doPostJSON(ctx, "/GRP/pic/api/cbp-convenios/consultar", "cbp-convenios", map[string]any{"ano": ano})
		if err != nil {
			return nil, err
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}

		rows, _, total, decErr := decodeSonnerObjectOrArray(body)
		if decErr != nil {
			return nil, fmt.Errorf("erro ao decodificar Sonner /cbp-convenios/consultar: %w", decErr)
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]ConvenioItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			desc := mapGetStr(r, "descricao", "objeto")
			concedente := mapGetStr(r, "concedente", "orgaoConcedente", "origem")
			if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) && !strings.Contains(strings.ToLower(concedente), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			items = append(items, ConvenioItem{
				Numero:        mapGetStr(r, "numero", "numeroConvenio", "codigo"),
				Concedente:    concedente,
				Convenente:    mapGetStr(r, "convenente", "orgaoConvenente", "entidade", "beneficiario"),
				Categoria:     mapGetStr(r, "categoria", "esfera"),
				Tipo:          mapGetStr(r, "tipo", "tipoConvenio"),
				Situacao:      mapGetStr(r, "situacao", "status"),
				Data:          mapGetStr(r, "data", "dataAssinatura", "dataCelebracao"),
				Vigencia:      mapGetStr(r, "vigencia", "dataFim", "vigenciaFim"),
				Descricao:     desc,
				Valor:         mapGetFloat(r, "valor", "valorTotal", "valorPactuado", "valorConvenio"),
				ValorLiberado: mapGetFloat(r, "valorLiberado", "valorRepassado", "valorRecebido", "valorPago"),
				Fonte:         sonnerFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &ConveniosResult{
			Municipio:      m,
			Fonte:          sonnerFonte,
			Plataforma:     sonnerPlataforma,
			PortalURL:      sess.root,
			TotalRegistros: total,
			Convenios:      items,
			Disponivel:     true,
		}, nil
	}

	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("Sonner /cbp-convenios/consultar retornou HTTP %d", lastStatus)
	}

	return &ConveniosResult{
		Municipio:  m,
		Fonte:      sonnerFonte,
		Plataforma: sonnerPlataforma,
		PortalURL:  sess.root,
		Convenios:  []ConvenioItem{},
		Disponivel: true,
		Mensagem:   "Nenhum convênio encontrado para o período consultado.",
	}, nil
}
