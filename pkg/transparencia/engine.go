package transparencia

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Engine é o motor de alta performance em Go para pesquisa e conexão de transparência
// de todos os municípios e estados brasileiros.
type Engine struct {
	mu             sync.RWMutex
	estados        []*Estado
	municipios     []*Municipio
	byIBGE         map[string]*Municipio
	bySIAFI        map[string]*Municipio
	byCNPJ         map[string]*Municipio
	byUF           map[string][]*Municipio
	estadosByUF    map[string]*Estado
	estadosByCode  map[int]*Estado
	muniNormIndex  []*indexedMuni
	megasoftList      []MegasoftMunicipio
	megasoftByIBGE    map[string]MegasoftMunicipio
	portalCatalogList []PortalCatalogMunicipio
	portalCatalogIBGE map[string]PortalCatalogMunicipio
	portalHTMLList    []PortalHTMLMunicipio
	portalHTMLIBGE    map[string]PortalHTMLMunicipio
}

type indexedMuni struct {
	m      *Municipio
	norm   string
	tokens []string
}

var (
	globalMegaMu           sync.RWMutex
	globalMegaByIBGE       = make(map[string]MegasoftMunicipio)
	globalPortalCatMu      sync.RWMutex
	globalPortalCatByIBGE  = make(map[string]PortalCatalogMunicipio)
	globalPortalHTMLMu     sync.RWMutex
	globalPortalHTMLByIBGE = make(map[string]PortalHTMLMunicipio)
)

// LookupMegasoftByIBGE consulta o catálogo nacional embutido de portais Megasoft pelo código IBGE.
func LookupMegasoftByIBGE(codIBGE string) (MegasoftMunicipio, bool) {
	globalMegaMu.RLock()
	defer globalMegaMu.RUnlock()
	m, ok := globalMegaByIBGE[strings.TrimSpace(codIBGE)]
	return m, ok
}

// LookupPortalCatalogByIBGE consulta o catálogo nacional embutido de portais municipais verificados pelo código IBGE.
func LookupPortalCatalogByIBGE(codIBGE string) (PortalCatalogMunicipio, bool) {
	globalPortalCatMu.RLock()
	defer globalPortalCatMu.RUnlock()
	m, ok := globalPortalCatByIBGE[strings.TrimSpace(codIBGE)]
	return m, ok
}

// LookupPortalHTMLByIBGE consulta o catálogo nacional embutido de portais municipais HTML (sem API estruturada) pelo código IBGE.
func LookupPortalHTMLByIBGE(codIBGE string) (PortalHTMLMunicipio, bool) {
	globalPortalHTMLMu.RLock()
	defer globalPortalHTMLMu.RUnlock()
	m, ok := globalPortalHTMLByIBGE[strings.TrimSpace(codIBGE)]
	return m, ok
}

// NewEngine inicializa o motor carregando os dados oficiais embutidos de UFs e municípios.
func NewEngine() (*Engine, error) {
	e := &Engine{
		estados:           make([]*Estado, 0, 27),
		municipios:        make([]*Municipio, 0, 5571),
		byIBGE:            make(map[string]*Municipio, 5571),
		bySIAFI:           make(map[string]*Municipio, 5571),
		byCNPJ:            make(map[string]*Municipio, 5571),
		byUF:              make(map[string][]*Municipio, 27),
		estadosByUF:       make(map[string]*Estado, 27),
		estadosByCode:     make(map[int]*Estado, 27),
		muniNormIndex:     make([]*indexedMuni, 0, 5571),
		megasoftList:      make([]MegasoftMunicipio, 0, 256),
		megasoftByIBGE:    make(map[string]MegasoftMunicipio, 256),
		portalCatalogList: make([]PortalCatalogMunicipio, 0, 2200),
		portalCatalogIBGE: make(map[string]PortalCatalogMunicipio, 2200),
		portalHTMLList:    make([]PortalHTMLMunicipio, 0, 2500),
		portalHTMLIBGE:    make(map[string]PortalHTMLMunicipio, 2500),
	}

	if err := e.loadEstados(); err != nil {
		return nil, fmt.Errorf("falha ao carregar UFs: %w", err)
	}

	if err := e.loadMunicipios(); err != nil {
		return nil, fmt.Errorf("falha ao carregar municípios: %w", err)
	}

	if err := e.loadMegasoftMunicipios(); err != nil {
		return nil, fmt.Errorf("falha ao carregar catálogo Megasoft: %w", err)
	}

	if err := e.loadPortaisMunicipais(); err != nil {
		return nil, fmt.Errorf("falha ao carregar catálogo nacional de portais: %w", err)
	}

	if err := e.loadPortaisHTMLMunicipais(); err != nil {
		return nil, fmt.Errorf("falha ao carregar catálogo de portais HTML: %w", err)
	}

	// Atualiza totais de municípios por estado
	for _, est := range e.estados {
		munis := e.byUF[est.UF]
		est.TotalMunicipios = len(munis)
		if est.Capital == "" {
			if info, ok := ufsInfo[est.UF]; ok {
				est.Capital = info.CapitalNome
			}
		}
	}

	return e, nil
}

