package transparencia

import (
	"fmt"
	"strings"
)

// InfoTribunal guarda dados do Tribunal de Contas responsável pelo estado e seus municípios.
type InfoTribunal struct {
	NomeEstado       string
	URLEstado        string
	NomeMunicipios   string
	URLMunicipios    string
	AssembleiaNome   string
	AssembleiaURL    string
	PortalEstado     string
	TransparenciaGov string
	CapitalNome      string
}

// Mapeamento oficial dos 27 estados (UFs) e seus órgãos fiscalizadores e portais de transparência.
var ufsInfo = map[string]InfoTribunal{
	"AC": {
		NomeEstado:       "Tribunal de Contas do Estado do Acre (TCE-AC)",
		URLEstado:        "https://www.tceac.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Acre (TCE-AC)",
		URLMunicipios:    "https://www.tceac.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado do Acre (ALEAC)",
		AssembleiaURL:    "https://www.al.ac.leg.br",
		PortalEstado:     "https://www.ac.gov.br",
		TransparenciaGov: "https://transparencia.ac.gov.br",
		CapitalNome:      "Rio Branco",
	},
	"AL": {
		NomeEstado:       "Tribunal de Contas do Estado de Alagoas (TCE-AL)",
		URLEstado:        "https://tceal.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Alagoas (TCE-AL)",
		URLMunicipios:    "https://tceal.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado de Alagoas (ALEAL)",
		AssembleiaURL:    "https://www.al.al.leg.br",
		PortalEstado:     "https://alagoas.al.gov.br",
		TransparenciaGov: "https://transparencia.al.gov.br",
		CapitalNome:      "Maceió",
	},
	"AP": {
		NomeEstado:       "Tribunal de Contas do Estado do Amapá (TCE-AP)",
		URLEstado:        "https://tceap.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Amapá (TCE-AP)",
		URLMunicipios:    "https://tceap.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado do Amapá (ALAP)",
		AssembleiaURL:    "https://www.al.ap.leg.br",
		PortalEstado:     "https://www.amapa.gov.br",
		TransparenciaGov: "https://transparencia.ap.gov.br",
		CapitalNome:      "Macapá",
	},
	"AM": {
		NomeEstado:       "Tribunal de Contas do Estado do Amazonas (TCE-AM)",
		URLEstado:        "https://tceam.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Amazonas (TCE-AM)",
		URLMunicipios:    "https://tceam.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado do Amazonas (ALEAM)",
		AssembleiaURL:    "https://www.aleam.gov.br",
		PortalEstado:     "https://www.amazonas.am.gov.br",
		TransparenciaGov: "https://transparencia.am.gov.br",
		CapitalNome:      "Manaus",
	},
	"BA": {
		NomeEstado:       "Tribunal de Contas do Estado da Bahia (TCE-BA)",
		URLEstado:        "https://www.tce.ba.gov.br",
		NomeMunicipios:   "Tribunal de Contas dos Municípios do Estado da Bahia (TCM-BA)",
		URLMunicipios:    "https://www.tcm.ba.gov.br",
		AssembleiaNome:   "Assembleia Legislativa da Bahia (ALBA)",
		AssembleiaURL:    "https://www.al.ba.leg.br",
		PortalEstado:     "https://www.ba.gov.br",
		TransparenciaGov: "https://www.transparencia.ba.gov.br",
		CapitalNome:      "Salvador",
	},
	"CE": {
		NomeEstado:       "Tribunal de Contas do Estado do Ceará (TCE-CE)",
		URLEstado:        "https://www.tce.ce.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Ceará (TCE-CE)",
		URLMunicipios:    "https://www.tce.ce.gov.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado do Ceará (ALECE)",
		AssembleiaURL:    "https://www.al.ce.leg.br",
		PortalEstado:     "https://www.ceara.gov.br",
		TransparenciaGov: "https://transparencia.ce.gov.br",
		CapitalNome:      "Fortaleza",
	},
	"DF": {
		NomeEstado:       "Tribunal de Contas do Distrito Federal (TCDF)",
		URLEstado:        "https://www.tc.df.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Distrito Federal (TCDF)",
		URLMunicipios:    "https://www.tc.df.gov.br",
		AssembleiaNome:   "Câmara Legislativa do Distrito Federal (CLDF)",
		AssembleiaURL:    "https://www.cl.df.gov.br",
		PortalEstado:     "https://www.df.gov.br",
		TransparenciaGov: "https://transparencia.df.gov.br",
		CapitalNome:      "Brasília",
	},
	"ES": {
		NomeEstado:       "Tribunal de Contas do Estado do Espírito Santo (TCE-ES)",
		URLEstado:        "https://www.tcees.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Espírito Santo (TCE-ES)",
		URLMunicipios:    "https://cidades.tcees.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Espírito Santo (ALES)",
		AssembleiaURL:    "https://www.al.es.leg.br",
		PortalEstado:     "https://www.es.gov.br",
		TransparenciaGov: "https://transparencia.es.gov.br",
		CapitalNome:      "Vitória",
	},
	"GO": {
		NomeEstado:       "Tribunal de Contas do Estado de Goiás (TCE-GO)",
		URLEstado:        "https://www.tce.go.gov.br",
		NomeMunicipios:   "Tribunal de Contas dos Municípios do Estado de Goiás (TCM-GO)",
		URLMunicipios:    "https://www.tcmgo.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado de Goiás (ALEGO)",
		AssembleiaURL:    "https://portal.al.go.leg.br",
		PortalEstado:     "https://www.goias.gov.br",
		TransparenciaGov: "https://transparencia.go.gov.br",
		CapitalNome:      "Goiânia",
	},
	"MA": {
		NomeEstado:       "Tribunal de Contas do Estado do Maranhão (TCE-MA)",
		URLEstado:        "https://www.tcema.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Maranhão (TCE-MA)",
		URLMunicipios:    "https://www.tcema.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Maranhão (ALEMA)",
		AssembleiaURL:    "https://www.al.ma.leg.br",
		PortalEstado:     "https://www.ma.gov.br",
		TransparenciaGov: "https://transparencia.ma.gov.br",
		CapitalNome:      "São Luís",
	},
	"MT": {
		NomeEstado:       "Tribunal de Contas do Estado de Mato Grosso (TCE-MT)",
		URLEstado:        "https://www.tce.mt.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Mato Grosso (TCE-MT)",
		URLMunicipios:    "https://www.tce.mt.gov.br",
		AssembleiaNome:   "Assembleia Legislativa de Mato Grosso (ALMT)",
		AssembleiaURL:    "https://www.al.mt.leg.br",
		PortalEstado:     "https://www.mt.gov.br",
		TransparenciaGov: "https://transparencia.mt.gov.br",
		CapitalNome:      "Cuiabá",
	},
	"MS": {
		NomeEstado:       "Tribunal de Contas do Estado de Mato Grosso do Sul (TCE-MS)",
		URLEstado:        "https://www.tce.ms.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Mato Grosso do Sul (TCE-MS)",
		URLMunicipios:    "https://www.tce.ms.gov.br",
		AssembleiaNome:   "Assembleia Legislativa de Mato Grosso do Sul (ALEMS)",
		AssembleiaURL:    "https://al.ms.leg.br",
		PortalEstado:     "https://www.ms.gov.br",
		TransparenciaGov: "https://transparencia.ms.gov.br",
		CapitalNome:      "Campo Grande",
	},
	"MG": {
		NomeEstado:       "Tribunal de Contas do Estado de Minas Gerais (TCE-MG)",
		URLEstado:        "https://www.tce.mg.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Minas Gerais (TCE-MG)",
		URLMunicipios:    "https://www.tce.mg.gov.br",
		AssembleiaNome:   "Assembleia Legislativa de Minas Gerais (ALMG)",
		AssembleiaURL:    "https://www.almg.gov.br",
		PortalEstado:     "https://www.mg.gov.br",
		TransparenciaGov: "https://transparencia.mg.gov.br",
		CapitalNome:      "Belo Horizonte",
	},
	"PA": {
		NomeEstado:       "Tribunal de Contas do Estado do Pará (TCE-PA)",
		URLEstado:        "https://www.tcepa.tc.br",
		NomeMunicipios:   "Tribunal de Contas dos Municípios do Estado do Pará (TCM-PA)",
		URLMunicipios:    "https://www.tcmpa.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Pará (ALEPA)",
		AssembleiaURL:    "https://www.alepa.pa.gov.br",
		PortalEstado:     "https://www.pa.gov.br",
		TransparenciaGov: "https://transparencia.pa.gov.br",
		CapitalNome:      "Belém",
	},
	"PB": {
		NomeEstado:       "Tribunal de Contas do Estado da Paraíba (TCE-PB)",
		URLEstado:        "https://tce.pb.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado da Paraíba (TCE-PB - Sagres)",
		URLMunicipios:    "https://sagres.tce.pb.gov.br",
		AssembleiaNome:   "Assembleia Legislativa da Paraíba (ALPB)",
		AssembleiaURL:    "https://www.al.pb.leg.br",
		PortalEstado:     "https://paraiba.pb.gov.br",
		TransparenciaGov: "https://transparencia.pb.gov.br",
		CapitalNome:      "João Pessoa",
	},
	"PR": {
		NomeEstado:       "Tribunal de Contas do Estado do Paraná (TCE-PR)",
		URLEstado:        "https://www.tce.pr.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Paraná (TCE-PR)",
		URLMunicipios:    "https://www.tce.pr.gov.br",
		AssembleiaNome:   "Assembleia Legislativa do Paraná (ALEP)",
		AssembleiaURL:    "https://www.assembleia.pr.leg.br",
		PortalEstado:     "https://www.pr.gov.br",
		TransparenciaGov: "https://transparencia.pr.gov.br",
		CapitalNome:      "Curitiba",
	},
	"PE": {
		NomeEstado:       "Tribunal de Contas do Estado de Pernambuco (TCE-PE)",
		URLEstado:        "https://www.tce.pe.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Pernambuco (TCE-PE - Tome Conta)",
		URLMunicipios:    "https://tomeconta.tce.pe.gov.br",
		AssembleiaNome:   "Assembleia Legislativa de Pernambuco (ALEPE)",
		AssembleiaURL:    "https://www.alepe.pe.gov.br",
		PortalEstado:     "https://www.pe.gov.br",
		TransparenciaGov: "https://transparencia.pe.gov.br",
		CapitalNome:      "Recife",
	},
	"PI": {
		NomeEstado:       "Tribunal de Contas do Estado do Piauí (TCE-PI)",
		URLEstado:        "https://www.tcepi.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Piauí (TCE-PI - Sagres)",
		URLMunicipios:    "https://sistemas.tce.pi.gov.br/sagresws",
		AssembleiaNome:   "Assembleia Legislativa do Estado do Piauí (ALEPI)",
		AssembleiaURL:    "https://www.al.pi.leg.br",
		PortalEstado:     "https://www.pi.gov.br",
		TransparenciaGov: "https://transparencia.pi.gov.br",
		CapitalNome:      "Teresina",
	},
	"RJ": {
		NomeEstado:       "Tribunal de Contas do Estado do Rio de Janeiro (TCE-RJ)",
		URLEstado:        "https://www.tcerj.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Rio de Janeiro (TCE-RJ)",
		URLMunicipios:    "https://www.tcerj.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado do Rio de Janeiro (ALERJ)",
		AssembleiaURL:    "https://www.alerj.rj.gov.br",
		PortalEstado:     "https://www.rj.gov.br",
		TransparenciaGov: "https://transparencia.rj.gov.br",
		CapitalNome:      "Rio de Janeiro",
	},
	"RN": {
		NomeEstado:       "Tribunal de Contas do Estado do Rio Grande do Norte (TCE-RN)",
		URLEstado:        "https://www.tce.rn.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Rio Grande do Norte (TCE-RN)",
		URLMunicipios:    "https://www.tce.rn.gov.br",
		AssembleiaNome:   "Assembleia Legislativa do Rio Grande do Norte (ALRN)",
		AssembleiaURL:    "https://www.al.rn.leg.br",
		PortalEstado:     "https://www.rn.gov.br",
		TransparenciaGov: "https://transparencia.rn.gov.br",
		CapitalNome:      "Natal",
	},
	"RS": {
		NomeEstado:       "Tribunal de Contas do Estado do Rio Grande do Sul (TCE-RS)",
		URLEstado:        "https://portal.tce.rs.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Rio Grande do Sul (TCE-RS - Painel Cidadão)",
		URLMunicipios:    "https://portal.tce.rs.gov.br/dadosabertos",
		AssembleiaNome:   "Assembleia Legislativa do Estado do Rio Grande do Sul (ALRS)",
		AssembleiaURL:    "https://www.al.rs.gov.br",
		PortalEstado:     "https://www.rs.gov.br",
		TransparenciaGov: "https://transparencia.rs.gov.br",
		CapitalNome:      "Porto Alegre",
	},
	"RO": {
		NomeEstado:       "Tribunal de Contas do Estado de Rondônia (TCE-RO)",
		URLEstado:        "https://tcero.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Rondônia (TCE-RO)",
		URLMunicipios:    "https://tcero.tc.br",
		AssembleiaNome:   "Assembleia Legislativa de Rondônia (ALERO)",
		AssembleiaURL:    "https://www.al.ro.leg.br",
		PortalEstado:     "https://rondonia.ro.gov.br",
		TransparenciaGov: "https://transparencia.ro.gov.br",
		CapitalNome:      "Porto Velho",
	},
	"RR": {
		NomeEstado:       "Tribunal de Contas do Estado de Roraima (TCE-RR)",
		URLEstado:        "https://tce.rr.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Roraima (TCE-RR)",
		URLMunicipios:    "https://tce.rr.gov.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado de Roraima (ALERR)",
		AssembleiaURL:    "https://al.rr.leg.br",
		PortalEstado:     "https://roraima.rr.gov.br",
		TransparenciaGov: "https://transparencia.rr.gov.br",
		CapitalNome:      "Boa Vista",
	},
	"SC": {
		NomeEstado:       "Tribunal de Contas do Estado de Santa Catarina (TCE-SC)",
		URLEstado:        "https://www.tcesc.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Santa Catarina (TCE-SC)",
		URLMunicipios:    "https://www.tcesc.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado de Santa Catarina (ALESC)",
		AssembleiaURL:    "https://www.alesc.sc.gov.br",
		PortalEstado:     "https://www.sc.gov.br",
		TransparenciaGov: "https://transparencia.sc.gov.br",
		CapitalNome:      "Florianópolis",
	},
	"SP": {
		NomeEstado:       "Tribunal de Contas do Estado de São Paulo (TCE-SP)",
		URLEstado:        "https://www.tce.sp.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de São Paulo (TCE-SP)",
		URLMunicipios:    "https://www.tce.sp.gov.br",
		AssembleiaNome:   "Assembleia Legislativa do Estado de São Paulo (ALESP)",
		AssembleiaURL:    "https://www.al.sp.gov.br",
		PortalEstado:     "https://www.saopaulo.sp.gov.br",
		TransparenciaGov: "https://transparencia.sp.gov.br",
		CapitalNome:      "São Paulo",
	},
	"SE": {
		NomeEstado:       "Tribunal de Contas do Estado de Sergipe (TCE-SE)",
		URLEstado:        "https://tce.se.gov.br",
		NomeMunicipios:   "Tribunal de Contas do Estado de Sergipe (TCE-SE)",
		URLMunicipios:    "https://tce.se.gov.br",
		AssembleiaNome:   "Assembleia Legislativa de Sergipe (ALESE)",
		AssembleiaURL:    "https://al.se.leg.br",
		PortalEstado:     "https://www.se.gov.br",
		TransparenciaGov: "https://transparencia.se.gov.br",
		CapitalNome:      "Aracaju",
	},
	"TO": {
		NomeEstado:       "Tribunal de Contas do Estado do Tocantins (TCE-TO)",
		URLEstado:        "https://www.tceto.tc.br",
		NomeMunicipios:   "Tribunal de Contas do Estado do Tocantins (TCE-TO)",
		URLMunicipios:    "https://www.tceto.tc.br",
		AssembleiaNome:   "Assembleia Legislativa do Tocantins (ALETO)",
		AssembleiaURL:    "https://al.to.leg.br",
		PortalEstado:     "https://www.to.gov.br",
		TransparenciaGov: "https://transparencia.to.gov.br",
		CapitalNome:      "Palmas",
	},
}

