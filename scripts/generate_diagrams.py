#!/usr/bin/env python3
"""Generate architecture diagrams for mcp-brasil documentation.

Produces 4 PNG diagrams in docs/concepts/img/:
  - system_overview.png
  - feature_anatomy.png
  - auto_registry_flow.png
  - data_flow.png

Requirements: graphviz (brew install graphviz), diagrams (pip install diagrams)
Usage: python scripts/generate_diagrams.py
"""

from __future__ import annotations

import os
from pathlib import Path

from diagrams import Cluster, Diagram, Edge
from diagrams.generic.blank import Blank
from diagrams.onprem.client import Client
from diagrams.onprem.compute import Server
from diagrams.programming.flowchart import Action, Decision, StartEnd
from diagrams.programming.framework import FastAPI
from diagrams.programming.language import Python

OUTPUT_DIR = Path(__file__).resolve().parent.parent / "docs" / "concepts" / "img"
GRAPH_ATTR = {"fontsize": "14", "bgcolor": "white", "pad": "0.5"}
NODE_ATTR = {"fontsize": "11"}
EDGE_ATTR = {"fontsize": "10"}


def system_overview() -> None:
    """Diagram 1: High-level system architecture."""
    with Diagram(
        "mcp-brasil — System Overview",
        filename=str(OUTPUT_DIR / "system_overview"),
        direction="TB",
        show=False,
        graph_attr=GRAPH_ATTR,
        node_attr=NODE_ATTR,
        edge_attr=EDGE_ATTR,
    ):
        client = Client("MCP Client\n(Claude, GPT, ...)")

        with Cluster("mcp-brasil Root Server"):
            root = FastAPI("FastMCP\nserver.py")
            registry = Python("FeatureRegistry")
            meta = Python("Meta-Tools\n(listar, recomendar,\nplanejar, lote)")
            root - registry
            root - meta

        client >> root

        with Cluster("Econômico"):
            bacen = Python("bacen")
            ibge = Python("ibge")
            Python("transparência")
            Python("transferegov")

        with Cluster("Legislativo"):
            camara = Python("câmara")
            Python("senado")

        with Cluster("Judiciário"):
            datajud = Python("datajud")
            Python("jurisprudência")

        with Cluster("Eleitoral"):
            tse = Python("tse")

        with Cluster("Fiscalização"):
            tcu = Python("tcu")
            Python("TCEs (9)")

        with Cluster("Ambiental & Saúde"):
            inpe = Python("inpe")
            Python("ana")
            Python("saúde")

        with Cluster("Outros"):
            compras = Python("compras")
            Python("brasilapi")
            Python("dados_abertos")
            Python("diário_oficial")

        with Cluster("Agentes"):
            redator = Python("redator")

        registry >> Edge(style="dashed") >> bacen
        registry >> Edge(style="dashed") >> camara
        registry >> Edge(style="dashed") >> datajud
        registry >> Edge(style="dashed") >> tse
        registry >> Edge(style="dashed") >> tcu
        registry >> Edge(style="dashed") >> inpe
        registry >> Edge(style="dashed") >> compras
        registry >> Edge(style="dashed") >> redator

        apis = Server("APIs Governamentais\n(gov.br, ibge.gov.br,\nbcb.gov.br, ...)")

        bacen >> apis
        ibge >> apis
        camara >> apis
        datajud >> apis
        tse >> apis
        tcu >> apis
        inpe >> apis
        compras >> apis


def feature_anatomy() -> None:
    """Diagram 2: Internal structure of a feature package."""
    with Diagram(
        "Anatomia de uma Feature (data/ibge/)",
        filename=str(OUTPUT_DIR / "feature_anatomy"),
        direction="LR",
        show=False,
        graph_attr=GRAPH_ATTR,
        node_attr=NODE_ATTR,
        edge_attr=EDGE_ATTR,
    ):
        with Cluster("data/ibge/"):
            Python("__init__.py\nFEATURE_META")
            server = FastAPI("server.py\nmcp: FastMCP")
            tools = Python("tools.py\nbuscar_localidades()\nconsultar_populacao()")
            client = Python("client.py\nhttp_get() async")
            schemas = Python("schemas.py\nBaseModel")
            Python("constants.py\nIBGE_API_BASE")

            server >> Edge(label="registra") >> tools
            tools >> Edge(label="delega HTTP") >> client
            client >> Edge(label="retorna") >> schemas

        shared = Python("_shared/\nhttp_client\ncache\nformatting")
        api = Server("API IBGE\nibge.gov.br")

        tools >> Edge(style="dashed", label="usa") >> shared
        client >> Edge(style="dashed", label="usa") >> shared
        client >> api


