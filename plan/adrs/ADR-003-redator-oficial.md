# ADR-003: Redator Oficial — Agent Tools para Documentos Oficiais

## Status

**Proposto** — 2026-03-22

## Contexto

O `mcp-brasil` atualmente foca em **consulta de dados** (IBGE, Bacen, Transparência). Mas um dos usos mais poderosos para govtech é a **geração de documentos oficiais** — despachos, memorandos, ofícios, portarias, pareceres — que seguem formatação e linguagem padronizadas.

A ideia é criar uma feature `redator` que combina os **3 primitivos do MCP**:

- **Prompts** → "Agentes especialistas" (Redator de Despacho, Redator de Memorando, etc.)
- **Resources** → Templates e normas de redação oficial (Manual de Redação da Presidência)
- **Tools** → Funções que geram, validam e formatam os documentos

Isso transforma o `mcp-brasil` de "consulta de dados" para **"assistente govtech completo"**.

---

## Como funciona no MCP

### Os 3 primitivos e seus papéis

```
┌─────────────────────────────────────────────────────────┐
│                    Claude Desktop                        │
│                                                         │
│  Usuário seleciona prompt "Redator de Despacho"         │
│  → MCP carrega o template + normas como contexto        │
│  → LLM usa tools para consultar dados (Bacen, IBGE)    │
│  → LLM gera o documento seguindo o template             │
│  → Tool formata e valida o documento final              │
└─────────────────────────────────────────────────────────┘

  PROMPT (agente)         RESOURCE (contexto)        TOOL (ação)
  ┌──────────────┐       ┌──────────────────┐      ┌──────────────────┐
  │ @mcp.prompt  │       │ @mcp.resource    │      │ @mcp.tool        │
  │              │       │                  │      │                  │
  │ redator_     │  ───▶ │ template://      │ ───▶ │ gerar_documento  │
  │ despacho()   │       │ despacho         │      │ validar_doc      │
  │              │       │                  │      │ formatar_data    │
  │ Instrui o    │       │ Estrutura +      │      │ consultar_dados  │
  │ LLM sobre    │       │ normas +         │      │ (via outras      │
  │ como redigir │       │ exemplos         │      │  features)       │
  └──────────────┘       └──────────────────┘      └──────────────────┘
```

### Fluxo real de uso

```
Usuário no Claude Desktop:
  → Seleciona prompt "Redator de Despacho" (aparece como slash command)
  → Preenche: "Aprovar contratação de 3 servidores temporários para TI"

Claude recebe:
  1. Prompt do redator (role, instruções, formato)
  2. Resource com template de despacho (estrutura obrigatória)
  3. Resource com normas de redação oficial (vocabulário, pronomes de tratamento)

Claude executa:
  4. Tool formatar_data() → "Teresina, 22 de março de 2026"
  5. Tool consultar_norma() → retorna trecho relevante do manual
  6. Gera o texto do despacho seguindo o template

Resultado:
  Documento formatado, pronto para copiar ou exportar
```

---

## Arquitetura dentro do mcp-brasil

### Nova feature: `redator/`

```
src/mcp_brasil/
├── redator/                    # ★ Nova feature auto-descoberta
│   ├── __init__.py             # FEATURE_META
│   ├── server.py               # Registra tools + prompts + resources
│   ├── tools.py                # Funções de geração e validação
│   ├── prompts.py              # Prompts especializados por tipo de documento
│   ├── resources.py            # Templates e normas como MCP resources
│   ├── templates/              # Templates dos documentos (Markdown/texto)
│   │   ├── despacho.md
│   │   ├── memorando.md
│   │   ├── oficio.md
│   │   ├── portaria.md
│   │   ├── parecer.md
│   │   ├── nota_tecnica.md
│   │   └── ata.md
│   ├── normas/                 # Normas de redação oficial
│   │   ├── manual_redacao.md   # Resumo do Manual de Redação da Presidência
│   │   ├── pronomes.md         # Pronomes de tratamento
│   │   └── fechos.md           # Fechos e saudações
│   ├── schemas.py              # Pydantic models
│   └── constants.py            # Tipos de documentos, enums
```

---

## Implementação

### 1. Feature Meta — `redator/__init__.py`