func (e *Engine) loadEstados() error {
	r := csv.NewReader(bytes.NewReader(estadosCSV))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return err
	}
	_ = header // ignora cabeçalho

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(rec) < 6 {
			continue
		}

		// codigo_uf,uf,nome,latitude,longitude,regiao
		codeUF, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		uf := strings.ToUpper(strings.TrimSpace(rec[1]))
		nome := strings.TrimSpace(rec[2])
		lat, _ := strconv.ParseFloat(strings.TrimSpace(rec[3]), 64)
		lon, _ := strconv.ParseFloat(strings.TrimSpace(rec[4]), 64)
		regiao := strings.TrimSpace(rec[5])

		info := ufsInfo[uf]
		radarURL := "https://radardatransparencia.atricon.org.br/radar-da-transparencia-publica.html"
		pncpURL := fmt.Sprintf("https://pncp.gov.br/app/editais?uf=%s", uf)

		est := &Estado{
			CodigoUF:             codeUF,
			UF:                   uf,
			Nome:                 nome,
			Latitude:             lat,
			Longitude:            lon,
			Regiao:               regiao,
			Capital:              info.CapitalNome,
			PortalTransparencia:  info.TransparenciaGov,
			PortalGoverno:        info.PortalEstado,
			TribunalContasNome:   info.NomeEstado,
			TribunalContasURL:    info.URLEstado,
			AssembleiaNome:       info.AssembleiaNome,
			AssembleiaURL:        info.AssembleiaURL,
			SiconfiEnteID:        strconv.Itoa(codeUF),
			RadarTransparenciaURL: radarURL,
			PNCPURL:              pncpURL,
		}

		e.estados = append(e.estados, est)
		e.estadosByUF[uf] = est
		e.estadosByCode[codeUF] = est
	}

	// Ordena estados por nome
	sort.Slice(e.estados, func(i, j int) bool {
		return e.estados[i].Nome < e.estados[j].Nome
	})

	return nil
}

