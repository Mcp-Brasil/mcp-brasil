"""Validação e normalização local de CPF e CNPJ (sem rede, sem crédito).

Aceita CNPJ numérico e alfanumérico (IN RFB 2.229/2024): 12 posições
alfanuméricas (0-9, A-Z) mais 2 dígitos verificadores numéricos, com módulo 11
sobre ``ord(caractere) - 48``. Espaços e quebras de linha NAS BORDAS são
tolerados (removidos com ``strip()`` antes de validar); o valor bruto do
interior é validado antes de normalizar, então espaço INTERNO e caracteres
estranhos tornam o documento inválido.
"""

from __future__ import annotations

from .constants import PESOS_CNPJ_DV1, PESOS_CNPJ_DV2


def _apenas_digitos(valor: str) -> str:
    return "".join(c for c in valor if c.isdigit())


def _sem_separadores(valor: str) -> str:
    """Remove apenas a pontuação padrão (. / -), preservando letras e espaços."""
    return valor.replace(".", "").replace("/", "").replace("-", "")


def _dv_mod11(vals: list[int], pesos: tuple[int, ...]) -> int:
    soma = sum(v * p for v, p in zip(vals, pesos, strict=False))
    resto = soma % 11
    return 0 if resto < 2 else 11 - resto


def normalizar_cpf(cpf: str) -> str:
    """Normaliza um CPF para conter apenas dígitos (bordas toleradas com ``strip``)."""
    return _apenas_digitos(cpf.strip())


def normalizar_cnpj(cnpj: str) -> str:
    """Normaliza um CNPJ preservando letras (isalnum, nunca ``\\D``) em maiúsculas.

    Espaços e quebras de linha nas bordas são tolerados (``strip``).
    """
    return "".join(c for c in cnpj.strip() if c.isalnum()).upper()


def validar_cpf(cpf: str) -> bool:
    """Valida um CPF a partir do valor bruto.

    Espaços e quebras de linha nas bordas são tolerados (``strip``); espaço
    interno e caracteres estranhos tornam o documento inválido.
    """
    cpf = cpf.strip()
    if not cpf:
        return False
    digitos = _apenas_digitos(cpf)
    if len(digitos) != 11 or digitos != _sem_separadores(cpf):
        return False
    if digitos == digitos[0] * 11:
        return False
    n = [int(c) for c in digitos]
    d1 = _dv_mod11(n[:9], tuple(range(10, 1, -1)))
    d2 = _dv_mod11([*n[:9], d1], tuple(range(11, 1, -1)))
    return d1 == n[9] and d2 == n[10]


def _validar_cnpj_numerico(digitos: str) -> bool:
    if digitos == digitos[0] * 14:
        return False
    vals = [int(c) for c in digitos]
    d1 = _dv_mod11(vals[:12], PESOS_CNPJ_DV1)
    d2 = _dv_mod11([*vals[:12], d1], PESOS_CNPJ_DV2)
    return d1 == vals[12] and d2 == vals[13]


def _validar_cnpj_alfanumerico(cnpj: str) -> bool:
    # cnpj: 14 posições A-Z/0-9, já em maiúsculas; DV sempre numérico.
    if len(set(cnpj)) == 1 or not cnpj[12:].isdigit():
        return False
    vals = [ord(c) - 48 for c in cnpj]
    d1 = _dv_mod11(vals[:12], PESOS_CNPJ_DV1)
    d2 = _dv_mod11([*vals[:12], d1], PESOS_CNPJ_DV2)
    return d1 == int(cnpj[12]) and d2 == int(cnpj[13])


def validar_cnpj(cnpj: str) -> bool:
    """Valida um CNPJ numérico ou alfanumérico a partir do valor bruto.

    Espaços e quebras de linha nas bordas são tolerados (``strip``); espaço
    interno e caracteres fora de ``[0-9A-Za-z]`` mais a pontuação padrão
    tornam o documento inválido.
    """
    cnpj = cnpj.strip()
    if not cnpj:
        return False
    digitos = _apenas_digitos(cnpj)
    sem_sep = _sem_separadores(cnpj)
    # Numérico: 14 dígitos e nada além da pontuação padrão foi removido.
    if len(digitos) == 14 and digitos == sem_sep:
        return _validar_cnpj_numerico(digitos)
    cnpj_sem_mascara = sem_sep.upper()
    if len(cnpj_sem_mascara) == 14 and cnpj_sem_mascara.isalnum() and cnpj_sem_mascara.isascii():
        return _validar_cnpj_alfanumerico(cnpj_sem_mascara)
    return False


def classificar_documento(documento: str) -> str:
    """Classifica o documento como ``cpf``, ``cnpj`` ou ``desconhecido``.

    Espaços e quebras de linha nas bordas são tolerados (``strip``).
    """
    documento = documento.strip()
    if not documento:
        return "desconhecido"
    digitos = _apenas_digitos(documento)
    if len(digitos) == 11 and digitos == _sem_separadores(documento):
        return "cpf"
    if len(normalizar_cnpj(documento)) == 14:
        return "cnpj"
    return "desconhecido"
