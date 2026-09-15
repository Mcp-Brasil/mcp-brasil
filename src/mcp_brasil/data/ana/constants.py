"""Constants for the ANA feature."""

HIDROWEB_API = "https://www.snirh.gov.br/hidroweb/rest/api"
SAR_API = "https://www.ana.gov.br/sar0"

ESTACOES_URL = f"{HIDROWEB_API}/estacao/codigoestacao"
TELEMETRIA_URL = f"{HIDROWEB_API}/estacao/telemetrica"
RESERVATORIOS_URL = f"{SAR_API}/Medicao"

# Optional auth: Hidroweb/SNIRH started returning HTTP 401 on endpoints that
# used to be public (issue #37). When set, the token is sent as a Bearer
# credential; when unset, requests go out anonymously as before.
AUTH_ENV_VAR = "ANA_HIDROWEB_TOKEN"
AUTH_HEADER_NAME = "Authorization"

TIPOS_ESTACAO = {
    1: "Fluviométrica",
    2: "Pluviométrica",
}