```python
"""Feature Redator Oficial — Geração de documentos oficiais."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="redator",
    description="Redação de documentos oficiais: despacho, memorando, ofício, portaria, parecer",
    version="0.1.0",
    requires_auth=False,
    tags=["documentos", "redacao-oficial", "govtech"],
)
```

### 2. Templates como Resources — `redator/resources.py`

```python
"""Resources: templates e normas de redação oficial.

Resources são dados que o LLM carrega como contexto.
O client MCP pode ler esses resources para entender
a estrutura e as normas de cada tipo de documento.
"""

from pathlib import Path

TEMPLATES_DIR = Path(__file__).parent / "templates"
NORMAS_DIR = Path(__file__).parent / "normas"


def _load_file(directory: Path, filename: str) -> str:
    """Carrega um arquivo de template ou norma."""
    filepath = directory / filename
    if not filepath.exists():
        raise FileNotFoundError(f"Arquivo não encontrado: {filepath}")
    return filepath.read_text(encoding="utf-8")


# === Templates de documentos ===

def get_template_despacho() -> str:
    """Template oficial para Despacho administrativo."""
    return _load_file(TEMPLATES_DIR, "despacho.md")


def get_template_memorando() -> str:
    """Template oficial para Memorando interno."""
    return _load_file(TEMPLATES_DIR, "memorando.md")


def get_template_oficio() -> str:
    """Template oficial para Ofício externo."""
    return _load_file(TEMPLATES_DIR, "oficio.md")


def get_template_portaria() -> str:
    """Template oficial para Portaria."""
    return _load_file(TEMPLATES_DIR, "portaria.md")


def get_template_parecer() -> str:
    """Template oficial para Parecer técnico."""
    return _load_file(TEMPLATES_DIR, "parecer.md")


def get_template_nota_tecnica() -> str:
    """Template oficial para Nota Técnica."""
    return _load_file(TEMPLATES_DIR, "nota_tecnica.md")


def get_template_ata() -> str:
    """Template oficial para Ata de reunião."""
    return _load_file(TEMPLATES_DIR, "ata.md")


# === Normas de redação ===

def get_manual_redacao() -> str:
    """Resumo das normas do Manual de Redação da Presidência da República."""
    return _load_file(NORMAS_DIR, "manual_redacao.md")


def get_pronomes_tratamento() -> str:
    """Tabela de pronomes de tratamento oficiais."""
    return _load_file(NORMAS_DIR, "pronomes.md")


def get_fechos_oficiais() -> str:
    """Fechos e saudações para correspondência oficial."""
    return _load_file(NORMAS_DIR, "fechos.md")
```

### 3. Prompts como "Agentes" — `redator/prompts.py`

