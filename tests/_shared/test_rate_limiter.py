"""Tests for the async RateLimiter."""

from __future__ import annotations

import asyncio
import time

import pytest

from mcp_brasil._shared.rate_limiter import RateLimiter


class TestRateLimiter:
    @pytest.mark.asyncio
    async def test_allows_within_limit(self) -> None:
        limiter = RateLimiter(max_requests=5, period=60.0)
        for _ in range(5):
            async with limiter:
                pass
        # All 5 should succeed without blocking

    @pytest.mark.asyncio
    async def test_blocks_when_exhausted(self) -> None:
        limiter = RateLimiter(max_requests=2, period=60.0)
        async with limiter:
            pass
        async with limiter:
            pass

        # Third request should block; verify by using a short timeout
        with pytest.raises(asyncio.TimeoutError):
            await asyncio.wait_for(limiter.acquire(), timeout=0.05)

    @pytest.mark.asyncio
    async def test_allows_after_window_expires(self) -> None:
        limiter = RateLimiter(max_requests=1, period=0.05)
        async with limiter:
            pass
        # Wait for window to expire
        await asyncio.sleep(0.06)
        # Should be allowed now
        async with limiter:
            pass

    @pytest.mark.asyncio
    async def test_context_manager_protocol(self) -> None:
        limiter = RateLimiter(max_requests=10, period=60.0)
        async with limiter as ctx:
            assert ctx is limiter

    @pytest.mark.asyncio
    async def test_purge_removes_old_timestamps(self) -> None:
        limiter = RateLimiter(max_requests=2, period=0.05)
        now = time.monotonic()
        # Simulate old timestamps
        limiter._timestamps.append(now - 1.0)
        limiter._timestamps.append(now - 1.0)
        # Purge should clear them, allowing new requests
        async with limiter:
            pass
        assert len(limiter._timestamps) == 1

    @pytest.mark.asyncio
    async def test_concurrent_access(self) -> None:
        limiter = RateLimiter(max_requests=3, period=60.0)
        results: list[int] = []

        async def worker(i: int) -> None:
            async with limiter:
                results.append(i)

        await asyncio.gather(worker(0), worker(1), worker(2))
        assert sorted(results) == [0, 1, 2]
