"""Pydantic schemas for transparency of Brazilian municipalities and states."""

from pydantic import BaseModel, Field


class Municipio(BaseModel):
    codigo_ibge: str = Field(description="Código IBGE oficial de 7 dígitos")
    nome: str = Field(description="Nome do município")
    latitude: float
    longitude: float
    capital: bool = Field(description="Se é a capital da UF")
    codigo_uf: int = Field(description="Código da UF no IBGE")
    uf: str = Field(description="Sigla da Unidade Federativa")
    regiao: str = Field(description="Região geográfica do Brasil")
    siafi_id: str = Field(description="Código SIAFI de 4 dígitos")
    ddd: str = Field(description="Código DDD de telefonia")
    fuso_horario: str
    slug: str = Field(description="Slug normalizado para URLs")
    cnpj: str = Field(default="", description="CNPJ oficial da Prefeitura")
    populacao: int = Field(default=0, description="População estimada do município")


class Estado(BaseModel):
    codigo_uf: int = Field(description="Código da UF no IBGE")
    uf: str = Field(description="Sigla da UF (2 letras)")
    nome: str = Field(description="Nome da Unidade Federativa")
    latitude: float
    longitude: float
    regiao: str = Field(description="Região geográfica")
    capital: str = Field(description="Nome da capital")
    total_municipios: int = Field(description="Total de municípios na UF")
    portal_transparencia: str = Field(description="URL do Portal da Transparência estadual")
    portal_governo: str = Field(description="URL do Portal do Governo do Estado")
    tribunal_contas_nome: str = Field(description="Nome do Tribunal de Contas (TCE)")
    tribunal_contas_url: str = Field(description="URL do Tribunal de Contas")
    assembleia_nome: str = Field(description="Nome do Poder Legislativo estadual")
    assembleia_url: str = Field(description="URL da Assembleia Legislativa")
    siconfi_ente_id: str = Field(description="Identificador no SICONFI")
    radar_transparencia_url: str = Field(description="URL de avaliação no Radar ATRICON")
    pncp_url: str = Field(description="URL de contratações no PNCP")


class MunicipioDossier(BaseModel):
    municipio: Municipio
    estado: Estado | None = None
    cnpj: str = Field(default="", description="CNPJ oficial da Prefeitura")
    populacao: int = Field(default=0, description="População estimada do município")
    portal_transparencia_prefeitura: str = Field(
        description="URL do Portal da Transparência Municipal"
    )
    portal_prefeitura: str = Field(description="URL do Site Oficial da Prefeitura")
    portal_camara: str = Field(description="URL da Câmara Municipal de Vereadores")
    tribunal_contas_competente: str = Field(
        description="Tribunal de Contas fiscalizador (TCE ou TCM)"
    )
    tribunal_contas_url: str = Field(description="URL do Tribunal de Contas competente")
    radar_transparencia_url: str = Field(
        description="URL do Radar Nacional da Transparência Pública (ATRICON)"
    )
    siconfi_url: str = Field(description="Link das declarações fiscais no SICONFI")
    siconfi_api: str = Field(description="Endpoint da API pública do SICONFI")
    pncp_url: str = Field(description="Link de consulta de licitações e contratos no PNCP")
    pncp_cnpj_url: str = Field(
        default="", description="Link de editais e contratações no PNCP filtrado por CNPJ"
    )
    transferegov_url: str = Field(description="Link de transferências no Transferegov")
    transparencia_federal_url: str = Field(
        description="Link de transferências e benefícios no Portal da Transparência CGU"
    )
    siconfi_dca_endpoint: str = Field(
        default="", description="Endpoint de dados fiscais anuais DCA no SICONFI"
    )
    siconfi_rreo_endpoint: str = Field(
        default="", description="Endpoint de execução orçamentária RREO no SICONFI"
    )
    siconfi_entregas_endpoint: str = Field(
        default="", description="Endpoint de extrato de entregas da LRF no SICONFI"
    )


class EstadoDossier(BaseModel):
    estado: Estado
    total_municipios: int
    municipios_principais: list[Municipio] = Field(default_factory=list)
    portal_transparencia: str
    portal_governo: str
    tribunal_contas_nome: str
    tribunal_contas_url: str
    assembleia_nome: str
    assembleia_url: str
    radar_transparencia_url: str
    siconfi_url: str
    siconfi_api: str
    pncp_url: str