```python
"""Prompts: agentes especializados por tipo de documento.

Cada prompt é um "agente virtual" que instrui o LLM sobre:
- Qual tipo de documento redigir
- Quais normas seguir
- Qual estrutura obrigatória
- Como usar as tools disponíveis

No Claude Desktop, esses prompts aparecem como opções
selecionáveis (similar a slash commands).
"""

from fastmcp.prompts.base import AssistantMessage, UserMessage


def redator_despacho(assunto: str, contexto: str = "") -> list:
    """Redator de Despacho — decisão administrativa sobre processo ou requerimento.

    Use este prompt para redigir despachos que aprovam, indeferem,
    encaminham ou determinam providências sobre processos administrativos.

    Args:
        assunto: O que o despacho deve decidir/encaminhar.
        contexto: Informações adicionais (número do processo, setor, etc.).
    """
    return [
        UserMessage(
            f"""Preciso redigir um DESPACHO oficial sobre: {assunto}

Contexto adicional: {contexto if contexto else 'Nenhum fornecido.'}

Instruções:
1. Consulte o template de despacho (resource template://despacho)
2. Consulte as normas de redação oficial (resource normas://manual_redacao)
3. Use a tool formatar_data_extenso() para a data atual
4. Use pronomes de tratamento adequados (resource normas://pronomes)
5. Siga a estrutura: cabeçalho → referência ao processo → fundamentação → decisão → fecho
6. Use linguagem impessoal, concisa, objetiva
7. Evite: gerúndio excessivo, jargão desnecessário, parágrafos longos"""
        ),
        AssistantMessage(
            "Entendido. Vou redigir o despacho seguindo as normas do Manual de "
            "Redação da Presidência da República. Consultando o template e as normas..."
        ),
    ]


def redator_memorando(
    destinatario: str,
    assunto: str,
    remetente: str = "",
) -> list:
    """Redator de Memorando — comunicação interna entre setores.

    Use este prompt para redigir memorandos de comunicação entre
    unidades administrativas de um mesmo órgão.

    Args:
        destinatario: Setor ou pessoa destinatária (ex: "Coordenação de TI").
        assunto: Tema do memorando.
        remetente: Setor remetente (opcional).
    """
    return [
        UserMessage(
            f"""Preciso redigir um MEMORANDO oficial.

De: {remetente if remetente else '[informar remetente]'}
Para: {destinatario}
Assunto: {assunto}

Instruções:
1. Consulte o template de memorando (resource template://memorando)
2. Siga o padrão: Mem. nº ___/ANO/SETOR
3. Linguagem direta, parágrafos curtos
4. Estrutura: identificação → exposição do assunto → providências solicitadas → fecho
5. Use a tool formatar_data_extenso() para a data
6. Fecho: "Atenciosamente" (para mesma hierarquia) ou "Respeitosamente" (para superior)"""
        ),
        AssistantMessage(
            "Vou redigir o memorando seguindo o padrão oficial. "
            "Consultando template e normas de redação..."
        ),
    ]


def redator_oficio(
    destinatario: str,
    cargo_destinatario: str,
    assunto: str,
    orgao_remetente: str = "",
) -> list:
    """Redator de Ofício — comunicação externa entre órgãos ou entidades.

    Use este prompt para redigir ofícios de comunicação oficial entre
    órgãos públicos, ou entre órgão público e entidade privada.

    Args:
        destinatario: Nome do destinatário.
        cargo_destinatario: Cargo (ex: "Secretário de Fazenda").
        assunto: Tema do ofício.
        orgao_remetente: Órgão que envia (opcional).
    """
    return [
        UserMessage(
            f"""Preciso redigir um OFÍCIO oficial.

Para: {destinatario} — {cargo_destinatario}
De: {orgao_remetente if orgao_remetente else '[informar órgão remetente]'}
Assunto: {assunto}

Instruções:
1. Consulte o template de ofício (resource template://oficio)
2. Siga o padrão: Of. nº ___/ANO/SETOR
3. Use o vocativo e pronome de tratamento corretos para o cargo
   (consulte resource normas://pronomes)
4. Estrutura: identificação → vocativo → exposição → solicitação/informação → fecho
5. Use a tool formatar_data_extenso() para data
6. Inclua local antes da data
7. Fecho conforme hierarquia (resource normas://fechos)"""
        ),
        AssistantMessage(
            "Vou redigir o ofício com o pronome de tratamento adequado ao cargo. "
            "Consultando normas de pronomes de tratamento e template..."
        ),
    ]


def redator_portaria(
    assunto: str,
    autoridade: str = "",
    fundamentacao: str = "",
) -> list:
    """Redator de Portaria — ato normativo de autoridade administrativa.

    Use este prompt para redigir portarias que nomeiam, designam,
    regulamentam ou determinam providências administrativas.

    Args:
        assunto: O que a portaria determina.
        autoridade: Autoridade que assina (ex: "Secretário de Administração").
        fundamentacao: Base legal (ex: "Lei nº 8.112/1990, art. 67").
    """
    return [
        UserMessage(
            f"""Preciso redigir uma PORTARIA oficial.

Autoridade: {autoridade if autoridade else '[informar autoridade]'}
Assunto: {assunto}
Fundamentação legal: {fundamentacao if fundamentacao else '[informar base legal]'}

Instruções:
1. Consulte o template de portaria (resource template://portaria)
2. Estrutura obrigatória:
   - Preâmbulo: "O [AUTORIDADE], no uso das atribuições que lhe confere..."
   - CONSIDERANDO (se necessário): justificativas
   - RESOLVE:
   - Artigos numerados (Art. 1º, Art. 2º...)
   - Data e assinatura
3. Linguagem normativa, imperativa
4. Cada artigo = uma disposição
5. Use a tool formatar_data_extenso()"""
        ),
        AssistantMessage(
            "Vou redigir a portaria com a estrutura normativa obrigatória. "
            "Consultando template e fundamentação..."
        ),
    ]


def redator_parecer(
    processo: str,
    consulta: str,
    area: str = "jurídico",
) -> list:
    """Redator de Parecer — manifestação técnica ou jurídica sobre consulta.

    Use este prompt para redigir pareceres técnicos, jurídicos ou
    administrativos em resposta a consultas formais.

    Args:
        processo: Número do processo ou referência.
        consulta: Pergunta ou tema a ser analisado.
        area: Área do parecer: "jurídico", "técnico", "contábil" (default: jurídico).
    """
    return [
        UserMessage(
            f"""Preciso redigir um PARECER {area.upper()} oficial.

Processo/Referência: {processo}
Consulta: {consulta}
Área: {area}

Instruções:
1. Consulte o template de parecer (resource template://parecer)
2. Estrutura obrigatória:
   - EMENTA: resumo em 2-3 linhas
   - I — DO RELATÓRIO: fatos e histórico
   - II — DA FUNDAMENTAÇÃO: análise técnica/jurídica
   - III — DA CONCLUSÃO: resposta objetiva à consulta
3. Cite legislação relevante (se souber)
4. Mantenha objetividade — parecer opina, não decide
5. Fecho: "É o parecer, s.m.j." (salvo melhor juízo)
6. Use a tool formatar_data_extenso()"""
        ),
        AssistantMessage(
            f"Vou redigir o parecer {area} com a estrutura técnica adequada. "
            "Consultando template e normas..."
        ),
    ]


def redator_nota_tecnica(
    assunto: str,
    dados: str = "",
) -> list:
    """Redator de Nota Técnica — análise técnica com dados e evidências.

    Use este prompt para redigir notas técnicas que fundamentam
    decisões com dados, análises e recomendações.

    Args:
        assunto: Tema da nota técnica.
        dados: Dados ou fontes a serem consultadas (opcional).
    """
    return [
        UserMessage(
            f"""Preciso redigir uma NOTA TÉCNICA.

Assunto: {assunto}
Dados/fontes disponíveis: {dados if dados else 'Consultar APIs disponíveis (IBGE, Bacen, etc.)'}

Instruções:
1. Consulte o template de nota técnica (resource template://nota_tecnica)
2. Se houver dados relevantes, use as tools do mcp-brasil para consultar:
   - IBGE: indicadores, população, PIB
   - Bacen: câmbio, Selic, inflação
   - Transparência: contratos, despesas
3. Estrutura:
   - ASSUNTO
   - 1. INTRODUÇÃO: contextualização
   - 2. ANÁLISE: dados e argumentos (com números reais se possível)
   - 3. CONCLUSÃO E RECOMENDAÇÕES
4. Inclua dados concretos sempre que possível
5. Use a tool formatar_data_extenso()"""
        ),
        AssistantMessage(
            "Vou redigir a nota técnica. Posso consultar dados em tempo real do "
            "IBGE, Banco Central e Portal da Transparência para fundamentar a análise. "
            "Vou verificar quais dados estão disponíveis..."
        ),
    ]
```

