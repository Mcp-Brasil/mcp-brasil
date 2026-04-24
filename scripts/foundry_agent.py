#!/usr/bin/env python3
"""Create, test, and delete the mcp-brasil agent on Azure AI Foundry.

Uses the azure-ai-projects v2 SDK with MCPTool to create an agent
that connects to the mcp-brasil MCP server and can be published to Teams.

Usage:
    python scripts/foundry_agent.py create
    python scripts/foundry_agent.py test [--question "..."]
    python scripts/foundry_agent.py delete --name <agent-name>

Environment variables:
    PROJECT_ENDPOINT   — Azure AI Foundry project endpoint
    MODEL_DEPLOYMENT   — Model deployment name (default: gpt-4o)
    MCP_BRASIL_URL     — mcp-brasil server URL (default: production)
    MCP_CONNECTION_ID  — Foundry project connection name for MCP auth (optional)

Prerequisites:
    uv sync --group foundry
    az login
"""

from __future__ import annotations

import argparse
import os
import sys
from typing import NoReturn

DEFAULT_ENDPOINT = "https://agent-brasil-resource.services.ai.azure.com/api/projects/agent-brasil"
DEFAULT_MODEL = "gpt-4o"
DEFAULT_MCP_URL = "https://mcp-brasil.purplepond-78f67485.eastus2.azurecontainerapps.io/mcp"

AGENT_NAME = "agente-brasil"

SYSTEM_PROMPT = """\
Você é o Agente Brasil, um assistente especializado em dados governamentais \
brasileiros. Você tem acesso a 41 APIs públicas via ferramentas MCP.

Regras:
1. Sempre responda em português brasileiro
2. Cite a fonte dos dados (ex: "Fonte: Banco Central do Brasil via API SGS")
3. Para consultas complexas, use planejar_consulta primeiro
4. Para múltiplas consultas simultâneas, use executar_lote
5. Se não souber qual ferramenta usar, use recomendar_tools
6. Formate números com separadores brasileiros (ex: R$ 1.234.567,89)
7. Inclua a data de referência dos dados quando disponível
8. Seja conciso mas completo — usuários estão no Teams, preferem respostas diretas

APIs disponíveis: IBGE, Banco Central, Câmara dos Deputados, Senado Federal, \
Portal da Transparência, DataJud/CNJ, TSE, INPE, ANVISA, CNES/DataSUS, PNCP, \
TCU, TCEs estaduais, BrasilAPI, dados.gov.br, e mais.
"""


def _get_env(name: str, default: str | None = None) -> str:
    value = os.environ.get(name, default)
    if not value:
        print(f"Error: {name} is not set and has no default.", file=sys.stderr)
        sys.exit(1)
    return value


def cmd_create(args: argparse.Namespace) -> None:
    """Create the mcp-brasil agent on Azure AI Foundry."""
    from azure.ai.projects import AIProjectClient
    from azure.ai.projects.models import MCPTool, PromptAgentDefinition
    from azure.identity import DefaultAzureCredential

    endpoint = _get_env("PROJECT_ENDPOINT", DEFAULT_ENDPOINT)
    model = _get_env("MODEL_DEPLOYMENT", DEFAULT_MODEL)
    mcp_url = _get_env("MCP_BRASIL_URL", DEFAULT_MCP_URL)
    connection_id = os.environ.get("MCP_CONNECTION_ID")

    mcp_kwargs: dict[str, object] = {
        "server_label": "mcp-brasil",
        "server_url": mcp_url,
        "require_approval": "never",
    }
    if connection_id:
        mcp_kwargs["project_connection_id"] = connection_id

    mcp_tool = MCPTool(**mcp_kwargs)  # type: ignore[arg-type]

    with (
        DefaultAzureCredential() as credential,
        AIProjectClient(endpoint=endpoint, credential=credential) as client,
    ):
        agent = client.agents.create_version(
            agent_name=AGENT_NAME,
            definition=PromptAgentDefinition(
                model=model,
                instructions=SYSTEM_PROMPT,
                tools=[mcp_tool],
            ),
        )
        print(f"Agent created: name={agent.name}, version={agent.version}, id={agent.id}")


def cmd_test(args: argparse.Namespace) -> None:
    """Send a test question to the agent and print the response."""
    from azure.ai.projects import AIProjectClient
    from azure.identity import DefaultAzureCredential

    endpoint = _get_env("PROJECT_ENDPOINT", DEFAULT_ENDPOINT)
    question = args.question or "Qual é a população atual do Brasil segundo o IBGE?"

    with (
        DefaultAzureCredential() as credential,
        AIProjectClient(endpoint=endpoint, credential=credential) as project_client,
        project_client.get_openai_client() as openai_client,
    ):
        conversation = openai_client.conversations.create()
        print(f"Conversation: {conversation.id}")

        response = openai_client.responses.create(
            conversation=conversation.id,
            input=question,
            extra_body={"agent_reference": {"name": AGENT_NAME, "type": "agent_reference"}},
        )

        # Handle MCP approval requests if agent uses require_approval="always"
        _handle_approvals(openai_client, response)

        print(f"\nQuestion: {question}")
        print(f"Response: {response.output_text}")


def _handle_approvals(openai_client: object, response: object) -> None:
    """Auto-approve any MCP tool approval requests."""
    from openai.types.responses.response_input_param import McpApprovalResponse

    approvals = []
    for item in response.output:  # type: ignore[attr-defined]
        if getattr(item, "type", None) == "mcp_approval_request" and item.id:
            approvals.append(
                McpApprovalResponse(
                    type="mcp_approval_response",
                    approve=True,
                    approval_request_id=item.id,
                )
            )

    if approvals:
        response = openai_client.responses.create(  # type: ignore[union-attr]
            input=approvals,
            previous_response_id=response.id,  # type: ignore[attr-defined]
            extra_body={"agent_reference": {"name": AGENT_NAME, "type": "agent_reference"}},
        )


def cmd_delete(args: argparse.Namespace) -> None:
    """Delete a specific agent version."""
    from azure.ai.projects import AIProjectClient
    from azure.identity import DefaultAzureCredential

    endpoint = _get_env("PROJECT_ENDPOINT", DEFAULT_ENDPOINT)
    name = args.name or AGENT_NAME

    with (
        DefaultAzureCredential() as credential,
        AIProjectClient(endpoint=endpoint, credential=credential) as client,
    ):
        versions = client.agents.list_versions(agent_name=name)
        if not versions.data:
            print(f"No versions found for agent '{name}'.")
            return

        for version in versions.data:
            client.agents.delete_version(agent_name=name, agent_version=version.version)
            print(f"Deleted: {name} v{version.version}")


def main(argv: list[str] | None = None) -> NoReturn:
    parser = argparse.ArgumentParser(
        description="Manage the mcp-brasil agent on Azure AI Foundry.",
    )
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("create", help="Create the agent with MCPTool")

    test_parser = sub.add_parser("test", help="Send a test question to the agent")
    test_parser.add_argument("--question", "-q", help="Custom question to ask")

    delete_parser = sub.add_parser("delete", help="Delete agent versions")
    delete_parser.add_argument("--name", "-n", help=f"Agent name (default: {AGENT_NAME})")

    args = parser.parse_args(argv)

    commands = {"create": cmd_create, "test": cmd_test, "delete": cmd_delete}
    commands[args.command](args)
    sys.exit(0)


if __name__ == "__main__":
    main()