func (e *Engine) loadMunicipios() error {
	r := csv.NewReader(bytes.NewReader(municipiosCSV))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return err
	}
	_ = header // ignora cabeçalho

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(rec) < 9 {
			continue
		}

		// codigo_ibge,nome,latitude,longitude,capital,codigo_uf,siafi_id,ddd,fuso_horario,cnpj,populacao
		codIBGE := strings.TrimSpace(rec[0])
		nome := strings.TrimSpace(rec[1])
		lat, _ := strconv.ParseFloat(strings.TrimSpace(rec[2]), 64)
		lon, _ := strconv.ParseFloat(strings.TrimSpace(rec[3]), 64)
		capital := strings.TrimSpace(rec[4]) == "1"
		codeUF, _ := strconv.Atoi(strings.TrimSpace(rec[5]))
		siafi := strings.TrimSpace(rec[6])
		ddd := strings.TrimSpace(rec[7])
		fuso := strings.TrimSpace(rec[8])

		cnpj := ""
		populacao := 0
		if len(rec) >= 10 {
			cnpj = cleanDigits(strings.TrimSpace(rec[9]))
		}
		if len(rec) >= 11 {
			populacao, _ = strconv.Atoi(strings.TrimSpace(rec[10]))
		}

		est := e.estadosByCode[codeUF]
		ufSigla := ""
		regiao := ""
		if est != nil {
			ufSigla = est.UF
			regiao = est.Regiao
		}

		slug := Slugify(nome)

		m := &Municipio{
			CodigoIBGE:  codIBGE,
			Nome:        nome,
			Latitude:    lat,
			Longitude:   lon,
			Capital:     capital,
			CodigoUF:    codeUF,
			UF:          ufSigla,
			Regiao:      regiao,
			SiafiID:     siafi,
			DDD:         ddd,
			FusoHorario: fuso,
			Slug:        slug,
			CNPJ:        cnpj,
			Populacao:   populacao,
		}

		e.municipios = append(e.municipios, m)
		e.byIBGE[codIBGE] = m
		if siafi != "" {
			e.bySIAFI[siafi] = m
		}
		if cnpj != "" {
			e.byCNPJ[cnpj] = m
		}
		if ufSigla != "" {
			e.byUF[ufSigla] = append(e.byUF[ufSigla], m)
		}

		norm := RemoveAccents(nome)
		tokens := Tokenize(nome)
		e.muniNormIndex = append(e.muniNormIndex, &indexedMuni{
			m:      m,
			norm:   norm,
			tokens: tokens,
		})
	}

	return nil
}

func (e *Engine) loadMegasoftMunicipios() error {
	if len(megasoftMunicipiosCSV) == 0 {
		return nil
	}
	r := csv.NewReader(bytes.NewReader(megasoftMunicipiosCSV))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	_ = header

	globalMegaMu.Lock()
	defer globalMegaMu.Unlock()

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(rec) < 4 {
			continue
		}
		codOrgao := 0
		if len(rec) >= 6 {
			codOrgao, _ = strconv.Atoi(strings.TrimSpace(rec[5]))
		}
		sitePref := ""
		if len(rec) >= 7 {
			sitePref = strings.TrimSpace(rec[6])
		}
		portalCam := ""
		if len(rec) >= 5 {
			portalCam = strings.TrimSpace(rec[4])
		}
		item := MegasoftMunicipio{
			CodigoIBGE:     strings.TrimSpace(rec[0]),
			Nome:           strings.TrimSpace(rec[1]),
			UF:             strings.ToUpper(strings.TrimSpace(rec[2])),
			PortalURL:      strings.TrimSpace(rec[3]),
			PortalCamara:   portalCam,
			CodigoOrgao:    codOrgao,
			SitePrefeitura: sitePref,
		}
		if item.CodigoIBGE == "" || item.PortalURL == "" {
			continue
		}
		e.megasoftList = append(e.megasoftList, item)
		e.megasoftByIBGE[item.CodigoIBGE] = item
		globalMegaByIBGE[item.CodigoIBGE] = item
	}

	sort.Slice(e.megasoftList, func(i, j int) bool {
		if e.megasoftList[i].UF != e.megasoftList[j].UF {
			return e.megasoftList[i].UF < e.megasoftList[j].UF
		}
		return e.megasoftList[i].Nome < e.megasoftList[j].Nome
	})
	return nil
}

// ListMegasoftMunicipios retorna todos os municípios com portais Megasoft verificados (com filtro opcional por UF).
func (e *Engine) ListMegasoftMunicipios(uf string) []MegasoftMunicipio {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ufUp := strings.ToUpper(strings.TrimSpace(uf))
	if ufUp == "" {
		res := make([]MegasoftMunicipio, len(e.megasoftList))
		copy(res, e.megasoftList)
		return res
	}
	res := make([]MegasoftMunicipio, 0)
	for _, it := range e.megasoftList {
		if it.UF == ufUp {
			res = append(res, it)
		}
	}
	return res
}