### 4. Tools — `redator/tools.py`

```python
"""Tools: funções executáveis para geração e validação de documentos.

Estas tools são chamadas pelo LLM durante a redação.
Complementam os prompts e resources.
"""

from datetime import datetime
import locale


MESES = {
    1: "janeiro", 2: "fevereiro", 3: "março", 4: "abril",
    5: "maio", 6: "junho", 7: "julho", 8: "agosto",
    9: "setembro", 10: "outubro", 11: "novembro", 12: "dezembro",
}

TIPOS_DOCUMENTO = [
    "despacho", "memorando", "oficio", "portaria",
    "parecer", "nota_tecnica", "ata",
]


async def formatar_data_extenso(
    cidade: str = "Brasília",
    estado: str = "DF",
) -> str:
    """Formata a data atual no padrão oficial brasileiro por extenso.

    Retorna a data no formato: "Cidade/UF, DD de mês de AAAA"
    Exemplo: "Brasília/DF, 22 de março de 2026"

    Args:
        cidade: Nome da cidade. Default: Brasília.
        estado: Sigla do estado (UF). Default: DF.
    """
    hoje = datetime.now()
    mes = MESES[hoje.month]
    return f"{cidade}/{estado}, {hoje.day} de {mes} de {hoje.year}"


async def gerar_numeracao(
    tipo: str,
    numero: int,
    ano: int | None = None,
    setor: str = "",
) -> str:
    """Gera a numeração oficial de um documento.

    Segue o padrão: TIPO nº NÚMERO/ANO/SETOR

    Args:
        tipo: Tipo do documento (despacho, memorando, oficio, portaria, parecer).
        numero: Número sequencial do documento.
        ano: Ano (default: ano atual).
        setor: Sigla do setor (ex: GAB, COORD-TI). Opcional.

    Returns:
        Numeração formatada (ex: "Mem. nº 42/2026/COORD-TI").
    """
    ano = ano or datetime.now().year
    tipo = tipo.lower().strip()

    prefixos = {
        "despacho": "Despacho",
        "memorando": "Mem.",
        "oficio": "Of.",
        "portaria": "Portaria",
        "parecer": "Parecer",
        "nota_tecnica": "Nota Técnica",
        "ata": "Ata",
    }

    prefixo = prefixos.get(tipo, tipo.capitalize())

    if setor:
        return f"{prefixo} nº {numero}/{ano}/{setor}"
    return f"{prefixo} nº {numero}/{ano}"


async def consultar_pronome_tratamento(cargo: str) -> str:
    """Retorna o pronome de tratamento correto para um cargo público.

    Consulta a tabela oficial de pronomes de tratamento do
    Manual de Redação da Presidência da República.

    Args:
        cargo: Cargo do destinatário (ex: "Governador", "Secretário", "Juiz").

    Returns:
        Pronome de tratamento, vocativo e abreviatura.
    """
    # Tabela baseada no Manual de Redação da Presidência
    pronomes = {
        # Presidente e Vice
        "presidente": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Presidente da República",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "vice-presidente": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Vice-Presidente da República",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        # Judiciário
        "ministro stf": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Ministro",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "juiz": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Meritíssimo Senhor Juiz",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "desembargador": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Desembargador",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        # Legislativo
        "senador": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Senador",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "deputado": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Deputado",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "vereador": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Vereador",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        # Executivo estadual/municipal
        "governador": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Governador",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "prefeito": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Prefeito",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "secretário": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Excelentíssimo Senhor Secretário",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        # Militares
        "general": {
            "tratamento": "Vossa Excelência",
            "vocativo": "Senhor General",
            "abreviatura": "V.Exa.",
            "envelope": "A Sua Excelência o Senhor",
        },
        "coronel": {
            "tratamento": "Vossa Senhoria",
            "vocativo": "Senhor Coronel",
            "abreviatura": "V.Sa.",
            "envelope": "Ao Senhor",
        },
        # Religiosos
        "papa": {
            "tratamento": "Vossa Santidade",
            "vocativo": "Santíssimo Padre",
            "abreviatura": "V.S.",
            "envelope": "A Sua Santidade o Papa",
        },
        "cardeal": {
            "tratamento": "Vossa Eminência",
            "vocativo": "Eminentíssimo Senhor Cardeal",
            "abreviatura": "V.Ema.",
            "envelope": "A Sua Eminência o Senhor Cardeal",
        },
        "bispo": {
            "tratamento": "Vossa Excelência Reverendíssima",
            "vocativo": "Excelentíssimo e Reverendíssimo Senhor Bispo",
            "abreviatura": "V.Exa.Revma.",
            "envelope": "A Sua Excelência Reverendíssima",
        },
        # Acadêmicos
        "reitor": {
            "tratamento": "Vossa Magnificência",
            "vocativo": "Magnífico Reitor",
            "abreviatura": "V.Maga.",
            "envelope": "Ao Magnífico Senhor Reitor",
        },
        # Default - demais autoridades
        "diretor": {
            "tratamento": "Vossa Senhoria",
            "vocativo": "Senhor Diretor",
            "abreviatura": "V.Sa.",
            "envelope": "Ao Senhor",
        },
        "coordenador": {
            "tratamento": "Vossa Senhoria",
            "vocativo": "Senhor Coordenador",
            "abreviatura": "V.Sa.",
            "envelope": "Ao Senhor",
        },
        "chefe": {
            "tratamento": "Vossa Senhoria",
            "vocativo": "Senhor Chefe",
            "abreviatura": "V.Sa.",
            "envelope": "Ao Senhor",
        },
    }

    cargo_lower = cargo.lower().strip()

    # Busca exata
    if cargo_lower in pronomes:
        p = pronomes[cargo_lower]
        return (
            f"Cargo: {cargo}\n"
            f"Tratamento: {p['tratamento']}\n"
            f"Vocativo: {p['vocativo']}\n"
            f"Abreviatura: {p['abreviatura']}\n"
            f"Endereçamento (envelope): {p['envelope']}"
        )

    # Busca parcial
    for key, p in pronomes.items():
        if key in cargo_lower or cargo_lower in key:
            return (
                f"Cargo: {cargo} (similar a: {key})\n"
                f"Tratamento: {p['tratamento']}\n"
                f"Vocativo: {p['vocativo']}\n"
                f"Abreviatura: {p['abreviatura']}\n"
                f"Endereçamento (envelope): {p['envelope']}"
            )

    # Default
    return (
        f"Cargo: {cargo}\n"
        f"Tratamento: Vossa Senhoria (padrão para demais autoridades)\n"
        f"Vocativo: Senhor(a) {cargo}\n"
        f"Abreviatura: V.Sa.\n"
        f"Endereçamento: Ao Senhor(a)"
    )


async def validar_documento(texto: str, tipo: str) -> str:
    """Valida se um documento segue as normas de redação oficial.

    Verifica aspectos formais como: presença de data, numeração,
    fecho adequado, uso de pronomes, estrutura obrigatória.

    Args:
        texto: Texto do documento para validar.
        tipo: Tipo do documento (despacho, memorando, oficio, etc.).

    Returns:
        Relatório de validação com problemas encontrados e sugestões.
    """
    problemas = []
    sugestoes = []

    # Verifica data
    meses_lista = list(MESES.values())
    tem_data = any(mes in texto.lower() for mes in meses_lista)
    if not tem_data:
        problemas.append("❌ Sem data por extenso no documento")

    # Verifica fecho
    fechos_validos = [
        "atenciosamente", "respeitosamente",
        "é o parecer", "s.m.j.",
    ]
    tem_fecho = any(fecho in texto.lower() for fecho in fechos_validos)
    if not tem_fecho and tipo not in ("portaria", "ata"):
        problemas.append("❌ Sem fecho oficial (Atenciosamente/Respeitosamente)")

    # Verifica numeração
    tem_numeracao = "nº" in texto or "n°" in texto
    if not tem_numeracao and tipo != "despacho":
        sugestoes.append("⚠️ Sem numeração (recomendado: Tipo nº X/ANO/SETOR)")

    # Verifica gerúndio excessivo (anti-pattern em redação oficial)
    import re
    gerundios = re.findall(r"\b\w+ndo\b", texto)
    if len(gerundios) > 5:
        sugestoes.append(
            f"⚠️ {len(gerundios)} gerúndios encontrados — "
            "redação oficial prefere formas diretas"
        )

    # Verifica parágrafos muito longos
    paragrafos = [p for p in texto.split("\n\n") if p.strip()]
    longos = [p for p in paragrafos if len(p) > 500]
    if longos:
        sugestoes.append(
            f"⚠️ {len(longos)} parágrafo(s) com mais de 500 caracteres — "
            "considere dividir para clareza"
        )

    # Monta relatório
    if not problemas and not sugestoes:
        return "✅ Documento segue as normas de redação oficial. Nenhum problema encontrado."

    relatorio = "📋 RELATÓRIO DE VALIDAÇÃO\n\n"
    if problemas:
        relatorio += "Problemas encontrados:\n"
        relatorio += "\n".join(f"  {p}" for p in problemas)
        relatorio += "\n\n"
    if sugestoes:
        relatorio += "Sugestões de melhoria:\n"
        relatorio += "\n".join(f"  {s}" for s in sugestoes)

    return relatorio


async def listar_tipos_documento() -> str:
    """Lista todos os tipos de documento oficial suportados.

    Returns:
        Lista formatada dos tipos com descrição breve.
    """
    tipos = {
        "despacho": "Decisão administrativa sobre processo ou requerimento",
        "memorando": "Comunicação interna entre setores do mesmo órgão",
        "oficio": "Comunicação externa entre órgãos ou com entidades",
        "portaria": "Ato normativo de autoridade administrativa",
        "parecer": "Manifestação técnica ou jurídica sobre consulta",
        "nota_tecnica": "Análise técnica com dados e recomendações",
        "ata": "Registro de reunião ou assembleia",
    }

    linhas = ["📄 Tipos de documento oficial suportados:\n"]
    for tipo, desc in tipos.items():
        linhas.append(f"  • {tipo}: {desc}")

    linhas.append(
        "\nUse os prompts do redator para gerar cada tipo. "
        "Exemplo: selecione 'Redator de Despacho' no menu de prompts."
    )
    return "\n".join(linhas)
```

