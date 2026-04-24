"""Constants for BCB Olinda APIs."""

from __future__ import annotations

PTAX_BASE = "https://olinda.bcb.gov.br/olinda/servico/PTAX/versao/v1/odata"
EXPECT_BASE = "https://olinda.bcb.gov.br/olinda/servico/Expectativas/versao/v1/odata"
TAXA_JUROS_BASE = "https://olinda.bcb.gov.br/olinda/servico/taxaJuros/versao/v2/odata"

# Indicadores suportados na Expectativas Focus
INDICADORES_FOCUS: dict[str, str] = {
    "IPCA": "IPCA — inflação oficial",
    "IGP-M": "IGP-M — Índice Geral de Preços do Mercado",
    "IGP-DI": "IGP-DI — Índice Geral de Preços - Disponibilidade Interna",
    "IPA-DI": "IPA-DI — Índice de Preços ao Produtor Amplo",
    "INPC": "INPC — Índice Nacional de Preços ao Consumidor",
    "INCC": "INCC — Índice Nacional da Construção Civil",
    "Selic": "Selic — Taxa básica de juros",
    "Meta para taxa over-selic": "Meta Selic definida pelo Copom",
    "PIB Total": "PIB Total — crescimento",
    "PIB Agropecuária": "PIB Agropecuária — crescimento",
    "PIB Indústria": "PIB Indústria — crescimento",
    "PIB Serviços": "PIB Serviços — crescimento",
    "Produção industrial": "Produção industrial — variação",
    "Câmbio": "Câmbio R$/US$",
    "Conta corrente": "Conta corrente — saldo (USD bi)",
    "Balança comercial": "Balança comercial — saldo (USD bi)",
    "Investimento direto no país": "IED — investimento direto no país (USD bi)",
    "Dívida líquida do setor público": "DLSP — % do PIB",
    "Resultado primário": "Resultado primário — % do PIB",
    "Resultado nominal": "Resultado nominal — % do PIB",
}
