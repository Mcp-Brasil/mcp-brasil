"""Resources for the DataJud feature — static reference data for LLM context."""

from __future__ import annotations

import json

from .constants import (
    CLASSES_PROCESSUAIS,
    DATAJUD_API_BASE,
    MPU_ASSUNTOS,
    MPU_CLASSES_NOMES,
    MPU_COMPLEMENTO_DESTINATARIO,
    MPU_COMPLEMENTO_TIPO_MEDIDA,
    MPU_DESTINATARIOS,
    MPU_MOV_NOMES,
    MPU_MOV_TODOS_LEGADOS,
    MPU_MOV_TODOS_NOVOS,
    MPU_TIPOS_MEDIDA,
    TRIBUNAIS,
    TRIBUNAL_NOMES,
)


def tribunais_disponiveis() -> str:
    """Lista de tribunais disponíveis na API DataJud com siglas e nomes."""
    data = [
        {"sigla": sigla, "nome": TRIBUNAL_NOMES.get(sigla, sigla.upper())}
        for sigla in sorted(TRIBUNAIS.keys())
    ]
    return json.dumps(data, ensure_ascii=False)


def classes_processuais() -> str:
    """Classes processuais comuns para busca no DataJud."""
    return json.dumps(CLASSES_PROCESSUAIS, ensure_ascii=False)


def info_api() -> str:
    """Informações gerais sobre a API DataJud (CNJ)."""
    data = {
        "nome": "API Pública DataJud — Conselho Nacional de Justiça",
        "url_base": DATAJUD_API_BASE,
        "autenticacao": "Requer API Key (cadastro em datajud.cnj.jus.br)",
        "formato": "Elasticsearch (POST com body JSON)",
        "documentacao": "https://datajud-wiki.cnj.jus.br/api-publica/",
        "cobertura": "Processos de todos os tribunais brasileiros",
        "total_tribunais": len(TRIBUNAIS),
    }
    return json.dumps(data, ensure_ascii=False)


# --- MPU Resources ---


def mpu_classes() -> str:
    """Classes processuais de Medidas Protetivas de Urgência (MPU).

    Inclui classes da Lei Maria da Penha (11.340/2006) e Lei Henry Borel (14.344/2022).
    Use os códigos para filtrar buscas no DataJud.
    """
    data = [{"codigo": codigo, "nome": nome} for codigo, nome in MPU_CLASSES_NOMES.items()]
    return json.dumps(data, ensure_ascii=False)


def mpu_movimentos() -> str:
    """Movimentos processuais de MPU (concessão, revogação, prorrogação).

    Inclui códigos novos (atualizados em Nov/2024) e legados.
    As tools de MPU aceitam ambos automaticamente.
    """
    novos = [
        {"codigo": cod, "nome": MPU_MOV_NOMES[cod], "tipo": "novo"} for cod in MPU_MOV_TODOS_NOVOS
    ]
    legados = [
        {"codigo": cod, "nome": MPU_MOV_NOMES[cod], "tipo": "legado"}
        for cod in MPU_MOV_TODOS_LEGADOS
    ]
    return json.dumps(novos + legados, ensure_ascii=False)


def mpu_complementos() -> str:
    """Complementos tabelados de MPU: destinatário e tipo de medida.

    Usados para filtrar buscas por destinatário (mulher, idoso, criança)
    e por tipo de medida (afastamento do lar, proibição de aproximação, etc.).
    """
    data = {
        "destinatario": {
            "codigo_complemento": MPU_COMPLEMENTO_DESTINATARIO,
            "valores": {nome: codigo for nome, codigo in MPU_DESTINATARIOS.items()},
        },
        "tipo_medida": {
            "codigo_complemento": MPU_COMPLEMENTO_TIPO_MEDIDA,
            "valores": {nome: codigo for nome, codigo in MPU_TIPOS_MEDIDA.items()},
        },
        "assuntos": [{"codigo": cod, "nome": nome} for cod, nome in MPU_ASSUNTOS.items()],
    }
    return json.dumps(data, ensure_ascii=False)