### 5. Server — `redator/server.py`

```python
"""Server do Redator Oficial.

Registra tools, prompts e resources para geração
de documentos oficiais no padrão brasileiro.
"""

from fastmcp import FastMCP

from .tools import (
    consultar_pronome_tratamento,
    formatar_data_extenso,
    gerar_numeracao,
    listar_tipos_documento,
    validar_documento,
)
from .prompts import (
    redator_despacho,
    redator_memorando,
    redator_nota_tecnica,
    redator_oficio,
    redator_parecer,
    redator_portaria,
)
from .resources import (
    get_fechos_oficiais,
    get_manual_redacao,
    get_pronomes_tratamento,
    get_template_ata,
    get_template_despacho,
    get_template_memorando,
    get_template_nota_tecnica,
    get_template_oficio,
    get_template_parecer,
    get_template_portaria,
)

mcp = FastMCP("mcp-brasil-redator")

# === Tools ===
mcp.tool(formatar_data_extenso)
mcp.tool(gerar_numeracao)
mcp.tool(consultar_pronome_tratamento)
mcp.tool(validar_documento)
mcp.tool(listar_tipos_documento)

# === Prompts (agentes especializados) ===
mcp.prompt(redator_despacho)
mcp.prompt(redator_memorando)
mcp.prompt(redator_oficio)
mcp.prompt(redator_portaria)
mcp.prompt(redator_parecer)
mcp.prompt(redator_nota_tecnica)

# === Resources (templates e normas) ===
mcp.resource("template://despacho")(get_template_despacho)
mcp.resource("template://memorando")(get_template_memorando)
mcp.resource("template://oficio")(get_template_oficio)
mcp.resource("template://portaria")(get_template_portaria)
mcp.resource("template://parecer")(get_template_parecer)
mcp.resource("template://nota_tecnica")(get_template_nota_tecnica)
mcp.resource("template://ata")(get_template_ata)
mcp.resource("normas://manual_redacao")(get_manual_redacao)
mcp.resource("normas://pronomes")(get_pronomes_tratamento)
mcp.resource("normas://fechos")(get_fechos_oficiais)
```

