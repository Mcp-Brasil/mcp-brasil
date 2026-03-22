"""Async rate limiter using a sliding window of timestamps.

Usage::

    limiter = RateLimiter(max_requests=80, period=60.0)

    async with limiter:
        await do_request()
"""

from __future__ import annotations

import asyncio
import time
from collections import deque


class RateLimiter:
    """Token-bucket-style rate limiter with sliding window.

    Args:
        max_requests: Maximum number of requests allowed in the window.
        period: Window duration in seconds.
    """

    def __init__(self, max_requests: int, period: float) -> None:
        self._max_requests = max_requests
        self._period = period
        self._timestamps: deque[float] = deque()
        self._lock = asyncio.Lock()

    def _purge(self, now: float) -> None:
        """Remove timestamps outside the current window."""
        cutoff = now - self._period
        while self._timestamps and self._timestamps[0] <= cutoff:
            self._timestamps.popleft()

    async def acquire(self) -> None:
        """Wait until a request slot is available, then reserve it."""
        while True:
            async with self._lock:
                now = time.monotonic()
                self._purge(now)
                if len(self._timestamps) < self._max_requests:
                    self._timestamps.append(now)
                    return
                # Calculate wait time until oldest entry expires
                wait = self._timestamps[0] + self._period - now
            await asyncio.sleep(max(wait, 0.01))

    async def __aenter__(self) -> RateLimiter:
        await self.acquire()
        return self

    async def __aexit__(self, *exc: object) -> None:
        pass
