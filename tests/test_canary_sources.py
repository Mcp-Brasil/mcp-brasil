"""Offline consistency tests for scripts/canary_sources.py.

These tests never touch the network: they only validate that every entry
in CANARY_URLS refers to a real, discoverable feature and is well-formed.
Live probing is the canary workflow's job (.github/workflows/canary-sources.yml).
"""

import importlib.util
import pathlib
import sys

from mcp_brasil._shared.feature import FeatureRegistry

_CANARY_PATH = pathlib.Path(__file__).resolve().parent.parent / "scripts" / "canary_sources.py"


def _load_canary_urls() -> dict[str, str]:
    spec = importlib.util.spec_from_file_location("canary_sources", _CANARY_PATH)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    # @dataclass resolves annotations via sys.modules — register first.
    sys.modules["canary_sources"] = module
    spec.loader.exec_module(module)
    urls: dict[str, str] = module.CANARY_URLS
    return urls


def _known_features() -> set[str]:
    reg = FeatureRegistry()
    reg.discover("mcp_brasil.data")
    reg.discover("mcp_brasil.agentes")
    return set(reg.features) | set(reg.skipped)


class TestCanaryUrls:
    def test_keys_match_discoverable_features(self) -> None:
        """Every CANARY_URLS key must be a real feature (issue #35)."""
        known = _known_features()
        for name in _load_canary_urls():
            assert name in known, f"CANARY_URLS[{name!r}] is not a known feature"

    def test_urls_well_formed(self) -> None:
        """Entries must be absolute https URLs without whitespace."""
        for name, url in _load_canary_urls().items():
            assert url.startswith("https://"), f"CANARY_URLS[{name!r}] must use https"
            assert not any(c.isspace() for c in url), f"CANARY_URLS[{name!r}] has space"

    def test_tce_ce_has_real_endpoint(self) -> None:
        """tce_ce must declare a real endpoint so path changes fail (issue #35)."""
        assert "tce_ce" in _load_canary_urls()