### 6. Template exemplo — `redator/templates/despacho.md`

```markdown
# MODELO: DESPACHO

## Estrutura Obrigatória

```
DESPACHO

Ref.: [Processo/Expediente nº ___]

[Texto do despacho — fundamentação e decisão]

[Parágrafo 1: Referência ao processo ou documento que originou o despacho]
[Parágrafo 2: Fundamentação — base legal ou justificativa]
[Parágrafo 3: Decisão — o que se decide/determina/encaminha]
[Parágrafo 4 (opcional): Providências adicionais]

[Cidade/UF], [data por extenso].

[Nome da autoridade]
[Cargo]
```

## Regras

1. O despacho é DECISÓRIO — deve conter uma determinação clara
2. Referência ao processo é OBRIGATÓRIA
3. Linguagem impessoal e direta
4. Evitar gerúndio: "Encaminho" (não "Encaminhando")
5. Verbos comuns: aprovo, indefiro, encaminho, determino, autorizo
6. Parágrafos curtos (3-5 linhas máximo)
7. Não usar "vimos por meio deste" — usar forma direta

## Exemplo

DESPACHO

Ref.: Processo nº 23456.789012/2026-01

Trata-se de requerimento de licença para capacitação
formulado pelo servidor João da Silva, matrícula nº 123456,
lotado na Coordenação de Tecnologia da Informação.

Analisados os documentos acostados aos autos, verifico que
o pedido atende aos requisitos do art. 87 da Lei nº 8.112/1990,
com comprovação de matrícula em curso de pós-graduação na
área de atuação do servidor.

DEFIRO o pedido de licença para capacitação pelo período
de 3 (três) meses, de 1º de abril a 30 de junho de 2026.

Encaminhe-se à Coordenação de Gestão de Pessoas para
as providências cabíveis.

Teresina/PI, 22 de março de 2026.

FULANO DE TAL
Diretor de Administração
```