def auto_registry_flow() -> None:
    """Diagram 3: Auto-registry discovery flowchart."""
    with Diagram(
        "Auto-Registry — Fluxo de Discovery",
        filename=str(OUTPUT_DIR / "auto_registry_flow"),
        direction="TB",
        show=False,
        graph_attr=GRAPH_ATTR,
        node_attr=NODE_ATTR,
        edge_attr=EDGE_ATTR,
    ):
        start = StartEnd("discover(pkg)")
        iter_mod = Action("iter_modules(pkg)")
        skip_check = Decision("nome começa\ncom '_'?")
        skip = Action("pular")
        import_init = Action("import __init__.py")
        meta_check = Decision("FEATURE_META\nexiste?")
        skip2 = Action("pular")
        auth_check = Decision("requires_auth\ne env var OK?")
        skip3 = Action("pular\n(silencioso)")
        import_server = Action("import server.py")
        mount = Action("mount(mcp,\nnamespace=name)")
        end = StartEnd("próximo módulo\nou fim")

        start >> iter_mod >> skip_check
        skip_check >> Edge(label="sim") >> skip >> end
        skip_check >> Edge(label="não") >> import_init >> meta_check
        meta_check >> Edge(label="não") >> skip2 >> end
        meta_check >> Edge(label="sim") >> auth_check
        auth_check >> Edge(label="não") >> skip3 >> end
        auth_check >> Edge(label="sim") >> import_server >> mount >> end


def data_flow() -> None:
    """Diagram 4: Request/response data flow pipeline."""
    with Diagram(
        "Fluxo de Dados — Request & Response",
        filename=str(OUTPUT_DIR / "data_flow"),
        direction="LR",
        show=False,
        graph_attr={**GRAPH_ATTR, "nodesep": "0.8"},
        node_attr=NODE_ATTR,
        edge_attr=EDGE_ATTR,
    ):
        user = Client("Usuário")
        mcp_client = Client("MCP Client")
        bm25 = Python("BM25 Filter\n(top-10 tools)")

        with Cluster("mcp-brasil"):
            tools = Python("tools.py\norquestra")
            client = Python("client.py\nhttpx async")
            rate = Python("Rate Limiter\nsliding window")

        api = Server("API Gov\n(JSON)")

        # Request path
        user >> Edge(label="pergunta") >> mcp_client
        mcp_client >> Edge(label="tool call") >> bm25
        bm25 >> Edge(label="dispatch") >> tools
        tools >> client >> rate >> api

        # Response path (reverse labels)
        api >> Edge(label="JSON", style="dashed", color="darkgreen") >> client
        client >> Edge(label="Pydantic", style="dashed", color="darkgreen") >> tools
        tools >> Edge(label="Markdown", style="dashed", color="darkgreen") >> mcp_client
        mcp_client >> Edge(label="resposta", style="dashed", color="darkgreen") >> user

        # Retry annotation
        _ = Blank("")
        rate >> Edge(label="retry 429/5xx", style="dotted", color="red") >> rate


def main() -> None:
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    # diagrams lib uses cwd for temp files, so switch to output dir
    original_cwd = os.getcwd()
    os.chdir(OUTPUT_DIR)
    try:
        system_overview()
        feature_anatomy()
        auto_registry_flow()
        data_flow()
    finally:
        os.chdir(original_cwd)

    generated = sorted(OUTPUT_DIR.glob("*.png"))
    print(f"Generated {len(generated)} diagrams in {OUTPUT_DIR}/")
    for p in generated:
        print(f"  {p.name}")


if __name__ == "__main__":
    main()
