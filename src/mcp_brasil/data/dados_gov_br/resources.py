"""Static reference data for the dados_gov_br feature."""

from __future__ import annotations

import json


def formatos_disponiveis() -> str:
    """Formatos de arquivo comuns no Portal Dados Abertos."""
    data = [
        {"formato": "CSV", "descricao": "Valores separados por vírgula"},
        {"formato": "JSON", "descricao": "JavaScript Object Notation"},
        {"formato": "XML", "descricao": "Extensible Markup Language"},
        {"formato": "XLS/XLSX", "descricao": "Planilha Microsoft Excel"},
        {"formato": "ODS", "descricao": "Open Document Spreadsheet"},
        {"formato": "PDF", "descricao": "Documento portátil"},
        {"formato": "SHP", "descricao": "Shapefile geográfico"},
        {"formato": "GeoJSON", "descricao": "Dados geográficos em JSON"},
        {"formato": "API", "descricao": "Interface programática"},
    ]
    return json.dumps(data, ensure_ascii=False, indent=2)


def documentacao() -> str:
    """Documentação e links úteis da API do Portal Dados Abertos."""
    return (
        "# API do Portal Brasileiro de Dados Abertos\n\n"
        "- Portal: https://dados.gov.br\n"
        "- Swagger UI: https://dados.gov.br/swagger-ui/index.html\n"
        "- OpenAPI Spec: https://dados.gov.br/v3/api-docs\n"
        "- Catálogo Conecta: https://www.gov.br/conecta/catalogo/apis/"
        "api-portal-de-dados-abertos\n\n"
        "## Autenticação\n\n"
        "A API requer chave de acesso via header `chave-api-dados-abertos`.\n"
        "Para obter a chave:\n"
        "1. Acesse dados.gov.br com conta Gov.br (nível prata ou ouro)\n"
        "2. Vá em 'Minha Conta' e gere o token\n"
        "3. Configure a variável de ambiente `DADOS_GOV_BR_API_KEY`\n\n"
        "## Suporte\n\n"
        "- Negócio: suporte.dadosabertos@cgu.gov.br — (61) 2020-6538\n"
        "- Técnico: listadadosabertos@cgu.gov.br — (61) 2020-6760\n"
    )


def legislacao() -> str:
    """Base legal do Portal de Dados Abertos."""
    return (
        "# Legislação de Dados Abertos\n\n"
        "- **Lei 12.527/2011** — Lei de Acesso à Informação (LAI)\n"
        "- **Decreto 8.777/2016** — Política de Dados Abertos da "
        "Administração Federal\n"
        "- **Lei 13.709/2018** — LGPD (Lei Geral de Proteção de Dados)\n"
        "- **Decreto 10.332/2020** — Estratégia de Governo Digital 2020-2022\n"
        "- **Lei 14.129/2021** — Governo Digital\n"
    )