---

## Como o usuário experimenta isso

### No Claude Desktop

```
1. Usuário abre Claude Desktop com mcp-brasil configurado
2. Clica no ícone de prompts (ou digita /)
3. Vê os prompts disponíveis:
   📄 Redator de Despacho
   📄 Redator de Memorando
   📄 Redator de Ofício
   📄 Redator de Portaria
   📄 Redator de Parecer
   📄 Redator de Nota Técnica
4. Seleciona "Redator de Despacho"
5. Preenche os campos:
   - Assunto: "Aprovar licença capacitação servidor João Silva"
   - Contexto: "Processo 23456/2026, 3 meses, pós-graduação em TI"
6. Claude gera o despacho completo seguindo todas as normas
```

### No Cursor / VS Code

```
1. Dev está escrevendo código de um sistema govtech
2. Precisa gerar um template de despacho para o sistema
3. Usa: "Use o prompt redator_despacho para gerar um exemplo"
4. Claude gera o despacho e o dev integra no sistema
```

### Via API / Agent

```python
# Um agent pode usar as tools programaticamente
result = await client.call_tool("formatar_data_extenso", {
    "cidade": "Teresina",
    "estado": "PI"
})
# → "Teresina/PI, 22 de março de 2026"

result = await client.call_tool("consultar_pronome_tratamento", {
    "cargo": "Governador"
})
# → Tratamento: Vossa Excelência
#   Vocativo: Excelentíssimo Senhor Governador
```