// GetMegasoftMunicipio busca um portal Megasoft verificado pelo código IBGE.
func (e *Engine) GetMegasoftMunicipio(codigoIBGE string) (MegasoftMunicipio, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	item, ok := e.megasoftByIBGE[strings.TrimSpace(codigoIBGE)]
	return item, ok
}

func (e *Engine) loadPortaisMunicipais() error {
	if len(portaisMunicipaisCSV) == 0 {
		return nil
	}
	r := csv.NewReader(bytes.NewReader(portaisMunicipaisCSV))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	_ = header

	globalPortalCatMu.Lock()
	defer globalPortalCatMu.Unlock()

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(rec) < 5 {
			continue
		}
		tenantID := ""
		if len(rec) >= 6 {
			tenantID = strings.TrimSpace(rec[5])
		}
		entNome := ""
		if len(rec) >= 7 {
			entNome = strings.TrimSpace(rec[6])
		}
		item := PortalCatalogMunicipio{
			CodigoIBGE:   strings.TrimSpace(rec[0]),
			Nome:         strings.TrimSpace(rec[1]),
			UF:           strings.ToUpper(strings.TrimSpace(rec[2])),
			Plataforma:   strings.TrimSpace(rec[3]),
			PortalURL:    strings.TrimSpace(rec[4]),
			TenantID:     tenantID,
			EntidadeNome: entNome,
		}
		if item.CodigoIBGE == "" || item.Plataforma == "" || item.PortalURL == "" {
			continue
		}
		e.portalCatalogList = append(e.portalCatalogList, item)
		e.portalCatalogIBGE[item.CodigoIBGE] = item
		globalPortalCatByIBGE[item.CodigoIBGE] = item
	}

	sort.Slice(e.portalCatalogList, func(i, j int) bool {
		if e.portalCatalogList[i].UF != e.portalCatalogList[j].UF {
			return e.portalCatalogList[i].UF < e.portalCatalogList[j].UF
		}
		return e.portalCatalogList[i].Nome < e.portalCatalogList[j].Nome
	})
	return nil
}

// ListPortaisCatalogMunicipios retorna os municípios do catálogo nacional verificado (filtro opcional por UF).
func (e *Engine) ListPortaisCatalogMunicipios(uf string) []PortalCatalogMunicipio {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ufUp := strings.ToUpper(strings.TrimSpace(uf))
	if ufUp == "" {
		res := make([]PortalCatalogMunicipio, len(e.portalCatalogList))
		copy(res, e.portalCatalogList)
		return res
	}
	res := make([]PortalCatalogMunicipio, 0)
	for _, it := range e.portalCatalogList {
		if it.UF == ufUp {
			res = append(res, it)
		}
	}
	return res
}

// GetPortalCatalogMunicipio busca um portal verificado no catálogo nacional pelo código IBGE.
func (e *Engine) GetPortalCatalogMunicipio(codigoIBGE string) (PortalCatalogMunicipio, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	item, ok := e.portalCatalogIBGE[strings.TrimSpace(codigoIBGE)]
	return item, ok
}

func (e *Engine) loadPortaisHTMLMunicipais() error {
	if len(portaisHTMLMunicipaisCSV) == 0 {
		return nil
	}
	r := csv.NewReader(bytes.NewReader(portaisHTMLMunicipaisCSV))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	_ = header

	globalPortalHTMLMu.Lock()
	defer globalPortalHTMLMu.Unlock()

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(rec) < 5 {
			continue
		}
		sitePref := ""
		if len(rec) >= 6 {
			sitePref = strings.TrimSpace(rec[5])
		}
		obs := ""
		if len(rec) >= 7 {
			obs = strings.TrimSpace(rec[6])
		}
		item := PortalHTMLMunicipio{
			CodigoIBGE:     strings.TrimSpace(rec[0]),
			Nome:           strings.TrimSpace(rec[1]),
			UF:             strings.ToUpper(strings.TrimSpace(rec[2])),
			Plataforma:     strings.TrimSpace(rec[3]),
			PortalURL:      strings.TrimSpace(rec[4]),
			SitePrefeitura: sitePref,
			Observacoes:    obs,
		}
		if item.CodigoIBGE == "" || item.Plataforma == "" || item.PortalURL == "" {
			continue
		}
		e.portalHTMLList = append(e.portalHTMLList, item)
		e.portalHTMLIBGE[item.CodigoIBGE] = item
		globalPortalHTMLByIBGE[item.CodigoIBGE] = item
	}

	sort.Slice(e.portalHTMLList, func(i, j int) bool {
		if e.portalHTMLList[i].UF != e.portalHTMLList[j].UF {
			return e.portalHTMLList[i].UF < e.portalHTMLList[j].UF
		}
		return e.portalHTMLList[i].Nome < e.portalHTMLList[j].Nome
	})
	return nil
}

