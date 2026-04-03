"""Constants for the DataJud (CNJ) feature."""

# API base
DATAJUD_API_BASE = "https://api-publica.datajud.cnj.jus.br/api_publica_"

# Default pagination
DEFAULT_PAGE_SIZE = 10
MAX_PAGE_SIZE = 10000

# Tribunal endpoint mapping (sigla → sufixo da URL)
TRIBUNAIS: dict[str, str] = {
    # Tribunais Superiores (STF não está disponível no DataJud)
    "stj": "stj",
    "tst": "tst",
    "stm": "stm",
    "tse": "tse",
    # Tribunais Regionais Federais
    "trf1": "trf1",
    "trf2": "trf2",
    "trf3": "trf3",
    "trf4": "trf4",
    "trf5": "trf5",
    "trf6": "trf6",
    # Tribunais Regionais do Trabalho
    "trt1": "trt1",
    "trt2": "trt2",
    "trt3": "trt3",
    "trt4": "trt4",
    "trt5": "trt5",
    "trt6": "trt6",
    "trt7": "trt7",
    "trt8": "trt8",
    "trt9": "trt9",
    "trt10": "trt10",
    "trt11": "trt11",
    "trt12": "trt12",
    "trt13": "trt13",
    "trt14": "trt14",
    "trt15": "trt15",
    "trt16": "trt16",
    "trt17": "trt17",
    "trt18": "trt18",
    "trt19": "trt19",
    "trt20": "trt20",
    "trt21": "trt21",
    "trt22": "trt22",
    "trt23": "trt23",
    "trt24": "trt24",
    # Tribunais de Justiça Estaduais
    "tjac": "tjac",
    "tjal": "tjal",
    "tjam": "tjam",
    "tjap": "tjap",
    "tjba": "tjba",
    "tjce": "tjce",
    "tjdft": "tjdft",
    "tjes": "tjes",
    "tjgo": "tjgo",
    "tjma": "tjma",
    "tjmg": "tjmg",
    "tjms": "tjms",
    "tjmt": "tjmt",
    "tjpa": "tjpa",
    "tjpb": "tjpb",
    "tjpe": "tjpe",
    "tjpi": "tjpi",
    "tjpr": "tjpr",
    "tjrj": "tjrj",
    "tjrn": "tjrn",
    "tjro": "tjro",
    "tjrr": "tjrr",
    "tjrs": "tjrs",
    "tjsc": "tjsc",
    "tjse": "tjse",
    "tjsp": "tjsp",
    "tjto": "tjto",
    # Tribunais Regionais Eleitorais
    "tre-ac": "tre-ac",
    "tre-al": "tre-al",
    "tre-am": "tre-am",
    "tre-ap": "tre-ap",
    "tre-ba": "tre-ba",
    "tre-ce": "tre-ce",
    "tre-dft": "tre-dft",
    "tre-es": "tre-es",
    "tre-go": "tre-go",
    "tre-ma": "tre-ma",
    "tre-mg": "tre-mg",
    "tre-ms": "tre-ms",
    "tre-mt": "tre-mt",
    "tre-pa": "tre-pa",
    "tre-pb": "tre-pb",
    "tre-pe": "tre-pe",
    "tre-pi": "tre-pi",
    "tre-pr": "tre-pr",
    "tre-rj": "tre-rj",
    "tre-rn": "tre-rn",
    "tre-ro": "tre-ro",
    "tre-rr": "tre-rr",
    "tre-rs": "tre-rs",
    "tre-sc": "tre-sc",
    "tre-se": "tre-se",
    "tre-sp": "tre-sp",
    "tre-to": "tre-to",
    # Justiça Militar Estadual
    "tjmmg": "tjmmg",
    "tjmrs": "tjmrs",
    "tjmsp": "tjmsp",
}

