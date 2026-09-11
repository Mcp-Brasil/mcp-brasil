"""Constantes da feature CPF.CNPJ.

Este módulo não importa nada interno do projeto (regra ADR-001 rule #3).
"""

# URL base da API. Somente HTTPS: o token da conta viaja no caminho da URL
# (/{token}/{pacote}/{documento}) e não pode trafegar sem criptografia.
CPFCNPJ_BASE = "https://api.cpfcnpj.com.br"

# Pacotes de consulta de CNPJ usados por esta feature.
PACOTE_CADASTRO = 5  # razão, fantasia, endereço, IBGE
PACOTE_COMPLETO = 6  # cadastro + situação, CNAE, natureza, porte, QSA, Simples, regimes
PACOTE_INSCRICOES = 16  # inscrições estaduais por UF

# Teto de requisições por segundo por conta (a API responde erro 1007 ao exceder).
MAX_REQ_POR_SEGUNDO = 20

# Segmento fixo do endpoint gratuito de saldo (GET /{token}/saldo/{pacote}).
SEGMENTO_SALDO = "saldo"

# Pesos do módulo 11 para o dígito verificador do CNPJ (numérico e alfanumérico).
PESOS_CNPJ_DV1 = (5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)
PESOS_CNPJ_DV2 = (6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)

# Mapa erroCodigo -> mensagem amigável (documentação CPF.CNPJ, TAREFAS §4).
# Nenhum destes códigos consome crédito da conta.
MENSAGENS_ERRO = {
    200: "CNPJ inválido ou incompleto.",
    201: "CNPJ inválido ou incompleto.",
    202: "CNPJ não encontrado na Receita Federal.",
    1000: "Token inválido ou IP não autorizado para a conta CPF.CNPJ.",
    1001: "Sem créditos no pacote da conta CPF.CNPJ.",
    1002: "Conta CPF.CNPJ suspensa ou inativa.",
    1003: "IP e token bloqueados até a data informada pela CPF.CNPJ.",
    1004: "Pacote indisponível na conta CPF.CNPJ.",
    1005: "Fornecedor indisponível no momento, tente novamente mais tarde.",
    1006: "Fornecedor indisponível no momento, tente novamente mais tarde.",
    1007: "Limite de 20 requisições por segundo excedido, aguarde 1 segundo.",
}
