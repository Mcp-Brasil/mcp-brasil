import asyncio
import pytest
from mcp_brasil._shared.batch import execute_batch, build_dispatch, _dispatch

@pytest.mark.asyncio
async def test_execute_batch_timeout(monkeypatch):
    # Register a slow tool
    async def slow_tool():
        await asyncio.sleep(0.5)
        return "Concluido"
        
    async def fast_tool():
        return "Rapido"
        
    _dispatch["slow_tool"] = slow_tool
    _dispatch["fast_tool"] = fast_tool
    
    # Mock timeout to be very short
    monkeypatch.setattr("mcp_brasil._shared.batch.BATCH_TIMEOUT", 0.1)
    
    queries = [
        {"tool": "fast_tool", "args": {}},
        {"tool": "slow_tool", "args": {}},
    ]
    
    result = await execute_batch(queries, ctx=None)
    
    assert "## fast_tool\n\nRapido" in result
    assert "## slow_tool\n\nErro: Expirou (limite de tempo atingido antes da conclusao)." in result