// ListPortaisHTMLMunicipios retorna os municípios do catálogo de portais HTML sem API estruturada (filtro opcional por UF).
func (e *Engine) ListPortaisHTMLMunicipios(uf string) []PortalHTMLMunicipio {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ufUp := strings.ToUpper(strings.TrimSpace(uf))
	if ufUp == "" {
		res := make([]PortalHTMLMunicipio, len(e.portalHTMLList))
		copy(res, e.portalHTMLList)
		return res
	}
	res := make([]PortalHTMLMunicipio, 0)
	for _, it := range e.portalHTMLList {
		if it.UF == ufUp {
			res = append(res, it)
		}
	}
	return res
}

// GetPortalHTMLMunicipio busca um portal HTML municipal catalogado pelo código IBGE.
func (e *Engine) GetPortalHTMLMunicipio(codigoIBGE string) (PortalHTMLMunicipio, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	item, ok := e.portalHTMLIBGE[strings.TrimSpace(codigoIBGE)]
	return item, ok
}

// ListEstados retorna todas as 27 Unidades Federativas.
func (e *Engine) ListEstados() []*Estado {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]*Estado, len(e.estados))
	copy(res, e.estados)
	return res
}

// GetEstado busca uma UF por sigla ("SP", "RJ") ou código IBGE ("35", "33").
func (e *Engine) GetEstado(ufOrCode string) (*Estado, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	query := strings.ToUpper(strings.TrimSpace(ufOrCode))
	if est, ok := e.estadosByUF[query]; ok {
		return est, nil
	}
	if code, err := strconv.Atoi(query); err == nil {
		if est, ok := e.estadosByCode[code]; ok {
			return est, nil
		}
	}
	// Busca por nome
	qNorm := RemoveAccents(query)
	for _, est := range e.estados {
		if RemoveAccents(est.Nome) == qNorm {
			return est, nil
		}
	}

	return nil, fmt.Errorf("estado/UF não encontrado: %s", ufOrCode)
}

// GetMunicipioByIBGE busca um município pelo código IBGE de 7 dígitos.
func (e *Engine) GetMunicipioByIBGE(code string) (*Municipio, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	c := strings.TrimSpace(code)
	if m, ok := e.byIBGE[c]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("município não encontrado para o código IBGE: %s", code)
}

// GetMunicipioBySIAFI busca um município pelo código SIAFI de 4 dígitos.
func (e *Engine) GetMunicipioBySIAFI(code string) (*Municipio, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	c := strings.TrimSpace(code)
	if m, ok := e.bySIAFI[c]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("município não encontrado para o código SIAFI: %s", code)
}

// cleanDigits remove todos os caracteres não numéricos de uma string.
func cleanDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// GetMunicipioByCNPJ busca um município pelo CNPJ oficial da Prefeitura (14 dígitos ou formatado).
func (e *Engine) GetMunicipioByCNPJ(cnpj string) (*Municipio, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	clean := cleanDigits(cnpj)
	if m, ok := e.byCNPJ[clean]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("município não encontrado para o CNPJ: %s", cnpj)
}

