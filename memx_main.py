"""
memx HTTP Bridge — exposes memx Python library as a REST API for the
OpenClaw TypeScript plugin to consume.

Endpoints:
  POST /add          — store messages, extract facts
  POST /search       — semantic search
  POST /list         — list all memories for a user
  GET  /get/{id}     — fetch one memory by ID
  DELETE /del/{id}   — delete one memory
  POST /delete_all   — delete all memories for a user
  GET  /stats        — knowledge base statistics
  GET  /health       — liveness probe
"""

import logging
from typing import Optional, List, Any, Dict

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

logging.basicConfig(level=logging.INFO, format="%(asctime)s [memx] %(message)s")
log = logging.getLogger("memx-bridge")

app = FastAPI(title="memx HTTP Bridge", version="1.0.0")

# ACE is disabled until memx fully supports mem0ai 1.x API.
# In proxy mode memx is a thin wrapper over mem0 — all storage/retrieval
# goes through Qdrant + Ollama nomic-embed-text + qwen3:8b.
MEMX_CONFIG = {
    "ace_enabled": False,
    "embedder": {
        "provider": "ollama",
        "config": {
            "model": "nomic-embed-text",
            "ollama_base_url": "http://127.0.0.1:11434",
            "embedding_dims": 768,
        },
    },
    "llm": {
        "provider": "ollama",
        "config": {
            "model": "qwen3:8b",
            "ollama_base_url": "http://127.0.0.1:11434",
        },
    },
    "vector_store": {
        "provider": "qdrant",
        "config": {
            "host": "127.0.0.1",
            "port": 6333,
            "collection_name": "memx_memory",
            "embedding_model_dims": 768,
        },
    },
}

_memory = None

def get_memory():
    global _memory
    if _memory is None:
        from memx import Memory
        log.info("Initializing memx Memory instance...")
        _memory = Memory.from_config(MEMX_CONFIG)
        log.info("memx ready.")
    return _memory


class Message(BaseModel):
    role: str
    content: str

class AddRequest(BaseModel):
    messages: List[Message]
    user_id: str
    run_id: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None

class SearchRequest(BaseModel):
    query: str
    user_id: str
    limit: int = 5
    threshold: float = 0.3
    run_id: Optional[str] = None

class ListRequest(BaseModel):
    user_id: str
    run_id: Optional[str] = None

class DeleteAllRequest(BaseModel):
    user_id: str


def normalize_results(raw) -> List[Dict[str, Any]]:
    if isinstance(raw, list):
        return raw
    if isinstance(raw, dict):
        for key in ("results", "memories", "data"):
            if key in raw and isinstance(raw[key], list):
                return raw[key]
    return []

def normalize_item(item) -> Dict[str, Any]:
    if isinstance(item, dict):
        return item
    try:
        return {
            "id": str(getattr(item, "id", "") or ""),
            "memory": str(getattr(item, "memory", getattr(item, "text", "")) or ""),
            "user_id": str(getattr(item, "user_id", "") or ""),
            "score": float(getattr(item, "score", 0) or 0),
            "categories": list(getattr(item, "categories", []) or []),
            "metadata": dict(getattr(item, "metadata", {}) or {}),
            "created_at": str(getattr(item, "created_at", "") or ""),
            "updated_at": str(getattr(item, "updated_at", "") or ""),
        }
    except Exception:
        return {"raw": str(item)}


@app.get("/health")
def health():
    return {"status": "ok", "version": "1.0.0"}


@app.post("/add")
def add_memory(req: AddRequest):
    mem = get_memory()
    msgs = [{"role": m.role, "content": m.content} for m in req.messages]
    kwargs: Dict[str, Any] = {"user_id": req.user_id}
    if req.run_id:
        kwargs["run_id"] = req.run_id
    if req.metadata:
        kwargs["metadata"] = req.metadata
    try:
        raw = mem.add(msgs, **kwargs)
    except Exception as e:
        log.error(f"add failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))

    if isinstance(raw, dict) and "results" in raw:
        results = raw["results"]
    elif isinstance(raw, list):
        results = raw
    else:
        results = []

    normalized = []
    for r in results:
        if isinstance(r, dict):
            normalized.append(r)
        else:
            normalized.append({
                "id": str(getattr(r, "id", "")),
                "memory": str(getattr(r, "memory", getattr(r, "text", ""))),
                "event": str(getattr(r, "event", "ADD")),
            })
    return {"results": normalized}


@app.post("/search")
def search_memory(req: SearchRequest):
    mem = get_memory()
    kwargs: Dict[str, Any] = {"user_id": req.user_id, "limit": req.limit}
    if req.run_id:
        kwargs["run_id"] = req.run_id
    try:
        raw = mem.search(req.query, **kwargs)
    except Exception as e:
        log.error(f"search failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))
    items = normalize_results(raw)
    return {"results": [normalize_item(i) for i in items]}


@app.post("/list")
def list_memories(req: ListRequest):
    mem = get_memory()
    kwargs: Dict[str, Any] = {"user_id": req.user_id}
    if req.run_id:
        kwargs["run_id"] = req.run_id
    try:
        raw = mem.get_all(**kwargs)
    except Exception as e:
        log.error(f"list failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))
    items = normalize_results(raw)
    return {"results": [normalize_item(i) for i in items]}


@app.get("/get/{memory_id}")
def get_memory_by_id(memory_id: str):
    mem = get_memory()
    try:
        raw = mem.get(memory_id)
    except Exception as e:
        raise HTTPException(status_code=404, detail=str(e))
    return normalize_item(raw)


@app.delete("/del/{memory_id}")
def delete_memory(memory_id: str):
    mem = get_memory()
    try:
        mem.delete(memory_id)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    return {"success": True}


@app.post("/delete_all")
def delete_all_memories(req: DeleteAllRequest):
    mem = get_memory()
    try:
        mem.delete_all(user_id=req.user_id)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    return {"success": True}


@app.get("/stats")
def get_stats():
    mem = get_memory()
    try:
        status = mem.status()
        if hasattr(status, "__dict__"):
            return vars(status)
        if isinstance(status, dict):
            return status
        return {"raw": str(status)}
    except Exception as e:
        log.warning(f"stats failed: {e}")
        try:
            all_mem = mem.get_all(user_id="anita")
            count = len(normalize_results(all_mem))
            return {"total": count, "ace_enabled": True}
        except Exception as e2:
            return {"error": str(e2)}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="127.0.0.1", port=7788, log_level="info")
