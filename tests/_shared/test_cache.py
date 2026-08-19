"""Testes do cache com TTL."""

import time

import pytest

from mcp_brasil._shared.cache import TTLCache, ttl_cache


class TestTTLCache:
    def test_set_and_get(self) -> None:
        cache = TTLCache(ttl=60)
        cache.set("key", "value")
        assert cache.get("key") == "value"

    def test_get_missing_key(self) -> None:
        cache = TTLCache(ttl=60)
        assert cache.get("missing") is None

    def test_expired_entry_returns_none(self) -> None:
        cache = TTLCache(ttl=0.01)
        cache.set("key", "value")
        time.sleep(0.02)
        assert cache.get("key") is None

    def test_clear(self) -> None:
        cache = TTLCache(ttl=60)
        cache.set("a", 1)
        cache.set("b", 2)
        cache.clear()
        assert cache.size == 0

    def test_maxsize_eviction(self) -> None:
        cache = TTLCache(ttl=60, maxsize=2)
        cache.set("a", 1)
        cache.set("b", 2)
        cache.set("c", 3)  # should evict oldest
        assert cache.size <= 2

    def test_evicts_expired_first(self) -> None:
        cache = TTLCache(ttl=0.01, maxsize=2)
        cache.set("a", 1)
        time.sleep(0.02)  # "a" expires
        cache.set("b", 2)
        cache.set("c", 3)  # evicts expired "a", not "b"
        assert cache.get("b") == 2
        assert cache.get("c") == 3

    def test_size_property(self) -> None:
        cache = TTLCache(ttl=60)
        assert cache.size == 0
        cache.set("x", 1)
        assert cache.size == 1


class TestTtlCacheDecorator:
    @pytest.mark.asyncio
    async def test_caches_none_result(self) -> None:
        call_count = 0

        @ttl_cache(ttl=60)
        async def fetch_optional() -> None:
            nonlocal call_count
            call_count += 1
            return None

        assert await fetch_optional() is None
        assert await fetch_optional() is None
        assert call_count == 1

    @pytest.mark.asyncio
    async def test_caches_result(self) -> None:
        call_count = 0

        @ttl_cache(ttl=60)
        async def fetch_data(key: str) -> str:
            nonlocal call_count
            call_count += 1
            return f"result-{key}"

        result1 = await fetch_data("a")
        result2 = await fetch_data("a")
        assert result1 == "result-a"
        assert result2 == "result-a"
        assert call_count == 1  # second call was cached

    @pytest.mark.asyncio
    async def test_different_args_different_cache(self) -> None:
        call_count = 0

        @ttl_cache(ttl=60)
        async def fetch(key: str) -> str:
            nonlocal call_count
            call_count += 1
            return f"result-{key}"

        await fetch("a")
        await fetch("b")
        assert call_count == 2

    @pytest.mark.asyncio
    async def test_expired_cache_refetches(self) -> None:
        call_count = 0

        @ttl_cache(ttl=0.01)
        async def fetch() -> str:
            nonlocal call_count
            call_count += 1
            return "data"

        await fetch()
        time.sleep(0.02)
        await fetch()
        assert call_count == 2

    @pytest.mark.asyncio
    async def test_cache_attribute_exposed(self) -> None:
        @ttl_cache(ttl=60)
        async def fetch() -> str:
            return "data"

        assert hasattr(fetch, "cache")
        assert isinstance(fetch.cache, TTLCache)

    @pytest.mark.asyncio
    async def test_clear_cache(self) -> None:
        call_count = 0

        @ttl_cache(ttl=60)
        async def fetch() -> str:
            nonlocal call_count
            call_count += 1
            return "data"

        await fetch()
        fetch.cache.clear()
        await fetch()
        assert call_count == 2

    @pytest.mark.asyncio
    async def test_kwargs_in_cache_key(self) -> None:
        call_count = 0

        @ttl_cache(ttl=60)
        async def fetch(uf: str = "SP") -> str:
            nonlocal call_count
            call_count += 1
            return f"data-{uf}"

        await fetch(uf="SP")
        await fetch(uf="RJ")
        await fetch(uf="SP")  # cached
        assert call_count == 2