// BuildMunicipioDossier cria o dossiê consolidado de transparência de um município.
func BuildMunicipioDossier(m *Municipio, e *Estado) *MunicipioDossier {
	ufLower := strings.ToLower(m.UF)
	slug := m.Slug

	// URLs municipais diretas
	transpPrefeitura := fmt.Sprintf("https://transparencia.%s.%s.gov.br", slug, ufLower)
	sitePrefeitura := fmt.Sprintf("https://www.%s.%s.gov.br", slug, ufLower)
	siteCamara := fmt.Sprintf("https://camara%s.%s.leg.br", slug, ufLower)
	if mega, ok := LookupMegasoftByIBGE(m.CodigoIBGE); ok {
		if mega.PortalURL != "" {
			transpPrefeitura = mega.PortalURL
		}
		if mega.SitePrefeitura != "" {
			sitePrefeitura = mega.SitePrefeitura
		}
		if mega.PortalCamara != "" {
			siteCamara = mega.PortalCamara
		}
	}

	// Tribunal de contas competente
	tribunalNome := ""
	tribunalURL := ""
	if m.CodigoIBGE == "3550308" { // São Paulo capital tem TCM próprio
		tribunalNome = "Tribunal de Contas do Município de São Paulo (TCM-SP)"
		tribunalURL = "https://www.tcm.sp.gov.br"
	} else if m.CodigoIBGE == "3304557" { // Rio de Janeiro capital tem TCM próprio
		tribunalNome = "Tribunal de Contas do Município do Rio de Janeiro (TCM-Rio)"
		tribunalURL = "https://www.tcm.rj.gov.br"
	} else if info, ok := ufsInfo[m.UF]; ok {
		tribunalNome = info.NomeMunicipios
		tribunalURL = info.URLMunicipios
	}

	radarURL := "https://radardatransparencia.atricon.org.br/radar-da-transparencia-publica.html"
	siconfiURL := fmt.Sprintf("https://siconfi.tesouro.gov.br/siconfi/index.jsf?cod_ibge=%s", m.CodigoIBGE)
	siconfiAPI := fmt.Sprintf("https://apidatalake.tesouro.gov.br/ords/siconfi/tt/entes?cod_ibge=%s", m.CodigoIBGE)
	pncpURL := fmt.Sprintf("https://pncp.gov.br/app/editais?uf=%s", m.UF)
	transferegovURL := fmt.Sprintf("https://transferegov.sistema.gov.br/portal/painel-repasses?uf=%s", m.UF)
	transpFederal := fmt.Sprintf("https://portaldatransparencia.gov.br/localidades/%s-%s", m.CodigoIBGE, slug)

	pncpCNPJURL := ""
	if m.CNPJ != "" {
		pncpCNPJURL = fmt.Sprintf("https://pncp.gov.br/app/editais?q=%s&uf=%s", m.CNPJ, m.UF)
	}

	dcaEndpoint := fmt.Sprintf("https://apidatalake.tesouro.gov.br/ords/siconfi/tt/dca?an_exercicio=2023&id_ente=%s", m.CodigoIBGE)
	rreoEndpoint := fmt.Sprintf("https://apidatalake.tesouro.gov.br/ords/siconfi/tt/rreo?an_exercicio=2024&nr_periodo=1&co_tipo_demonstrativo=RREO&id_ente=%s", m.CodigoIBGE)
	entregasEndpoint := fmt.Sprintf("https://apidatalake.tesouro.gov.br/ords/siconfi/tt/extrato_entregas?an_referencia=2024&id_ente=%s", m.CodigoIBGE)

	return &MunicipioDossier{
		Municipio:                     m,
		Estado:                        e,
		CNPJ:                          m.CNPJ,
		Populacao:                     m.Populacao,
		PortalTransparenciaPrefeitura: transpPrefeitura,
		PortalPrefeitura:              sitePrefeitura,
		PortalCamara:                  siteCamara,
		TribunalContasCompetente:      tribunalNome,
		TribunalContasURL:             tribunalURL,
		RadarTransparenciaURL:         radarURL,
		SiconfiURL:                    siconfiURL,
		SiconfiAPI:                    siconfiAPI,
		PNCPURL:                       pncpURL,
		PNCPCNPJURL:                   pncpCNPJURL,
		TransferegovURL:               transferegovURL,
		TransparenciaFederalURL:       transpFederal,
		SiconfiDCAEndpoint:            dcaEndpoint,
		SiconfiRREOEndpoint:           rreoEndpoint,
		SiconfiEntregasEndpoint:       entregasEndpoint,
	}
}

