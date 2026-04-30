package provider

// Preset holds display and connection info for a well-known API provider.
// Inspired by CC-Switch's provider_defaults — adapted for the Lurus ecosystem.
type Preset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`        // icon key for frontend (e.g. "openai", "anthropic")
	IconColor   string `json:"iconColor"`   // hex color
	Category    string `json:"category"`    // "official" | "china" | "proxy" | "self-hosted"
	BaseURL     string `json:"baseUrl"`     // default API endpoint
	KeyFormat   string `json:"keyFormat"`   // placeholder hint (e.g. "sk-...")
	DocsURL     string `json:"docsUrl"`     // link to API docs
	Models      string `json:"models"`      // comma-separated popular model IDs
	Description string `json:"description"` // one-liner
	FreeTier    bool   `json:"freeTier"`    // true if provider offers a meaningful free tier (no credit card)
	NeedsProxy  bool   `json:"needsProxy"`  // true if blocked in mainland China (needs xray/VPN proxy)
}

// Presets returns all built-in provider presets.
// Data sourced from CC-Switch's provider_defaults.rs + community contributions.
func Presets() []Preset {
	return builtinPresets
}

// PresetByID returns a single preset by ID, or nil if not found.
func PresetByID(id string) *Preset {
	for i := range builtinPresets {
		if builtinPresets[i].ID == id {
			return &builtinPresets[i]
		}
	}
	return nil
}

// PresetsByCategory returns presets filtered by category.
func PresetsByCategory(category string) []Preset {
	var out []Preset
	for _, p := range builtinPresets {
		if p.Category == category {
			out = append(out, p)
		}
	}
	return out
}

var builtinPresets = []Preset{
	// ═══════════════════════════════════════
	// Official providers
	// ═══════════════════════════════════════
	{
		ID: "anthropic", Name: "Anthropic", Icon: "anthropic", IconColor: "#D4915D",
		Category: "official", BaseURL: "https://api.anthropic.com",
		KeyFormat: "sk-ant-...", DocsURL: "https://docs.anthropic.com",
		Models:      "claude-opus-4-20250514,claude-sonnet-4-20250514,claude-haiku-4-5-20251001",
		Description: "Claude models — Opus, Sonnet, Haiku",
		NeedsProxy: true,
	},
	{
		ID: "openai", Name: "OpenAI", Icon: "openai", IconColor: "#00A67E",
		Category: "official", BaseURL: "https://api.openai.com/v1",
		KeyFormat: "sk-...", DocsURL: "https://platform.openai.com/docs",
		Models:      "gpt-4.1,gpt-4.1-mini,o3,o4-mini,codex-mini-latest",
		Description: "GPT-4.1, o3, Codex — chat, reasoning, code",
		NeedsProxy: true,
	},
	{
		ID: "google", Name: "Google AI", Icon: "google", IconColor: "#4285F4",
		Category: "official", BaseURL: "https://generativelanguage.googleapis.com/v1beta",
		KeyFormat: "AIza...", DocsURL: "https://ai.google.dev/docs",
		Models:      "gemini-2.5-pro,gemini-2.5-flash,gemini-2.0-flash",
		Description: "Gemini 2.5 Pro/Flash — multimodal, long context",
		FreeTier: true, NeedsProxy: true,
	},
	{
		ID: "mistral", Name: "Mistral AI", Icon: "mistral", IconColor: "#FF7000",
		Category: "official", BaseURL: "https://api.mistral.ai/v1",
		KeyFormat: "...", DocsURL: "https://docs.mistral.ai",
		Models:      "mistral-large-latest,codestral-latest,mistral-small-latest",
		Description: "Mistral Large, Codestral, Small",
		NeedsProxy: true,
	},
	{
		ID: "xai", Name: "xAI", Icon: "xai", IconColor: "#000000",
		Category: "official", BaseURL: "https://api.x.ai/v1",
		KeyFormat: "xai-...", DocsURL: "https://docs.x.ai",
		Models:      "grok-3,grok-3-mini",
		Description: "Grok-3, Grok-3 Mini",
		NeedsProxy: true,
	},
	{
		ID: "cohere", Name: "Cohere", Icon: "cohere", IconColor: "#39594D",
		Category: "official", BaseURL: "https://api.cohere.com/v2",
		KeyFormat: "...", DocsURL: "https://docs.cohere.com",
		Models:      "command-a-03-2025,command-r-plus",
		Description: "Command A, Command R+",
		NeedsProxy: true,
	},

	// ═══════════════════════════════════════
	// China providers
	// ═══════════════════════════════════════
	{
		ID: "deepseek", Name: "DeepSeek", Icon: "deepseek", IconColor: "#1E88E5",
		Category: "china", BaseURL: "https://api.deepseek.com",
		KeyFormat: "sk-...", DocsURL: "https://platform.deepseek.com/api-docs",
		Models:      "deepseek-chat,deepseek-reasoner",
		Description: "DeepSeek V3 / R1 — high-performance open model",
	},
	{
		ID: "zhipu", Name: "Zhipu AI (智谱)", Icon: "zhipu", IconColor: "#0F62FE",
		Category: "china", BaseURL: "https://open.bigmodel.cn/api/paas/v4",
		KeyFormat: "...", DocsURL: "https://open.bigmodel.cn/dev/howuse/introduction",
		Models:      "glm-4-plus,glm-4-flash",
		Description: "GLM-4 Plus/Flash — 中文优化",
	},
	{
		ID: "kimi", Name: "Kimi (月之暗面)", Icon: "kimi", IconColor: "#6366F1",
		Category: "china", BaseURL: "https://api.moonshot.cn/v1",
		KeyFormat: "sk-...", DocsURL: "https://platform.moonshot.cn/docs",
		Models:      "moonshot-v1-128k,moonshot-v1-32k,moonshot-v1-8k",
		Description: "Kimi — 超长上下文中文模型",
	},
	{
		ID: "qwen", Name: "Qwen (通义千问)", Icon: "alibaba", IconColor: "#FF6A00",
		Category: "china", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		KeyFormat: "sk-...", DocsURL: "https://help.aliyun.com/zh/model-studio/",
		Models:      "qwen-max,qwen-plus,qwen-turbo,qwen-vl-max",
		Description: "Qwen 2.5 — 阿里云百炼平台",
	},
	{
		ID: "baidu", Name: "Ernie (文心一言)", Icon: "baidu", IconColor: "#2932E1",
		Category: "china", BaseURL: "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop",
		KeyFormat: "...", DocsURL: "https://cloud.baidu.com/doc/WENXINWORKSHOP",
		Models:      "ernie-4.0-turbo-128k,ernie-speed-pro-128k",
		Description: "Ernie 4.0 — 百度文心",
	},
	{
		ID: "minimax", Name: "MiniMax", Icon: "minimax", IconColor: "#FF6B6B",
		Category: "china", BaseURL: "https://api.minimax.chat/v1",
		KeyFormat: "...", DocsURL: "https://platform.minimaxi.com/document",
		Models:      "MiniMax-Text-01,abab7-chat",
		Description: "MiniMax — 多模态中文模型",
	},
	{
		ID: "doubao", Name: "Doubao (豆包)", Icon: "bytedance", IconColor: "#3C8CE7",
		Category: "china", BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		KeyFormat: "...", DocsURL: "https://www.volcengine.com/docs/82379",
		Models:      "doubao-pro-256k,doubao-lite-128k",
		Description: "豆包 — 字节跳动火山引擎",
	},
	{
		ID: "yi", Name: "Yi (零一万物)", Icon: "yi", IconColor: "#1A73E8",
		Category: "china", BaseURL: "https://api.lingyiwanwu.com/v1",
		KeyFormat: "...", DocsURL: "https://platform.lingyiwanwu.com",
		Models:      "yi-lightning,yi-large",
		Description: "Yi Lightning/Large — 零一万物",
	},
	{
		ID: "stepfun", Name: "Step (阶跃星辰)", Icon: "stepfun", IconColor: "#7C3AED",
		Category: "china", BaseURL: "https://api.stepfun.com/v1",
		KeyFormat: "...", DocsURL: "https://platform.stepfun.com",
		Models:      "step-2-16k,step-1-flash",
		Description: "Step 2 — 阶跃星辰",
	},

	// ═══════════════════════════════════════
	// Proxy / aggregator platforms
	// ═══════════════════════════════════════
	{
		ID: "openrouter", Name: "OpenRouter", Icon: "openrouter", IconColor: "#6366F1",
		Category: "proxy", BaseURL: "https://openrouter.ai/api/v1",
		KeyFormat: "sk-or-...", DocsURL: "https://openrouter.ai/docs",
		Models:      "anthropic/claude-sonnet-4,openai/gpt-4.1,google/gemini-2.5-pro",
		Description: "Multi-model aggregator — pay per token",
		FreeTier: true, NeedsProxy: true,
	},
	{
		ID: "together", Name: "Together AI", Icon: "together", IconColor: "#0EA5E9",
		Category: "proxy", BaseURL: "https://api.together.xyz/v1",
		KeyFormat: "...", DocsURL: "https://docs.together.ai",
		Models:      "meta-llama/Llama-4-Maverick-17B-128E,deepseek-ai/DeepSeek-R1",
		Description: "Open model hosting — Llama, DeepSeek, Mistral",
		NeedsProxy: true,
	},
	{
		ID: "groq", Name: "Groq", Icon: "groq", IconColor: "#F55036",
		Category: "proxy", BaseURL: "https://api.groq.com/openai/v1",
		KeyFormat: "gsk_...", DocsURL: "https://console.groq.com/docs",
		Models:      "meta-llama/llama-4-scout-17b-16e-instruct,openai/gpt-oss-120b,llama-3.3-70b-versatile,moonshotai/kimi-k2-instruct,qwen/qwen3-32b,llama-3.1-8b-instant",
		Description: "Ultra-fast free inference — Llama 4, GPT-OSS 120B, Kimi K2, Qwen3",
		FreeTier: true, NeedsProxy: true,
	},
	{
		ID: "fireworks", Name: "Fireworks AI", Icon: "fireworks", IconColor: "#FF6B35",
		Category: "proxy", BaseURL: "https://api.fireworks.ai/inference/v1",
		KeyFormat: "fw_...", DocsURL: "https://docs.fireworks.ai",
		Models:      "accounts/fireworks/models/llama-v3p3-70b-instruct",
		Description: "Fast open model inference",
		NeedsProxy: true,
	},
	{
		ID: "siliconflow", Name: "SiliconFlow (硅基流动)", Icon: "siliconflow", IconColor: "#8B5CF6",
		Category: "proxy", BaseURL: "https://api.siliconflow.cn/v1",
		KeyFormat: "sk-...", DocsURL: "https://docs.siliconflow.cn",
		Models:      "deepseek-ai/DeepSeek-V3,Qwen/Qwen2.5-72B-Instruct",
		Description: "中国 GPU 云推理平台 — 价格优势",
	},
	{
		ID: "lurus", Name: "Lurus (本平台)", Icon: "lurus", IconColor: "#FF8C69",
		Category: "proxy", BaseURL: "https://api.lurus.cn",
		KeyFormat: "sk-...", DocsURL: "https://docs.lurus.cn",
		Models:      "deepseek-chat,claude-sonnet-4-20250514,gpt-4.1-mini",
		Description: "Lurus 统一网关 — 一次充值用所有模型",
	},

	// ═══════════════════════════════════════
	// Cloud platforms (AWS, Azure, GCP)
	// ═══════════════════════════════════════
	{
		ID: "azure", Name: "Azure OpenAI", Icon: "azure", IconColor: "#0078D4",
		Category: "cloud", BaseURL: "https://{resource}.openai.azure.com/openai/deployments/{deployment}",
		KeyFormat: "...", DocsURL: "https://learn.microsoft.com/azure/ai-services/openai/",
		Models:      "gpt-4.1,gpt-4.1-mini",
		Description: "Azure-hosted OpenAI — enterprise compliance",
	},
	{
		ID: "bedrock", Name: "AWS Bedrock", Icon: "aws", IconColor: "#FF9900",
		Category: "cloud", BaseURL: "https://bedrock-runtime.{region}.amazonaws.com",
		KeyFormat: "AKIA...", DocsURL: "https://docs.aws.amazon.com/bedrock/",
		Models:      "anthropic.claude-sonnet-4-20250514-v1:0,meta.llama4-maverick-17b-instruct-v1:0",
		Description: "AWS-hosted models — Claude, Llama, Titan",
	},
	{
		ID: "vertex", Name: "Google Vertex AI", Icon: "google", IconColor: "#34A853",
		Category: "cloud", BaseURL: "https://{region}-aiplatform.googleapis.com/v1",
		KeyFormat: "(service account)", DocsURL: "https://cloud.google.com/vertex-ai/docs",
		Models:      "gemini-2.5-pro,gemini-2.5-flash",
		Description: "GCP Vertex — enterprise Gemini hosting",
	},

	// ═══════════════════════════════════════
	// Self-hosted / local
	// ═══════════════════════════════════════
	{
		ID: "ollama", Name: "Ollama", Icon: "ollama", IconColor: "#FFFFFF",
		Category: "self-hosted", BaseURL: "http://localhost:11434/v1",
		KeyFormat: "ollama", DocsURL: "https://ollama.ai",
		Models:      "llama3.3,qwen2.5-coder,deepseek-r1",
		Description: "Local model runner — zero API cost",
	},
	{
		ID: "lmstudio", Name: "LM Studio", Icon: "lmstudio", IconColor: "#0A7AFF",
		Category: "self-hosted", BaseURL: "http://localhost:1234/v1",
		KeyFormat: "lm-studio", DocsURL: "https://lmstudio.ai",
		Models:      "(loaded model)", Description: "Local model GUI with OpenAI-compatible API",
	},
	{
		ID: "vllm", Name: "vLLM", Icon: "vllm", IconColor: "#E91E63",
		Category: "self-hosted", BaseURL: "http://localhost:8000/v1",
		KeyFormat: "token-...", DocsURL: "https://docs.vllm.ai",
		Models:      "(deployed model)", Description: "High-throughput inference server",
	},
	{
		ID: "custom", Name: "Custom (OpenAI Compatible)", Icon: "custom", IconColor: "#6B7280",
		Category: "self-hosted", BaseURL: "http://localhost:8080/v1",
		KeyFormat: "...", DocsURL: "",
		Models:      "", Description: "Any OpenAI-compatible API endpoint",
	},
}
