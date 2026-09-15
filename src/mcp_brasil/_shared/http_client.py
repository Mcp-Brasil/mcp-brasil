"""Shared async HTTP client for mcp-brasil.

Provides a configured httpx.AsyncClient factory and a fetch helper
with retry + exponential backoff for transient errors (5xx, 429, timeouts).

Usage:
    from mcp_brasil._shared.http_client import create_client, http_get

    # Option 1: client factory (for multiple requests in a feature client)
    async with create_client(base_url="https://api.example.com") as client:
        response = await client.get("/endpoint")

    # Option 2: one-shot fetch with automatic retry
    data = await http_get("https://api.example.com/endpoint")
"""

from __future__ import annotations

import asyncio
import logging
from typing import Any

import httpx

from mcp_brasil.exceptions import HttpClientError
from mcp_brasil.settings import HTTP_BACKOFF_BASE, HTTP_MAX_RETRIES, HTTP_TIMEOUT, USER_AGENT

logger = logging.getLogger(__name__)

# Status codes that trigger a retry
_RETRYABLE_STATUS_CODES = frozenset({429, 500, 502, 503, 504})


def create_client(
    base_url: str = "",
    timeout: float | None = None,
    headers: dict[str, str] | None = None,
) -> httpx.AsyncClient:
    """Create a configured httpx.AsyncClient.

    Args:
        base_url: Base URL for all requests.
        timeout: Request timeout in seconds. Default: settings.HTTP_TIMEOUT.
        headers: Extra headers to merge with defaults.

    Returns:
        Configured httpx.AsyncClient (use as async context manager).
    """
    default_headers = {
        "User-Agent": USER_AGENT,
        "Accept": "application/json",
    }
    if headers:
        default_headers.update(headers)

    return httpx.AsyncClient(
        base_url=base_url,
        timeout=httpx.Timeout(timeout or HTTP_TIMEOUT),
        headers=default_headers,
        follow_redirects=True,
    )


async def http_get(
    url: str,
    *,
    params: dict[str, Any] | None = None,
    headers: dict[str, str] | None = None,
    timeout: float | None = None,
    max_retries: int | None = None,
) -> Any:
    """Make a GET request with retry + exponential backoff.

    Retries on: HTTP 429/5xx, timeouts, and connection errors.
    Does NOT retry on 4xx (except 429) — those are client errors.

    Args:
        url: Full URL to request.
        params: Query parameters.
        headers: Extra headers (merged with defaults).
        timeout: Request timeout in seconds.
        max_retries: Max retry attempts. Default: settings.HTTP_MAX_RETRIES.

    Returns:
        Parsed JSON response.

    Raises:
        HttpClientError: On non-retryable errors or exhausted retries.
    """
    retries = max_retries if max_retries is not None else HTTP_MAX_RETRIES
    last_error: Exception | None = None

    async with create_client(timeout=timeout, headers=headers) as client:
        for attempt in range(retries + 1):
            try:
                response = await client.get(url, params=params)

                if response.status_code in _RETRYABLE_STATUS_CODES:
                    if attempt < retries:
                        wait = HTTP_BACKOFF_BASE * (2**attempt)
                        logger.warning(
                            "Retry %d/%d for %s (HTTP %d), waiting %.1fs",
                            attempt + 1,
                            retries,
                            url,
                            response.status_code,
                            wait,
                        )
                        await asyncio.sleep(wait)
                        continue
                    # Last attempt still failed with retryable status
                    raise HttpClientError(
                        f"Request to {url} failed after {retries + 1} attempts "
                        f"(last: HTTP {response.status_code})"
                    )

                response.raise_for_status()
                try:
                    return response.json()
                except ValueError as exc:
                    raise HttpClientError(
                        f"HTTP {response.status_code} from {url} returned "
                        f"a non-JSON body: {response.text[:200]!r}. "
                        "The API may be returning an error page instead of data."
                    ) from exc

            except httpx.HTTPStatusError as exc:
                raise HttpClientError(
                    f"HTTP {exc.response.status_code} from {url}: {exc.response.text[:200]}"
                ) from exc

            except (httpx.TimeoutException, httpx.ConnectError) as exc:
                last_error = exc
                if attempt < retries:
                    wait = HTTP_BACKOFF_BASE * (2**attempt)
                    logger.warning(
                        "Request to %s failed (attempt %d/%d): %s, waiting %.1fs",
                        url,
                        attempt + 1,
                        retries,
                        type(exc).__name__,
                        wait,
                    )
                    await asyncio.sleep(wait)
                    continue

    raise HttpClientError(f"Request to {url} failed after {retries + 1} attempts") from last_error


async def http_post(
    url: str,
    *,
    json_body: Any | None = None,
    params: dict[str, Any] | None = None,
    headers: dict[str, str] | None = None,
    timeout: float | None = None,
    max_retries: int | None = None,
) -> Any:
    """Make a POST request with retry + exponential backoff.

    Retries on: HTTP 429/5xx, timeouts, and connection errors.

    Args:
        url: Full URL to request.
        json_body: JSON body to send.
        params: Query parameters.
        headers: Extra headers (merged with defaults).
        timeout: Request timeout in seconds.
        max_retries: Max retry attempts. Default: settings.HTTP_MAX_RETRIES.

    Returns:
        Parsed JSON response.

    Raises:
        HttpClientError: On non-retryable errors or exhausted retries.
    """
    retries = max_retries if max_retries is not None else HTTP_MAX_RETRIES
    last_error: Exception | None = None

    async with create_client(timeout=timeout, headers=headers) as client:
        for attempt in range(retries + 1):
            try:
                response = await client.post(url, json=json_body, params=params)

                if response.status_code in _RETRYABLE_STATUS_CODES:
                    if attempt < retries:
                        wait = HTTP_BACKOFF_BASE * (2**attempt)
                        logger.warning(
                            "Retry %d/%d for %s (HTTP %d), waiting %.1fs",
                            attempt + 1,
                            retries,
                            url,
                            response.status_code,
                            wait,
                        )
                        await asyncio.sleep(wait)
                        continue
                    raise HttpClientError(
                        f"Request to {url} failed after {retries + 1} attempts "
                        f"(last: HTTP {response.status_code})"
                    )

                response.raise_for_status()
                try:
                    return response.json()
                except ValueError as exc:
                    raise HttpClientError(
                        f"HTTP {response.status_code} from {url} returned "
                        f"a non-JSON body: {response.text[:200]!r}. "
                        "The API may be returning an error page instead of data."
                    ) from exc

            except httpx.HTTPStatusError as exc:
                raise HttpClientError(
                    f"HTTP {exc.response.status_code} from {url}: {exc.response.text[:200]}"
                ) from exc

            except (httpx.TimeoutException, httpx.ConnectError) as exc:
                last_error = exc
                if attempt < retries:
                    wait = HTTP_BACKOFF_BASE * (2**attempt)
                    logger.warning(
                        "Request to %s failed (attempt %d/%d): %s, waiting %.1fs",
                        url,
                        attempt + 1,
                        retries,
                        type(exc).__name__,
                        wait,
                    )
                    await asyncio.sleep(wait)
                    continue

    raise HttpClientError(f"Request to {url} failed after {retries + 1} attempts") from last_error