// BuildEstadoDossier cria o dossiê consolidado de transparência de uma UF.
func BuildEstadoDossier(e *Estado, totalMunis int, principais []*Municipio) *EstadoDossier {
	siconfiURL := fmt.Sprintf("https://siconfi.tesouro.gov.br/siconfi/index.jsf?cod_ibge=%d", e.CodigoUF)
	siconfiAPI := fmt.Sprintf("https://apidatalake.tesouro.gov.br/ords/siconfi/tt/entes?cod_ibge=%d", e.CodigoUF)
	pncpURL := fmt.Sprintf("https://pncp.gov.br/app/editais?uf=%s", e.UF)

	return &EstadoDossier{
		Estado:                e,
		TotalMunicipios:       totalMunis,
		MunicipiosPrincipais:  principais,
		PortalTransparencia:   e.PortalTransparencia,
		PortalGoverno:         e.PortalGoverno,
		TribunalContasNome:    e.TribunalContasNome,
		TribunalContasURL:     e.TribunalContasURL,
		AssembleiaNome:        e.AssembleiaNome,
		AssembleiaURL:         e.AssembleiaURL,
		RadarTransparenciaURL: e.RadarTransparenciaURL,
		SiconfiURL:            siconfiURL,
		SiconfiAPI:            siconfiAPI,
		PNCPURL:               pncpURL,
	}
}