# Nomes legíveis dos tribunais
TRIBUNAL_NOMES: dict[str, str] = {
    "stj": "Superior Tribunal de Justiça",
    "tst": "Tribunal Superior do Trabalho",
    "stm": "Superior Tribunal Militar",
    "tse": "Tribunal Superior Eleitoral",
    "trf1": "TRF 1ª Região (DF, GO, MT, TO, AC, AM, AP, BA, MA, MG, PA, PI, RO, RR)",
    "trf2": "TRF 2ª Região (RJ, ES)",
    "trf3": "TRF 3ª Região (SP, MS)",
    "trf4": "TRF 4ª Região (RS, PR, SC)",
    "trf5": "TRF 5ª Região (PE, CE, AL, SE, RN, PB)",
    "trf6": "TRF 6ª Região (MG)",
    "tjsp": "Tribunal de Justiça de São Paulo",
    "tjrj": "Tribunal de Justiça do Rio de Janeiro",
    "tjmg": "Tribunal de Justiça de Minas Gerais",
    "tjrs": "Tribunal de Justiça do Rio Grande do Sul",
    "tjpr": "Tribunal de Justiça do Paraná",
    "tjsc": "Tribunal de Justiça de Santa Catarina",
    "tjba": "Tribunal de Justiça da Bahia",
    "tjpe": "Tribunal de Justiça de Pernambuco",
    "tjce": "Tribunal de Justiça do Ceará",
    "tjgo": "Tribunal de Justiça de Goiás",
    "tjdft": "Tribunal de Justiça do Distrito Federal e Territórios",
    # TREs
    "tre-ac": "Tribunal Regional Eleitoral do Acre",
    "tre-al": "Tribunal Regional Eleitoral de Alagoas",
    "tre-am": "Tribunal Regional Eleitoral do Amazonas",
    "tre-ap": "Tribunal Regional Eleitoral do Amapá",
    "tre-ba": "Tribunal Regional Eleitoral da Bahia",
    "tre-ce": "Tribunal Regional Eleitoral do Ceará",
    "tre-dft": "Tribunal Regional Eleitoral do Distrito Federal",
    "tre-es": "Tribunal Regional Eleitoral do Espírito Santo",
    "tre-go": "Tribunal Regional Eleitoral de Goiás",
    "tre-ma": "Tribunal Regional Eleitoral do Maranhão",
    "tre-mg": "Tribunal Regional Eleitoral de Minas Gerais",
    "tre-ms": "Tribunal Regional Eleitoral do Mato Grosso do Sul",
    "tre-mt": "Tribunal Regional Eleitoral do Mato Grosso",
    "tre-pa": "Tribunal Regional Eleitoral do Pará",
    "tre-pb": "Tribunal Regional Eleitoral da Paraíba",
    "tre-pe": "Tribunal Regional Eleitoral de Pernambuco",
    "tre-pi": "Tribunal Regional Eleitoral do Piauí",
    "tre-pr": "Tribunal Regional Eleitoral do Paraná",
    "tre-rj": "Tribunal Regional Eleitoral do Rio de Janeiro",
    "tre-rn": "Tribunal Regional Eleitoral do Rio Grande do Norte",
    "tre-ro": "Tribunal Regional Eleitoral de Rondônia",
    "tre-rr": "Tribunal Regional Eleitoral de Roraima",
    "tre-rs": "Tribunal Regional Eleitoral do Rio Grande do Sul",
    "tre-sc": "Tribunal Regional Eleitoral de Santa Catarina",
    "tre-se": "Tribunal Regional Eleitoral de Sergipe",
    "tre-sp": "Tribunal Regional Eleitoral de São Paulo",
    "tre-to": "Tribunal Regional Eleitoral de Tocantins",
    # Justiça Militar
    "tjmmg": "Tribunal de Justiça Militar de Minas Gerais",
    "tjmrs": "Tribunal de Justiça Militar do Rio Grande do Sul",
    "tjmsp": "Tribunal de Justiça Militar de São Paulo",
}

