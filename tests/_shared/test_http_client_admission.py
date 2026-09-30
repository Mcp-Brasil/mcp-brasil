import asyncio
import httpx
import pytest
import respx
from mcp_brasil._shared.http_client import http_get

@pytest.mark.asyncio
async def test_http_get_host_admission(monkeypatch):
    monkeypatch.setattr("mcp_brasil._shared.http_client.HTTP_MAX_CONCURRENCY_PER_HOST", 2)
    
    # We must re-init the semaphore dict to pick up the mocked setting
    from mcp_brasil._shared.http_client import _host_semaphores
    _host_semaphores.clear()
    
    in_flight = 0
    max_in_flight = 0
    
    async def slow_response(request):
        nonlocal in_flight, max_in_flight
        in_flight += 1
        if in_flight > max_in_flight:
            max_in_flight = in_flight
        await asyncio.sleep(0.1)
        in_flight -= 1
        return httpx.Response(200, json={"ok": True})

    with respx.mock:
        respx.get("https://api.example.com/data").mock(side_effect=slow_response)
        
        tasks = [http_get("https://api.example.com/data") for _ in range(5)]
        results = await asyncio.gather(*tasks)
        
        assert len(results) == 5
        assert max_in_flight == 2