class SearchResult(BaseModel):
    query: str
    uf: str = ""
    total_encontrado: int
    municipios: list[Municipio] = Field(default_factory=list)


class Stats(BaseModel):
    total_municipios: int
    total_ufs: int
    populacao_total: int = 0
    municipios_por_regiao: dict[str, int]
    municipios_por_uf: dict[str, int]
    capitais: list[str]


class ComparativoItem(BaseModel):
    tipo: str  # "municipio" ou "estado"
    nome: str
    uf: str
    codigo_ibge: str
    siafi_id: str = ""
    cnpj: str = ""
    populacao: int = 0
    regiao: str
    capital: bool = False
    portal_transparencia: str
    tribunal_contas_nome: str
    tribunal_contas_url: str
    radar_transparencia_url: str
    siconfi_url: str
    pncp_url: str


class ComparativoRelatorio(BaseModel):
    total_itens: int
    itens: list[ComparativoItem] = Field(default_factory=list)


class FiscalItem(BaseModel):
    exercicio: int
    instituicao: str
    cod_ibge: int
    uf: str = ""
    anexo: str = ""
    rotulo: str = ""
    coluna: str = ""
    cod_conta: str = ""
    conta: str = ""
    valor: float = 0.0
    populacao: int = 0


class EntregaItem(BaseModel):
    exercicio: int
    cod_ibge: int
    instituicao: str
    entregavel: str
    periodo: int
    periodicidade: str
    status_relatorio: str | None = None
    data_status: str
    forma_envio: str


class ProbeResult(BaseModel):
    alvo: str
    url: str
    status_http: int | None = None
    tempo_ms: int
    online: bool
    erro: str = ""


class PortalDiscoveryResult(BaseModel):
    municipio: Municipio
    url_resolvida: str = ""
    site_prefeitura: str = ""
    portal_camara: str = ""
    plataforma: str = "nao_identificado"
    versao: str = ""
    cod_administracao: int = 0
    entidade_nome: str = ""
    fonte_descoberta: str = ""
    servidores_suportado: bool = False
    candidatos_testados: list[str] = Field(default_factory=list)
    tempo_ms: int = 0
    observacoes: str = ""


class ServidorItem(BaseModel):
    nome: str
    matricula: str = ""
    cpf_parcial: str = ""
    cargo: str = ""
    funcao: str = ""
    lotacao: str = ""
    local_trabalho: str = ""
    vinculo: str = ""
    situacao: str = ""
    salario_base: float = 0.0
    salario_bruto: float = 0.0
    descontos: float = 0.0
    salario_liquido: float = 0.0
    competencia: str = ""
    carga_horaria: str = ""
    data_admissao: str = ""
    orgao: str = ""
    fonte: str = ""
    portal_url: str = ""


class ServidoresResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    servidores: list[ServidorItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class DespesaItem(BaseModel):
    numero_empenho: str = ""
    ano: int = 0
    data: str = ""
    fornecedor: str = ""
    documento_fornecedor: str = ""
    orgao: str = ""
    unidade: str = ""
    funcao: str = ""
    elemento: str = ""
    historico: str = ""
    valor_empenhado: float = 0.0
    valor_liquidado: float = 0.0
    valor_pago: float = 0.0
    fonte: str = ""


class DespesasResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    despesas: list[DespesaItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class ReceitaItem(BaseModel):
    codigo: str = ""
    descricao: str = ""
    categoria: str = ""
    data: str = ""
    valor_previsto: float = 0.0
    valor_arrecadado: float = 0.0
    fonte: str = ""


class ReceitasResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    receitas: list[ReceitaItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class LicitacaoItem(BaseModel):
    numero: str = ""
    ano: int = 0
    modalidade: str = ""
    situacao: str = ""
    data_abertura: str = ""
    objeto: str = ""
    valor_estimado: float = 0.0
    valor_homologado: float = 0.0
    link: str = ""
    fonte: str = ""


class LicitacoesResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    licitacoes: list[LicitacaoItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class ContratoItem(BaseModel):
    numero: str = ""
    ano: int = 0
    fornecedor: str = ""
    documento_fornecedor: str = ""
    objeto: str = ""
    situacao: str = ""
    data_assinatura: str = ""
    vigencia_inicio: str = ""
    vigencia_fim: str = ""
    valor: float = 0.0
    fonte: str = ""


class ContratosResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    contratos: list[ContratoItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class DiariaItem(BaseModel):
    beneficiario: str
    matricula: str = ""
    cargo: str = ""
    destino: str = ""
    motivo: str = ""
    data: str = ""
    data_saida: str = ""
    data_chegada: str = ""
    numero_diarias: float = 0.0
    valor: float = 0.0
    fonte: str = ""


class DiariasResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    diarias: list[DiariaItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class ObraItem(BaseModel):
    codigo: str = ""
    titulo: str = ""
    objeto: str = ""
    orgao: str = ""
    endereco: str = ""
    bairro: str = ""
    data_inicial: str = ""
    data_final: str = ""
    situacao: str = ""
    valor: float = 0.0
    fonte: str = ""


class ObrasResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    obras: list[ObraItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class ConvenioItem(BaseModel):
    numero: str = ""
    concedente: str = ""
    convenente: str = ""
    categoria: str = ""
    tipo: str = ""
    situacao: str = ""
    data: str = ""
    vigencia: str = ""
    descricao: str = ""
    valor: float = 0.0
    valor_liberado: float = 0.0
    fonte: str = ""


class ConveniosResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    convenios: list[ConvenioItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class LegislacaoItem(BaseModel):
    codigo: str = ""
    tipo_documento: str = ""
    numero: str = ""
    ano: int = 0
    data_publicacao: str = ""
    situacao: str = ""
    nome: str = ""
    descricao: str = ""
    fonte: str = ""


class LegislacaoResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    atos: list[LegislacaoItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class ConcursoItem(BaseModel):
    edital: str = ""
    data_publicacao: str = ""
    nome: str = ""
    cargo: str = ""
    inscricao: str = ""
    classificacao: int = 0
    situacao: str = ""
    fonte: str = ""


class ConcursosResult(BaseModel):
    municipio: Municipio
    fonte: str
    plataforma: str
    portal_url: str = ""
    total_registros: int = 0
    aprovados: list[ConcursoItem] = Field(default_factory=list)
    disponivel: bool = False
    mensagem: str = ""


class VerbaItem(BaseModel):
    descricao: str = ""
    valor: float = 0.0


class ServidorDetalheResult(BaseModel):
    municipio: Municipio
    nome: str = ""
    matricula: str = ""
    cpf_parcial: str = ""
    cargo: str = ""
    orgao: str = ""
    departamento: str = ""
    vinculo: str = ""
    situacao: str = ""
    competencia: str = ""
    data_admissao: str = ""
    carga_horaria: str = ""
    salario_bruto: float = 0.0
    descontos: float = 0.0
    salario_liquido: float = 0.0
    vencimentos: list[VerbaItem] = Field(default_factory=list)
    verbas_desconto: list[VerbaItem] = Field(default_factory=list)
    ferias: list[VerbaItem] = Field(default_factory=list)
    decimo_terceiro: list[VerbaItem] = Field(default_factory=list)
    portal_url: str = ""
    fonte: str = ""
    disponivel: bool = False
    mensagem: str = ""


class MegasoftMunicipio(BaseModel):
    codigo_ibge: str
    nome: str
    uf: str
    portal_url: str
    portal_camara: str = ""
    codigo_orgao: int = 0
    site_prefeitura: str = ""


class TransparenciaCompletaReport(BaseModel):
    dossier: MunicipioDossier
    discovery: PortalDiscoveryResult | None = None
    servidores: ServidoresResult | None = None
    diarias: DiariasResult | None = None
    despesas: DespesasResult | None = None
    receitas: ReceitasResult | None = None
    licitacoes: LicitacoesResult | None = None
    contratos: ContratosResult | None = None
    obras: ObrasResult | None = None
    convenios: ConveniosResult | None = None
    legislacao: LegislacaoResult | None = None
    concursos: ConcursosResult | None = None
    siconfi_dca: list[FiscalItem] = Field(default_factory=list)
    siconfi_rgf: list[FiscalItem] = Field(default_factory=list)
    siconfi_entregas: list[EntregaItem] = Field(default_factory=list)
    fontes_consultadas: list[str] = Field(default_factory=list)
    limitacoes: list[str] = Field(default_factory=list)
    tempo_total_ms: int = 0