# Classes processuais comuns
CLASSES_PROCESSUAIS: list[dict[str, str]] = [
    {"codigo": "1116", "nome": "Ação Civil Pública"},
    {"codigo": "7", "nome": "Ação Penal - Procedimento Ordinário"},
    {"codigo": "12078", "nome": "Execução Fiscal"},
    {"codigo": "1386", "nome": "Mandado de Segurança Cível"},
    {"codigo": "12158", "nome": "Procedimento Comum Cível"},
    {"codigo": "175", "nome": "Recurso Extraordinário"},
    {"codigo": "205", "nome": "Recurso Especial"},
    {"codigo": "1009", "nome": "Habeas Corpus"},
    {"codigo": "968", "nome": "Habeas Data"},
    {"codigo": "331", "nome": "Ação Direta de Inconstitucionalidade"},
]


# === MEDIDAS PROTETIVAS DE URGÊNCIA (MPU) ===

# Classes processuais — Lei Maria da Penha (11.340/2006)
MPU_CLASSE_CRIMINAL = 1268
MPU_CLASSE_INFRACIONAL = 12423
MPU_CLASSE_CIVEL = 15309
MPU_CLASSES_MARIA_PENHA = [MPU_CLASSE_CRIMINAL, MPU_CLASSE_INFRACIONAL, MPU_CLASSE_CIVEL]

# Classes processuais — Lei Henry Borel (14.344/2022)
MPU_CLASSE_HENRY_CRIMINAL = 15170
MPU_CLASSE_HENRY_INFRACIONAL = 15171
MPU_CLASSES_HENRY_BOREL = [MPU_CLASSE_HENRY_CRIMINAL, MPU_CLASSE_HENRY_INFRACIONAL]

MPU_CLASSES_TODAS = MPU_CLASSES_MARIA_PENHA + MPU_CLASSES_HENRY_BOREL

# Movimentos processuais — novos (atualizados em 29/11/2024)
MPU_MOV_CONCEDIDA = 15486
MPU_MOV_CONCEDIDA_PARCIAL = 15487
MPU_MOV_NAO_CONCEDIDA = 15488
MPU_MOV_REVOGADA = 15489
MPU_MOV_PRORROGADA = 15490

# Movimentos processuais — legados (pré Nov/2024, ainda em processos antigos)
MPU_MOV_CONCEDIDA_LEGADO = 11423
MPU_MOV_CONCEDIDA_PARCIAL_LEGADO = 11424
MPU_MOV_NAO_CONCEDIDA_LEGADO = 11425
MPU_MOV_REVOGADA_LEGADO = 11426
MPU_MOV_PRORROGADA_LEGADO = 14733

# Agregações de movimentos
MPU_MOV_CONCESSAO_TODOS = [
    MPU_MOV_CONCEDIDA,
    MPU_MOV_CONCEDIDA_LEGADO,
    MPU_MOV_CONCEDIDA_PARCIAL,
    MPU_MOV_CONCEDIDA_PARCIAL_LEGADO,
]

MPU_MOV_TODOS_NOVOS = [
    MPU_MOV_CONCEDIDA,
    MPU_MOV_CONCEDIDA_PARCIAL,
    MPU_MOV_NAO_CONCEDIDA,
    MPU_MOV_REVOGADA,
    MPU_MOV_PRORROGADA,
]

MPU_MOV_TODOS_LEGADOS = [
    MPU_MOV_CONCEDIDA_LEGADO,
    MPU_MOV_CONCEDIDA_PARCIAL_LEGADO,
    MPU_MOV_NAO_CONCEDIDA_LEGADO,
    MPU_MOV_REVOGADA_LEGADO,
    MPU_MOV_PRORROGADA_LEGADO,
]

MPU_MOV_TODOS = MPU_MOV_TODOS_NOVOS + MPU_MOV_TODOS_LEGADOS

# Complementos tabelados
MPU_COMPLEMENTO_DESTINATARIO = 31
MPU_COMPLEMENTO_TIPO_MEDIDA = 32

# Valores do complemento 31 (destinatário da medida protetiva)
MPU_DEST_MULHER = 124
MPU_DEST_IDOSO = 125
MPU_DEST_CRIANCA = 126

