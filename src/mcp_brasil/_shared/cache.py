"""Simple async-safe TTL cache for API responses.

Avoids hammering government APIs with repeated identical requests.
Uses an in-memory dict with per-entry expiration — no external deps.

Usage:
    from mcp_brasil._shared.cache import ttl_cache

    cache = ttl_cache(ttl=300)  # 5 minutes

    @cache
    async def listar_estados() -> list[Estado]:
        ...  # HTTP call only happens if cache miss or expired
"""

from __future__ import annotations

import functools
import time
from collections.abc import Callable
from typing import Any, TypeVar

F = TypeVar("F", bound=Callable[..., Any])
_MISSING = object()


class TTLCache:
    """In-memory cache with per-entry TTL expiration.

    Thread-safe for asyncio (single-threaded event loop).
    Not suitable for multi-process deployments — use Redis for that.
    """

    def __init__(self, ttl: float = 300.0, maxsize: int = 256) -> None:
        self._ttl = ttl
        self._maxsize = maxsize
        self._store: dict[str, tuple[float, Any]] = {}

    @property
    def size(self) -> int:
        """Number of entries currently in cache (including expired)."""
        return len(self._store)

    def get(self, key: str, default: Any = None) -> Any:
        """Get a value if it exists and hasn't expired."""
        entry = self._store.get(key)
        if entry is None:
            return default
        expires_at, value = entry
        if time.monotonic() > expires_at:
            del self._store[key]
            return default
        return value

    def set(self, key: str, value: Any) -> None:
        """Store a value with TTL expiration."""
        if len(self._store) >= self._maxsize:
            self._evict()
        self._store[key] = (time.monotonic() + self._ttl, value)

    def clear(self) -> None:
        """Remove all entries."""
        self._store.clear()

    def _evict(self) -> None:
        """Remove expired entries; if still full, remove oldest."""
        now = time.monotonic()
        expired = [k for k, (exp, _) in self._store.items() if now > exp]
        for k in expired:
            del self._store[k]

        # Still at capacity? Remove the entry expiring soonest
        if len(self._store) >= self._maxsize:
            oldest_key = min(self._store, key=lambda k: self._store[k][0])
            del self._store[oldest_key]


def ttl_cache(ttl: float = 300.0, maxsize: int = 256) -> Callable[[F], F]:
    """Decorator that caches async function results with TTL.

    Cache key is built from function name + stringified args/kwargs.

    Args:
        ttl: Time-to-live in seconds. Default: 300 (5 minutes).
        maxsize: Maximum cache entries. Default: 256.

    Returns:
        Decorator that wraps an async function with caching.

    Example:
        @ttl_cache(ttl=60)
        async def buscar_estados() -> list[Estado]:
            return await http_get(...)
    """
    cache = TTLCache(ttl=ttl, maxsize=maxsize)

    def decorator(func: F) -> F:
        @functools.wraps(func)
        async def wrapper(*args: Any, **kwargs: Any) -> Any:
            key = f"{func.__qualname__}:{args!r}:{kwargs!r}"
            cached = cache.get(key, _MISSING)
            if cached is not _MISSING:
                return cached
            result = await func(*args, **kwargs)
            cache.set(key, result)
            return result

        # Expose cache for testing/clearing
        wrapper.cache = cache  # type: ignore[attr-defined]
        return wrapper  # type: ignore[return-value]

    return decorator
