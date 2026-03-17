import json, pathlib

cfg_path = pathlib.Path.home() / ".openclaw" / "openclaw.json"
cfg = json.loads(cfg_path.read_text())

providers = cfg.setdefault("models", {}).setdefault("providers", {})
gw = providers.setdefault("lurus-gateway", {})
models_list = gw.setdefault("models", [])

existing_ids = {m["id"] for m in models_list}

new_models = [
    {"id": "gemini-3.1-pro-preview", "name": "Gemini 3.1 Pro", "reasoning": False,
     "input": ["text","image"], "cost": {"input":0,"output":0,"cacheRead":0,"cacheWrite":0},
     "contextWindow": 1000000, "maxTokens": 65536},
    {"id": "gemini-3.1-flash-lite-preview", "name": "Gemini 3.1 Flash Lite", "reasoning": False,
     "input": ["text","image"], "cost": {"input":0,"output":0,"cacheRead":0,"cacheWrite":0},
     "contextWindow": 1000000, "maxTokens": 64000},
    {"id": "qwen3.5-plus", "name": "Qwen 3.5 Plus", "reasoning": False,
     "input": ["text","image"], "cost": {"input":0,"output":0,"cacheRead":0,"cacheWrite":0},
     "contextWindow": 131072, "maxTokens": 8192},
    {"id": "qwen3.5-flash", "name": "Qwen 3.5 Flash", "reasoning": False,
     "input": ["text"], "cost": {"input":0,"output":0,"cacheRead":0,"cacheWrite":0},
     "contextWindow": 131072, "maxTokens": 8192},
]

added = []
for m in new_models:
    if m["id"] not in existing_ids:
        models_list.append(m)
        added.append(m["id"])

# Upgrade default model
agents_defaults = cfg.setdefault("agents", {}).setdefault("defaults", {})
agents_defaults["model"] = {
    "primary": "lurus-gateway/gemini-3.1-flash-lite-preview",
    "fallbacks": [
        "lurus-gateway/gemini-3.1-pro-preview",
        "lurus-gateway/gemini-2.5-flash",
        "lurus-gateway/deepseek-chat",
        "ollama/qwen3:8b"
    ]
}

agent_model_map = {
    "taizi":    "lurus-gateway/gemini-3.1-flash-lite-preview",
    "zhongshu": "lurus-gateway/gemini-3.1-pro-preview",
    "menxia":   "lurus-gateway/gemini-3.1-pro-preview",
    "shangshu": "lurus-gateway/gemini-3.1-flash-lite-preview",
    "hubu":     "lurus-gateway/gemini-3.1-flash-lite-preview",
    "libu":     "lurus-gateway/gemini-3.1-flash-lite-preview",
    "bingbu":   "lurus-gateway/gemini-3.1-flash-lite-preview",
    "xingbu":   "lurus-gateway/gemini-3.1-flash-lite-preview",
    "gongbu":   "lurus-gateway/gemini-3.1-flash-lite-preview",
    "libu_hr":  "lurus-gateway/gemini-3.1-flash-lite-preview",
    "zaochao":  "lurus-gateway/gemini-3.1-flash-lite-preview",
    "main":     "lurus-gateway/gemini-3.1-flash-lite-preview",
    "techie":   "lurus-gateway/gemini-3.1-pro-preview",
}

agents_list = cfg.get("agents", {}).get("list", [])
for ag in agents_list:
    ag_id = ag.get("id", "")
    if ag_id in agent_model_map:
        old = ag.get("model", "default")
        ag["model"] = agent_model_map[ag_id]
        print(ag_id, ":", old, "->", ag["model"])

cfg_path.write_text(json.dumps(cfg, ensure_ascii=False, indent=2))
print("Added models:", added)
print("Done")