// SearchMunicipios busca municípios nativamente por nome, código IBGE, código SIAFI com filtro opcional por UF.
// Altamente otimizado para latência sub-milissegundo (< 0.1ms).
func (e *Engine) SearchMunicipios(query string, uf string, limit int) []*Municipio {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}

	qTrim := strings.TrimSpace(query)
	ufTrim := strings.ToUpper(strings.TrimSpace(uf))

	// 1. Busca direta por CNPJ (14 dígitos limpos)
	cleanQ := cleanDigits(qTrim)
	if len(cleanQ) == 14 {
		if m, ok := e.byCNPJ[cleanQ]; ok {
			if ufTrim == "" || m.UF == ufTrim {
				return []*Municipio{m}
			}
		}
	}

	// 2. Busca direta por código IBGE (se a query for numérica de até 7 dígitos)
	if _, err := strconv.Atoi(qTrim); err == nil && len(qTrim) == 7 {
		if m, ok := e.byIBGE[qTrim]; ok {
			if ufTrim == "" || m.UF == ufTrim {
				return []*Municipio{m}
			}
		}
	}

	// 3. Busca direta por código SIAFI (se numérica de 4 dígitos)
	if len(qTrim) == 4 {
		if m, ok := e.bySIAFI[qTrim]; ok {
			if ufTrim == "" || m.UF == ufTrim {
				return []*Municipio{m}
			}
		}
	}

	// Se query vazia e UF especificada, retorna municípios daquela UF
	if qTrim == "" && ufTrim != "" {
		munis := e.byUF[ufTrim]
		if len(munis) > limit {
			return munis[:limit]
		}
		return munis
	}

	qNorm := RemoveAccents(qTrim)
	qTokens := Tokenize(qTrim)

	type scored struct {
		m     *Municipio
		score int
	}

	matches := make([]scored, 0, 100)

	for _, item := range e.muniNormIndex {
		if ufTrim != "" && item.m.UF != ufTrim {
			continue
		}

		score := 0
		// Match exato
		if item.norm == qNorm {
			score += 1000
		} else if strings.HasPrefix(item.norm, qNorm) {
			score += 500
		} else if strings.Contains(item.norm, qNorm) {
			score += 200
		} else if len(qTokens) > 0 {
			// Multi-token match: todos os tokens da busca devem estar presentes
			allMatched := true
			for _, qt := range qTokens {
				tokenMatched := false
				for _, mt := range item.tokens {
					if strings.HasPrefix(mt, qt) {
						tokenMatched = true
						break
					}
				}
				if !tokenMatched {
					allMatched = false
					break
				}
			}
			if allMatched {
				score += 100
			}
		}

		// Match em código IBGE
		if strings.HasPrefix(item.m.CodigoIBGE, qTrim) {
			score += 300
		}

		// Match em CNPJ
		if cleanQ != "" && len(cleanQ) >= 4 && strings.HasPrefix(item.m.CNPJ, cleanQ) {
			score += 350
		}

		if score > 0 {
			if item.m.Capital {
				score += 50 // Capitais têm leve prioridade em empates
			}
			matches = append(matches, scored{m: item.m, score: score})
		}
	}

	// Ordena por score decrescente, e por nome alfabético em caso de empate
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].m.Nome < matches[j].m.Nome
	})

	count := len(matches)
	if count > limit {
		count = limit
	}

	res := make([]*Municipio, count)
	for i := 0; i < count; i++ {
		res[i] = matches[i].m
	}
	return res
}

// ConnectMunicipio conecta a um município brasileiro e retorna o dossiê completo de transparência.
func (e *Engine) ConnectMunicipio(queryOrCode string, uf string) (*MunicipioDossier, error) {
	munis := e.SearchMunicipios(queryOrCode, uf, 1)
	if len(munis) == 0 {
		return nil, fmt.Errorf("nenhum município encontrado para a consulta: '%s' (UF: %s)", queryOrCode, uf)
	}

	m := munis[0]
	est, _ := e.GetEstado(m.UF)
	return BuildMunicipioDossier(m, est), nil
}