MPU_DESTINATARIOS: dict[str, int] = {
    "mulher": MPU_DEST_MULHER,
    "idoso": MPU_DEST_IDOSO,
    "crianca": MPU_DEST_CRIANCA,
}

# Valores do complemento 32 (tipo de medida protetiva)
MPU_TIPO_AFASTAMENTO_LAR = 128
MPU_TIPO_PROIBICAO_APROXIMACAO = 129
MPU_TIPO_PROIBICAO_CONTATO = 130
MPU_TIPO_PROIBICAO_FREQUENTAR = 131
MPU_TIPO_RESTRICAO_VISITAS = 132
MPU_TIPO_ALIMENTOS_PROVISORIOS = 133
MPU_TIPO_REABILITACAO_AGRESSOR = 134
MPU_TIPO_MONITORAMENTO_ELETRONICO = 135
MPU_TIPO_OUTRAS = 136

MPU_TIPOS_MEDIDA: dict[str, int] = {
    "afastamento_lar": MPU_TIPO_AFASTAMENTO_LAR,
    "proibicao_aproximacao": MPU_TIPO_PROIBICAO_APROXIMACAO,
    "proibicao_contato": MPU_TIPO_PROIBICAO_CONTATO,
    "proibicao_frequentar": MPU_TIPO_PROIBICAO_FREQUENTAR,
    "restricao_visitas": MPU_TIPO_RESTRICAO_VISITAS,
    "alimentos_provisorios": MPU_TIPO_ALIMENTOS_PROVISORIOS,
    "reabilitacao_agressor": MPU_TIPO_REABILITACAO_AGRESSOR,
    "monitoramento_eletronico": MPU_TIPO_MONITORAMENTO_ELETRONICO,
    "outras": MPU_TIPO_OUTRAS,
}

# Nomes legíveis dos movimentos MPU
MPU_MOV_NOMES: dict[int, str] = {
    MPU_MOV_CONCEDIDA: "Concedida medida protetiva",
    MPU_MOV_CONCEDIDA_PARCIAL: "Concedida em parte medida protetiva",
    MPU_MOV_NAO_CONCEDIDA: "Não concedida medida protetiva",
    MPU_MOV_REVOGADA: "Revogada medida protetiva",
    MPU_MOV_PRORROGADA: "Prorrogada a medida protetiva",
    MPU_MOV_CONCEDIDA_LEGADO: "Concedida medida protetiva (legado)",
    MPU_MOV_CONCEDIDA_PARCIAL_LEGADO: "Concedida em parte medida protetiva (legado)",
    MPU_MOV_NAO_CONCEDIDA_LEGADO: "Não concedida medida protetiva (legado)",
    MPU_MOV_REVOGADA_LEGADO: "Revogada medida protetiva (legado)",
    MPU_MOV_PRORROGADA_LEGADO: "Prorrogada a medida protetiva (legado)",
}

# Nomes legíveis das classes MPU
MPU_CLASSES_NOMES: dict[int, str] = {
    MPU_CLASSE_CRIMINAL: "MPU (Maria da Penha) — Criminal",
    MPU_CLASSE_INFRACIONAL: "MPU (Maria da Penha) — Infracional",
    MPU_CLASSE_CIVEL: "MPU (Maria da Penha) — Cível",
    MPU_CLASSE_HENRY_CRIMINAL: "MPU (Henry Borel) — Criminal",
    MPU_CLASSE_HENRY_INFRACIONAL: "MPU (Henry Borel) — Infracional",
}

# Assuntos relacionados a MPU
MPU_ASSUNTO_CIVEL = 15511
MPU_ASSUNTO_VIOLENCIA_DOMESTICA = 11529
MPU_ASSUNTO_MARIA_PENHA = 3596

MPU_ASSUNTOS: dict[int, str] = {
    MPU_ASSUNTO_CIVEL: "Medidas Protetivas de Urgência (Lei Maria da Penha) — Cível",
    MPU_ASSUNTO_VIOLENCIA_DOMESTICA: "Violência Doméstica Contra a Mulher",
    MPU_ASSUNTO_MARIA_PENHA: "Lei Maria da Penha",
}
