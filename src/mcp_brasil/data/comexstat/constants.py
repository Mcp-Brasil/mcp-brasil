"""Constants for the ComexStat feature."""

BASE_URL = "https://api-comexstat.mdic.gov.br"
QUERY_URL = f"{BASE_URL}/general"
COUNTRY_URL = f"{BASE_URL}/general/filters/country"

FLOW_EXPORT = "export"
FLOW_IMPORT = "import"

METRIC_FOB = "metricFOB"
METRIC_KG = "metricKG"

# Cloudflare on api-comexstat.mdic.gov.br blocks non-browser UAs
REQUEST_HEADERS: dict[str, str] = {
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/124.0.0.0 Safari/537.36"
    ),
    "Accept": "application/json, text/plain, */*",
    "Origin": "https://comexstat.mdic.gov.br",
    "Referer": "https://comexstat.mdic.gov.br/",
}