// ConnectEstado conecta a um estado brasileiro (UF) e retorna o dossiê completo de transparência.
func (e *Engine) ConnectEstado(ufOrCode string) (*EstadoDossier, error) {
	est, err := e.GetEstado(ufOrCode)
	if err != nil {
		return nil, err
	}

	munis := e.byUF[est.UF]
	principais := make([]*Municipio, 0, 5)
	for _, m := range munis {
		if m.Capital || len(principais) < 5 {
			principais = append(principais, m)
		}
		if len(principais) >= 10 {
			break
		}
	}

	return BuildEstadoDossier(est, len(munis), principais), nil
}

// GetStats retorna métricas consolidadas sobre os 5.570 municípios e 27 estados.
func (e *Engine) GetStats() *Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	porRegiao := make(map[string]int)
	porUF := make(map[string]int)
	capitais := make([]string, 0, 27)
	var popTotal int64

	for _, m := range e.municipios {
		porRegiao[m.Regiao]++
		porUF[m.UF]++
		popTotal += int64(m.Populacao)
		if m.Capital {
			capitais = append(capitais, fmt.Sprintf("%s (%s)", m.Nome, m.UF))
		}
	}

	sort.Strings(capitais)

	return &Stats{
		TotalMunicipios:     len(e.municipios),
		TotalUFs:            len(e.estados),
		PopulacaoTotal:      popTotal,
		MunicipiosPorRegiao: porRegiao,
		MunicipiosPorUF:     porUF,
		Capitais:            capitais,
	}
}

// CompareTransparencia compara a estrutura de transparência entre múltiplos municípios ou estados.
func (e *Engine) CompareTransparencia(alvos []string) (*ComparativoRelatorio, error) {
	rel := &ComparativoRelatorio{
		Itens: make([]*ComparativoItem, 0, len(alvos)),
	}

	for _, alvo := range alvos {
		alvoTrim := strings.TrimSpace(alvo)
		if alvoTrim == "" {
			continue
		}

		// Tenta buscar como estado primeiro (se tiver 2 letras)
		if len(alvoTrim) == 2 {
			if est, err := e.GetEstado(alvoTrim); err == nil {
				rel.Itens = append(rel.Itens, &ComparativoItem{
					Tipo:                 "estado",
					Nome:                 est.Nome,
					UF:                   est.UF,
					CodigoIBGE:           strconv.Itoa(est.CodigoUF),
					Regiao:               est.Regiao,
					Capital:              false,
					PortalTransparencia:  est.PortalTransparencia,
					TribunalContasNome:   est.TribunalContasNome,
					TribunalContasURL:    est.TribunalContasURL,
					RadarTransparenciaURL: est.RadarTransparenciaURL,
					SiconfiURL:           fmt.Sprintf("https://siconfi.tesouro.gov.br/siconfi/index.jsf?cod_ibge=%d", est.CodigoUF),
					PNCPURL:              est.PNCPURL,
				})
				continue
			}
		}

		// Busca como município
		munis := e.SearchMunicipios(alvoTrim, "", 1)
		if len(munis) > 0 {
			m := munis[0]
			est, _ := e.GetEstado(m.UF)
			dos := BuildMunicipioDossier(m, est)
			rel.Itens = append(rel.Itens, &ComparativoItem{
				Tipo:                 "municipio",
				Nome:                 m.Nome,
				UF:                   m.UF,
				CodigoIBGE:           m.CodigoIBGE,
				SiafiID:              m.SiafiID,
				CNPJ:                 m.CNPJ,
				Populacao:            m.Populacao,
				Regiao:               m.Regiao,
				Capital:              m.Capital,
				PortalTransparencia:  dos.PortalTransparenciaPrefeitura,
				TribunalContasNome:   dos.TribunalContasCompetente,
				TribunalContasURL:    dos.TribunalContasURL,
				RadarTransparenciaURL: dos.RadarTransparenciaURL,
				SiconfiURL:           dos.SiconfiURL,
				PNCPURL:              dos.PNCPURL,
			})
		}
	}

	rel.TotalItens = len(rel.Itens)
	return rel, nil
}
