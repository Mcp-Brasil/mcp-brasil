"""Static reference data for the TCE-ES feature."""

from __future__ import annotations

import json


def datasets_tce_es() -> str:
    """Datasets disponíveis do TCE-ES no portal dados.es.gov.br.

    Lista os recursos CKAN utilizados com seus IDs e parâmetros de consulta.
    """
    datasets = [
        {
            "resource_id": "e9541bc2-d5f5-4012-a98e-3c6af3e40cd4",
            "nome": "Licitações - TCEES (Aquisições)",
            "descricao": "Processos licitatórios do próprio Tribunal de Contas do ES",
            "filtros_disponiveis": ["AnoEdital", "Situacao"],
            "campos_principais": [
                "Modalidade",
                "NumeroEdital",
                "AnoEdital",
                "Objeto",
                "DataAbertura",
                "ValorReferencia",
                "ValorHomologado",
                "Situacao",
            ],
        },
        {
            "resource_id": "eeefc286-8d7c-47ec-a38e-3b9f1253e35a",
            "nome": "Contratos - TCEES (Aquisições)",
            "descricao": "Contratos celebrados pelo Tribunal de Contas do ES",
            "filtros_disponiveis": ["ContratoAno"],
            "campos_principais": [
                "ContratoNumero",
                "ContratoAno",
                "Modalidade",
                "ResumoObjeto",
                "FornecedorNome",
                "TermoOriginalValorGlobal",
                "VigenciaAtualValorGlobal",
                "Setor",
            ],
        },
        {
            "resource_id": "bdc86561-cb94-4da9-9131-42ebe5d6c5ac",
            "nome": "Contratações - Controle Externo (municípios e órgãos do ES)",
            "descricao": (
                "127 mil contratações de municípios e órgãos capixabas monitorados pelo TCE-ES"
            ),
            "filtros_disponiveis": ["AnoReferencia", "NomeEsferaAdministrativa"],
            "campos_principais": [
                "NomeUnidadeGestoraReferencia",
                "NomeEsferaAdministrativa",
                "ObjetoContratacao",
                "ModalidadeLicitacao",
                "ValorEstimado",
                "ValorTotalContratacao",
                "AnoReferencia",
                "SituacaoContratacao",
            ],
        },
        {
            "resource_id": "f5fb83a1-361d-4169-999b-b7d65d81689a",
            "nome": "Obras Públicas (Geo Obras)",
            "descricao": "Obras públicas cadastradas no TCE-ES com dados georreferenciados",
            "filtros_disponiveis": ["Situacao"],
            "campos_principais": [
                "Licitacao",
                "Contrato",
                "DataAssinaturaContrato",
                "Empresa",
                "EmpresaCNPJ",
                "ValorInicial",
                "Situacao",
            ],
        },
    ]
    return json.dumps(datasets, ensure_ascii=False, indent=2)
