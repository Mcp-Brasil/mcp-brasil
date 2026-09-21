"""Constants for the NucleoGov feature."""

#: NucleoGov powers the transparency portal of most municipalities in Goiás and
#: several other states. Portals usually live at
#: ``acessoainformacao.<municipio>.<uf>.gov.br`` and the municipality domain is
#: a parameter of every tool — there is no single API base.
API_BASE = ""

#: The portal's WAF rejects requests that identify as a program, so requests
#: carry a browser User-Agent. This is not a credential: the data is public
#: under the Access to Information Law (Lei 12.527/2011).
BROWSER_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

DEFAULT_TIMEOUT = 60.0
DEFAULT_LIMITE = 100

#: API actions for the ``_frl`` family (the most common). The suffix varies by
#: portal (``_frl``/``_cnt``); see the module docstring of ``client.py``.
ACAO_DESPESAS = "despesas_frl/listar"
ACAO_CONTRATOS = "contratos_frl/listar"
ACAO_LICITACOES = "licitacoes_frl/listar"

#: Legislation is served per municipality: ``atos_pf_{slug}/listar``.
ACAO_ATOS_TEMPLATE = "atos_pf_{slug}/listar"

#: Free-text filter shared by every action.
CAMPO_BUSCA = "txtbusca"