---

## O poder da composição: Redator + Dados

O diferencial único do `mcp-brasil` é que o redator pode **consultar dados reais** durante a redação:

```
Usuário: "Redija uma nota técnica sobre a evolução do PIB do Piauí"

Claude:
  1. Usa prompt redator_nota_tecnica
  2. Chama tool /ibge/consultar_pib(uf="PI")  → dados reais
  3. Chama tool /bacen/consultar_selic()        → taxa atual
  4. Chama tool /bacen/consultar_ipca()         → inflação atual
  5. Gera nota técnica com dados reais, não inventados
```

Nenhum outro MCP server faz isso — combinar redação oficial com dados governamentais em tempo real.

---

## Evolução futura

### v0.2 — Exportação DOCX
- Tool que gera arquivo .docx formatado com cabeçalho, numeração e brasão
- Integração com python-docx para formatação profissional

### v0.3 — Templates por órgão
- Cada órgão tem seu padrão (cabeçalho, logomarca, numeração)
- Resource dinâmico: `template://despacho?orgao=ETIPI`
- Contribuidores podem adicionar templates de seus órgãos

### v0.4 — Workflow completo
- Prompt "Fluxo de Documentos": gera despacho + memorando + ofício em sequência
- Integração com protocolo digital (SEI, e-Docs)

---

## Referências

- [Manual de Redação da Presidência da República](http://www.planalto.gov.br/ccivil_03/manual/manual.htm)
- [FastMCP Prompts](https://gofastmcp.com/servers/prompts)
- [FastMCP Resources](https://gofastmcp.com/servers/resources)
- [MCP Spec — Prompts](https://modelcontextprotocol.io/docs/concepts/prompts)
- [MCP Spec — Sampling](https://modelcontextprotocol.io/docs/concepts/sampling)
