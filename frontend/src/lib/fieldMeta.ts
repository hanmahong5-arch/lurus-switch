// Field-level metadata for tool-config forms. Each entry augments a single
// JSON-path inside a tool's config schema with bilingual labels, optional
// description, the schema-default value, and (when the field has security
// implications) a safety classifier + bilingual advisory text.
//
// The keys follow the dotted JSON path of the field, prefixed by the tool
// name — e.g. 'claude.permissions.allowBash'. This intentionally mirrors
// the field path users would see in settings.json so debugging / cross-
// referencing is mechanical.

export type FieldStatus =
  | 'set' // user explicitly set to a non-default value
  | 'default' // using the schema default
  | 'unset' // empty / not configured (and no default)
  | 'risky' // set to a value with security implications
  | 'safe' // set to a value that improves security posture

// SecurityRole describes how the field's *value* maps to a safety verdict.
// Most security-relevant booleans fall into one of the four buckets below.
//
//   'risky-when-true':  AllowBash / AllowWebFetch / ExperimentalFeatures
//   'safe-when-true':   sandbox.enabled / disableTelemetry
//   'risky-when-false': (none in Claude today, but reserved for future)
//   'safe-when-false':  (ditto)
//
// Fields without a security role are treated as preferences and don't
// trigger advisories.
export type SecurityRole =
  | 'risky-when-true'
  | 'safe-when-true'
  | 'risky-when-false'
  | 'safe-when-false'

export interface FieldMeta {
  labelZh: string
  labelEn: string
  descZh?: string
  descEn?: string
  // Schema default — used to compute 'default' vs 'set' status. Compared
  // with === so primitives only; arrays/maps default-detection is heuristic
  // (treated as 'default' when empty/empty-map).
  defaultValue?: unknown
  // Whether an empty/unset value is acceptable. Defaults to true (most
  // fields are optional). Set to false for required fields like apiKey
  // when no gateway is configured.
  required?: boolean
  securityRole?: SecurityRole
  // Advisory shown when getStatus() returns 'risky'.
  advisoryRiskyZh?: string
  advisoryRiskyEn?: string
  // Advisory shown when getStatus() returns 'safe'.
  advisorySafeZh?: string
  advisorySafeEn?: string
}

export const FIELD_META: Record<string, FieldMeta> = {
  // ─── Claude · Core ────────────────────────────────────────────────
  'claude.model': {
    labelZh: '默认模型',
    labelEn: 'Default Model',
    descZh: '所有对话默认使用的 Claude 模型。',
    descEn: 'Default Claude model used for all conversations.',
    defaultValue: 'claude-sonnet-4-20250514',
  },
  'claude.apiKey': {
    labelZh: 'API 密钥',
    labelEn: 'API Key',
    descZh: 'Anthropic API 密钥。留空则使用系统环境变量 ANTHROPIC_API_KEY。',
    descEn: 'Anthropic API key. Leave empty to inherit from ANTHROPIC_API_KEY.',
  },
  'claude.maxTokens': {
    labelZh: '最大输出 Token 数',
    labelEn: 'Max Output Tokens',
    descZh: '单次回复的最大 token 上限。Claude 4 系列建议 8192，长上下文任务可上调。',
    descEn: 'Token cap per response. 8192 is the safe default for Claude 4; raise for long-form tasks.',
    defaultValue: 8192,
  },
  'claude.customInstructions': {
    labelZh: '自定义指令',
    labelEn: 'Custom Instructions',
    descZh: '附加到所有对话的全局系统提示。等同于在每条消息前插入这段文字。',
    descEn: 'Appended to every conversation as a global system prompt — like a preface to each message.',
  },

  // ─── Claude · Permissions ─────────────────────────────────────────
  'claude.permissions.allowBash': {
    labelZh: '允许执行 Shell 命令',
    labelEn: 'Allow Bash',
    descZh: '允许 Claude 在你的 shell 中运行命令（git/npm/build 等）。',
    descEn: 'Permit Claude to run shell commands (git/npm/build/etc).',
    defaultValue: true,
    securityRole: 'risky-when-true',
    advisoryRiskyZh:
      '⚠ 已开启：CLI 可执行任意 shell 命令。建议同时开启沙箱（Sandbox）或在「允许命令」中收紧白名单，避免误删除文件等高危操作。',
    advisoryRiskyEn:
      '⚠ Enabled: the CLI can run arbitrary shell commands. Strongly recommend pairing with Sandbox enabled or a strict allowlist to prevent destructive ops.',
    advisorySafeZh: '已禁用：CLI 不会执行任何 shell 命令，最高安全性但损失自动化能力。',
    advisorySafeEn: 'Disabled: the CLI cannot run shell commands — safest but loses automation power.',
  },
  'claude.permissions.allowRead': {
    labelZh: '允许读取文件',
    labelEn: 'Allow Read',
    descZh: '允许读取本机文件（用于代码理解、上下文加载等）。',
    descEn: 'Permit reading local files (for code understanding and context).',
    defaultValue: true,
  },
  'claude.permissions.allowWrite': {
    labelZh: '允许写入/修改文件',
    labelEn: 'Allow Write',
    descZh: '允许 Claude 直接修改/创建文件。关闭后只能"建议改动"。',
    descEn: 'Permit Claude to modify/create files. When off, Claude can only "suggest" diffs.',
    defaultValue: true,
    securityRole: 'risky-when-true',
    advisoryRiskyZh:
      '⚠ 已开启：CLI 可直接修改你的代码文件。建议在 git 已提交且工作区干净的状态下使用，便于回滚。',
    advisoryRiskyEn:
      '⚠ Enabled: the CLI can modify code files directly. Recommend committing your work and keeping a clean tree so you can revert easily.',
  },
  'claude.permissions.allowWebFetch': {
    labelZh: '允许访问网络',
    labelEn: 'Allow Web Fetch',
    descZh: '允许 Claude 主动发起 HTTP 请求（抓取文档、查 API 等）。',
    descEn: 'Permit Claude to make outbound HTTP requests (fetch docs, query APIs).',
    defaultValue: false,
    securityRole: 'risky-when-true',
    advisoryRiskyZh:
      '⚠ 已开启：Claude 可访问任意外网，包括读取你 prompt 里的链接。注意不要在敏感对话里粘贴未知 URL，避免被诱导拉取恶意内容（prompt injection）。',
    advisoryRiskyEn:
      '⚠ Enabled: Claude can fetch any URL — including ones embedded in your prompt. Be cautious about pasting untrusted links to avoid prompt-injection attacks.',
    advisorySafeZh: '已禁用：CLI 不能联网抓取，安全但缺失实时信息查询能力。',
    advisorySafeEn: 'Disabled: no outbound fetches — safer but loses live info lookup.',
  },
  'claude.permissions.trustedDirectories': {
    labelZh: '受信任目录',
    labelEn: 'Trusted Directories',
    descZh: '在这些目录下，CLI 会跳过危险操作的二次确认。建议只填项目目录，避免整盘授权。',
    descEn: 'Directories where the CLI skips secondary confirmations for risky ops. Recommend project dirs only — never the whole drive.',
  },
  'claude.permissions.allowedBashCommands': {
    labelZh: '允许的 Bash 命令（白名单）',
    labelEn: 'Allowed Bash Commands',
    descZh: 'Glob 模式。配了白名单后，只有这里的命令能直接运行；其他需确认。',
    descEn: 'Glob patterns. When set, only commands matching these run without prompting.',
  },
  'claude.permissions.deniedBashCommands': {
    labelZh: '禁止的 Bash 命令（黑名单）',
    labelEn: 'Denied Bash Commands',
    descZh: 'Glob 模式。这里列出的命令永远不会被执行，建议至少加上 `rm -rf /*`、`sudo *`。',
    descEn: 'Glob patterns. Commands matching are blocked outright — recommend at minimum `rm -rf /*`, `sudo *`.',
  },

  // ─── Claude · Sandbox ─────────────────────────────────────────────
  'claude.sandbox.enabled': {
    labelZh: '启用沙箱',
    labelEn: 'Enable Sandbox',
    descZh: '在隔离环境（Docker/WSL）中执行命令。强烈推荐与「允许 Bash」搭配使用。',
    descEn: 'Run shell commands inside an isolated environment (Docker/WSL). Strongly recommended together with Allow Bash.',
    defaultValue: false,
    securityRole: 'safe-when-true',
    advisorySafeZh: '✓ 沙箱已启用：shell 命令在隔离环境中执行，即使误删文件也只影响沙箱。',
    advisorySafeEn: '✓ Sandbox active: shell commands run in isolation — even destructive ops only affect the sandbox.',
    advisoryRiskyZh:
      '⚠ 沙箱未启用且 Bash 已开。建议至少选 docker / wsl 之一，否则 CLI 可在你主机上执行任意命令。',
    advisoryRiskyEn:
      '⚠ Sandbox off while Bash is enabled. Recommend selecting docker or wsl — otherwise the CLI can run arbitrary commands on your host.',
  },
  'claude.sandbox.type': {
    labelZh: '沙箱类型',
    labelEn: 'Sandbox Type',
    descZh: 'docker：跨平台、隔离强；wsl：仅 Windows，启动快、性能高。',
    descEn: 'docker — cross-platform, strong isolation; wsl — Windows-only, faster startup.',
    defaultValue: 'none',
  },
  'claude.sandbox.dockerImage': {
    labelZh: 'Docker 镜像',
    labelEn: 'Docker Image',
    descZh: '沙箱基础镜像。常用 `ubuntu:22.04` 或自建带工具链的镜像。',
    descEn: 'Base image for the sandbox. Common: `ubuntu:22.04` or a custom toolchain image.',
  },

  // ─── Claude · Advanced ────────────────────────────────────────────
  'claude.advanced.verbose': {
    labelZh: '详细日志',
    labelEn: 'Verbose Logging',
    descZh: '打印详细调用日志，便于排错；正常使用建议关闭以减少干扰。',
    descEn: 'Print verbose logs for debugging. Disable in normal use to reduce noise.',
    defaultValue: false,
  },
  'claude.advanced.disableTelemetry': {
    labelZh: '禁用遥测',
    labelEn: 'Disable Telemetry',
    descZh: '关闭官方使用统计上报。隐私敏感场景建议开启。',
    descEn: 'Stops official usage telemetry from being reported. Recommended for privacy-sensitive setups.',
    defaultValue: false,
    securityRole: 'safe-when-true',
    advisorySafeZh: '✓ 遥测已关闭，CLI 使用数据不会上报到 Anthropic。',
    advisorySafeEn: '✓ Telemetry off — usage data is not reported back to Anthropic.',
  },
  'claude.advanced.experimentalFeatures': {
    labelZh: '启用实验特性',
    labelEn: 'Experimental Features',
    descZh: '开启未稳定的预览功能。可能不稳定或有破坏性变更。',
    descEn: 'Enables preview/unstable features. May break or change without notice.',
    defaultValue: false,
    securityRole: 'risky-when-true',
    advisoryRiskyZh: '⚠ 实验特性默认关闭。开启后可能影响稳定性，遇到问题先关掉再排查。',
    advisoryRiskyEn: '⚠ Experimental features are off by default. If you hit issues, disable this first before debugging.',
  },
  'claude.advanced.apiEndpoint': {
    labelZh: '自定义 API 端点',
    labelEn: 'API Endpoint',
    descZh: '指向 Lurus Switch 本地网关或第三方代理的 URL。留空则使用 Anthropic 官方端点。',
    descEn: 'URL to your Lurus Switch local gateway or a third-party relay. Leave empty to use Anthropic\'s official endpoint.',
  },
  'claude.advanced.timeout': {
    labelZh: '请求超时（秒）',
    labelEn: 'Timeout (s)',
    descZh: '单次 API 请求的最大等待时间。超长上下文/思考模式建议提高到 600 以上。',
    descEn: 'Max wait time per API request. Raise above 600 for very long contexts or extended thinking.',
    defaultValue: 300,
  },

  // ─── Codex · Core ─────────────────────────────────────────────────
  'codex.model': {
    labelZh: '默认模型',
    labelEn: 'Default Model',
    descZh: 'OpenAI 模型 ID。o4-mini 性价比高，o3 推理强；GPT-4o 仍是通用首选。',
    descEn: 'OpenAI model ID. o4-mini is cost-efficient, o3 is best for reasoning, GPT-4o is the safe default.',
    defaultValue: 'o4-mini',
  },
  'codex.apiKey': {
    labelZh: 'API 密钥',
    labelEn: 'API Key',
    descZh: 'OpenAI API key。留空则使用环境变量 OPENAI_API_KEY。',
    descEn: 'OpenAI API key. Leave empty to inherit from OPENAI_API_KEY.',
  },
  'codex.approvalMode': {
    labelZh: '审批模式',
    labelEn: 'Approval Mode',
    descZh: 'on-failure：失败才请求确认；unless-allow-listed：白名单外都问；never：从不询问（最危险，谨慎选）。',
    descEn: 'on-failure asks only after errors; unless-allow-listed asks for non-listed ops; never skips all prompts (use with caution).',
    defaultValue: 'on-failure',
  },

  // ─── Codex · Provider ─────────────────────────────────────────────
  'codex.provider.type': {
    labelZh: '服务商类型',
    labelEn: 'Provider Type',
    descZh: 'openai 直连官方；azure 走 Azure OpenAI；custom 走自定义代理（如 Lurus Switch 网关）。',
    descEn: 'openai talks to OpenAI directly; azure routes via Azure OpenAI; custom uses your own proxy (e.g. Lurus Switch gateway).',
    defaultValue: 'openai',
  },
  'codex.provider.baseUrl': {
    labelZh: '自定义 Base URL',
    labelEn: 'Custom Base URL',
    descZh: '指向你的代理或本地网关（如 http://localhost:port/v1）。仅 azure / custom 模式生效。',
    descEn: 'URL of your proxy or local gateway (e.g. http://localhost:port/v1). Only used for azure / custom.',
  },
  'codex.provider.azureDeployment': {
    labelZh: 'Azure 部署名',
    labelEn: 'Azure Deployment Name',
    descZh: '在 Azure OpenAI 中创建的 deployment 名称。',
    descEn: 'The deployment name configured in your Azure OpenAI resource.',
  },
  'codex.provider.azureApiVersion': {
    labelZh: 'Azure API 版本',
    labelEn: 'Azure API Version',
    descZh: '调用 Azure OpenAI 时附带的 api-version 参数，如 2024-02-01。',
    descEn: 'The api-version query parameter (e.g. 2024-02-01) when calling Azure OpenAI.',
  },

  // ─── Codex · Security ─────────────────────────────────────────────
  'codex.security.networkAccess': {
    labelZh: '网络访问策略',
    labelEn: 'Network Access',
    descZh: 'full：完全联网；restricted：仅允许已配置主机；none：完全离线。',
    descEn: 'full lets the CLI hit any host; restricted limits to configured hosts; none disables network entirely.',
    defaultValue: 'full',
  },
  'codex.security.commandExecution.enabled': {
    labelZh: '允许执行命令',
    labelEn: 'Allow Command Execution',
    descZh: '允许 Codex 在你的 shell 中运行命令。关闭后只能"建议"。',
    descEn: 'Permit Codex to execute shell commands. When off, it can only suggest commands.',
    defaultValue: true,
    securityRole: 'risky-when-true',
    advisoryRiskyZh: '⚠ 已开启：建议同时启用沙箱或在"允许命令"中收紧白名单。',
    advisoryRiskyEn: '⚠ Enabled — recommend pairing with sandbox or a strict allowlist below.',
  },
  'codex.security.commandExecution.allowedCommands': {
    labelZh: '允许的命令（白名单）',
    labelEn: 'Allowed Commands',
    descZh: 'Glob 模式。配了白名单后，只有这里的命令免询问。',
    descEn: 'Glob patterns. When set, only matching commands skip the approval prompt.',
  },
  'codex.security.commandExecution.deniedCommands': {
    labelZh: '禁止的命令（黑名单）',
    labelEn: 'Denied Commands',
    descZh: 'Glob 模式。匹配的命令永远不会执行。建议至少加上 `rm -rf /*`、`sudo *`。',
    descEn: 'Glob patterns. Matches are blocked outright. Recommend at minimum `rm -rf /*`, `sudo *`.',
  },
  'codex.security.fileAccess.allowedDirs': {
    labelZh: '允许的文件目录',
    labelEn: 'Allowed Directories',
    descZh: 'CLI 仅能读写这些目录。建议设为项目目录，避免整盘授权。',
    descEn: 'Restrict file ops to these directories. Recommend project dirs only — never the whole drive.',
  },
  'codex.security.fileAccess.readOnlyDirs': {
    labelZh: '只读目录',
    labelEn: 'Read-Only Directories',
    descZh: '可读不可写。常用：/etc, ~/.ssh 等敏感配置。',
    descEn: 'Readable but not writable. Common: /etc, ~/.ssh and other sensitive configs.',
  },
  'codex.security.fileAccess.deniedPatterns': {
    labelZh: '禁止的文件模式',
    labelEn: 'Denied Patterns',
    descZh: 'Glob。匹配的文件不可读不可写。强烈推荐加 `*.env`、`*.pem`、`id_rsa*` 等密钥文件。',
    descEn: 'Glob patterns. Matched files are blocked from any access. Strongly recommend `*.env`, `*.pem`, `id_rsa*`.',
  },

  // ─── Codex · MCP / History / Sandbox ──────────────────────────────
  'codex.mcp.enabled': {
    labelZh: '启用 MCP',
    labelEn: 'Enable MCP',
    descZh: '开启 Model Context Protocol 支持，可注入自定义工具。',
    descEn: 'Enable Model Context Protocol — lets you plug in custom tools.',
    defaultValue: false,
  },
  'codex.history.enabled': {
    labelZh: '保存对话历史',
    labelEn: 'Save Conversation History',
    descZh: '将每次对话写入本地文件，便于回顾。包含完整 prompt 与回复，注意敏感信息。',
    descEn: 'Persists every conversation locally. Includes full prompts/responses — careful with sensitive info.',
    defaultValue: true,
  },
  'codex.history.maxEntries': {
    labelZh: '历史最大条数',
    labelEn: 'Max History Entries',
    descZh: '超过后从最旧的开始删除。',
    descEn: 'Older entries are pruned once this cap is hit.',
    defaultValue: 1000,
  },
  'codex.sandbox.enabled': {
    labelZh: '启用沙箱',
    labelEn: 'Enable Sandbox',
    descZh: '在隔离环境中执行命令。强烈推荐与"允许执行命令"搭配。',
    descEn: 'Run shell commands in isolation. Strongly recommended together with command execution.',
    defaultValue: false,
    securityRole: 'safe-when-true',
    advisorySafeZh: '✓ 沙箱已启用：命令在隔离环境运行，主机文件不受影响。',
    advisorySafeEn: '✓ Sandbox active — commands run in isolation, host files untouched.',
  },
  'codex.sandbox.type': {
    labelZh: '沙箱类型',
    labelEn: 'Sandbox Type',
    descZh: 'docker：跨平台、隔离强。Codex 目前不支持 wsl 沙箱。',
    descEn: 'docker offers cross-platform strong isolation. wsl is not supported for Codex sandbox.',
    defaultValue: 'none',
  },

  // ─── Gemini · Core ────────────────────────────────────────────────
  'gemini.model': {
    labelZh: '默认模型',
    labelEn: 'Default Model',
    descZh: 'Gemini 模型 ID。2.5 Flash 速度快、成本低；2.5 Pro 推理强但慢且贵。',
    descEn: 'Gemini model ID. 2.5 Flash is fast and cheap; 2.5 Pro is best for reasoning but slower and pricier.',
    defaultValue: 'gemini-2.5-flash',
  },
  'gemini.projectId': {
    labelZh: 'GCP 项目 ID',
    labelEn: 'GCP Project ID',
    descZh: '使用 OAuth / Service Account / ADC 时必填。API Key 模式不需要。',
    descEn: 'Required for OAuth / Service Account / ADC auth. Not needed for API key mode.',
  },

  // ─── Gemini · Auth ────────────────────────────────────────────────
  'gemini.auth.type': {
    labelZh: '鉴权方式',
    labelEn: 'Auth Type',
    descZh: 'api_key：最简单，适合个人；oauth：浏览器登录；service_account：CI/CD；adc：用 gcloud 默认凭据。',
    descEn: 'api_key is simplest for solo use; oauth uses browser login; service_account fits CI/CD; adc uses gcloud default creds.',
    defaultValue: 'api_key',
  },
  'gemini.apiKey': {
    labelZh: 'API 密钥',
    labelEn: 'API Key',
    descZh: 'Google AI Studio 的 API key（AIza... 开头）。仅 api_key 模式使用。',
    descEn: 'API key from Google AI Studio (starts with AIza...). Used only in api_key mode.',
  },
  'gemini.auth.oauthClientId': {
    labelZh: 'OAuth Client ID',
    labelEn: 'OAuth Client ID',
    descZh: 'GCP Console 创建的 OAuth Client ID。',
    descEn: 'OAuth Client ID created in GCP Console.',
  },
  'gemini.auth.serviceAccountPath': {
    labelZh: '服务账号 JSON 路径',
    labelEn: 'Service Account JSON Path',
    descZh: 'GCP 服务账号 key 文件本地路径。注意保管，泄露等于账号被盗。',
    descEn: 'Local path to the GCP service account key file. Treat as a credential — leaking it compromises the account.',
  },

  // ─── Gemini · Behavior ────────────────────────────────────────────
  'gemini.behavior.sandbox': {
    labelZh: '启用沙箱',
    labelEn: 'Enable Sandbox',
    descZh: '在隔离环境中运行工具。建议开启，尤其是 YOLO 模式下。',
    descEn: 'Run tools in isolation. Recommended especially with YOLO mode on.',
    defaultValue: false,
    securityRole: 'safe-when-true',
    advisorySafeZh: '✓ 沙箱已启用：工具调用在隔离环境运行。',
    advisorySafeEn: '✓ Sandbox active — tool calls run in isolation.',
  },
  'gemini.behavior.yoloMode': {
    labelZh: 'YOLO 模式',
    labelEn: 'YOLO Mode',
    descZh: '所有工具调用自动批准，不再询问。极度危险——只在沙箱内或一次性脚本里用。',
    descEn: 'Auto-approves every tool call. Extremely dangerous — only safe inside a sandbox or for throwaway scripts.',
    defaultValue: false,
    securityRole: 'risky-when-true',
    advisoryRiskyZh: '⚠ YOLO 模式已开启：CLI 不会就任何操作征求你的同意。务必同时开沙箱。',
    advisoryRiskyEn: '⚠ YOLO mode on — the CLI will not ask for confirmation on any action. Must pair with sandbox.',
  },
  'gemini.behavior.autoApprove': {
    labelZh: '自动批准模式',
    labelEn: 'Auto-Approve Patterns',
    descZh: '匹配到的工具调用免询问。比 YOLO 安全：只放行特定低风险工具。',
    descEn: 'Tool calls matching these patterns skip the prompt. Safer than YOLO — limits the auto-approve list.',
  },
  'gemini.behavior.maxFileSize': {
    labelZh: '最大文件大小（字节）',
    labelEn: 'Max File Size (bytes)',
    descZh: '超过该大小的文件不读取。0 表示不限。',
    descEn: 'Files larger than this are not read. 0 means unlimited.',
    defaultValue: 0,
  },
  'gemini.behavior.allowedExtensions': {
    labelZh: '允许的文件扩展名',
    labelEn: 'Allowed Extensions',
    descZh: '只读取这些扩展名的文件。空表示全允许。',
    descEn: 'Restrict reads to these extensions. Empty means allow all.',
  },

  // ─── Gemini · Instructions ────────────────────────────────────────
  'gemini.instructions.projectDescription': {
    labelZh: '项目描述',
    labelEn: 'Project Description',
    descZh: '一段话简述项目用途，作为系统提示注入。',
    descEn: 'One-paragraph project summary, injected as a system prompt.',
  },
  'gemini.instructions.techStack': {
    labelZh: '技术栈',
    labelEn: 'Tech Stack',
    descZh: '主要语言/框架，例如：Go、React、PostgreSQL。',
    descEn: 'Main languages/frameworks, e.g. Go, React, PostgreSQL.',
  },
  'gemini.instructions.codeStyle': {
    labelZh: '代码风格',
    labelEn: 'Code Style',
    descZh: '团队代码规范，例如：Google Go style、2 空格缩进。',
    descEn: 'Team code conventions, e.g. Google Go style, 2-space indent.',
  },
  'gemini.instructions.customRules': {
    labelZh: '自定义规则',
    labelEn: 'Custom Rules',
    descZh: '一行一条的项目规约，例如"必须写测试"。',
    descEn: 'One-line rules per item, e.g. "always write tests".',
  },

  // ─── Gemini · Display / Advanced ──────────────────────────────────
  'gemini.display.theme': {
    labelZh: '主题',
    labelEn: 'Theme',
    descZh: 'CLI 输出的配色：dark / light / system 跟随系统。',
    descEn: 'CLI output palette: dark / light / system (follows OS).',
    defaultValue: 'dark',
  },
  'gemini.display.syntaxHighlight': {
    labelZh: '语法高亮',
    labelEn: 'Syntax Highlighting',
    descZh: '在终端高亮显示代码块。',
    descEn: 'Highlight code blocks in the terminal.',
    defaultValue: true,
  },
  'gemini.display.markdownRender': {
    labelZh: 'Markdown 渲染',
    labelEn: 'Markdown Rendering',
    descZh: '在终端渲染加粗/列表/代码块等 Markdown 格式。',
    descEn: 'Render Markdown (bold/lists/code) inside the terminal.',
    defaultValue: true,
  },
  'gemini.advanced.apiEndpoint': {
    labelZh: '自定义 API 端点',
    labelEn: 'API Endpoint',
    descZh: '指向 Lurus Switch 本地网关或第三方代理。留空使用 Google 官方端点。',
    descEn: 'URL of your Lurus Switch local gateway or a third-party relay. Leave empty for Google\'s official endpoint.',
  },

  // ─── ZeroClaw ─────────────────────────────────────────────────────
  'zeroclaw.provider.type': {
    labelZh: '服务商',
    labelEn: 'Provider',
    descZh: 'anthropic / openai / custom。custom 需配 base_url 指向你的代理。',
    descEn: 'anthropic / openai / custom. custom requires base_url pointing to your proxy.',
    defaultValue: 'anthropic',
  },
  'zeroclaw.provider.apiKey': {
    labelZh: 'API 密钥',
    labelEn: 'API Key',
    descZh: '所选服务商对应的 API key。',
    descEn: 'API key for the selected provider.',
  },
  'zeroclaw.provider.model': {
    labelZh: '模型',
    labelEn: 'Model',
    descZh: '完整模型 ID，例如 claude-sonnet-4-20250514。',
    descEn: 'Full model ID, e.g. claude-sonnet-4-20250514.',
    defaultValue: 'claude-sonnet-4-20250514',
  },
  'zeroclaw.provider.baseUrl': {
    labelZh: 'Base URL',
    labelEn: 'Base URL',
    descZh: '可选。custom 模式必填；其他模式留空使用官方端点。',
    descEn: 'Optional. Required for custom; leave empty otherwise to use the official endpoint.',
  },
  'zeroclaw.gateway.port': {
    labelZh: 'Gateway 端口',
    labelEn: 'Gateway Port',
    descZh: 'ZeroClaw 本地网关监听端口。1024-65535。',
    descEn: 'Local port ZeroClaw listens on. Range 1024-65535.',
    defaultValue: 8765,
  },
  'zeroclaw.memory.backend': {
    labelZh: '记忆后端',
    labelEn: 'Memory Backend',
    descZh: 'sqlite：本地持久化；in-memory：进程退出即清。',
    descEn: 'sqlite persists locally; in-memory clears when the process exits.',
    defaultValue: 'sqlite',
  },
  'zeroclaw.security.sandbox': {
    labelZh: '启用沙箱',
    labelEn: 'Enable Sandbox',
    descZh: '工具调用在隔离环境中执行。',
    descEn: 'Run tool calls in an isolated environment.',
    defaultValue: false,
    securityRole: 'safe-when-true',
    advisorySafeZh: '✓ 沙箱已启用。',
    advisorySafeEn: '✓ Sandbox active.',
  },
  'zeroclaw.security.allowExec': {
    labelZh: '允许执行命令',
    labelEn: 'Allow Command Execution',
    descZh: '允许 ZeroClaw 在你 shell 里运行命令。建议同时开启沙箱。',
    descEn: 'Permit ZeroClaw to run shell commands. Recommend pairing with sandbox.',
    defaultValue: true,
    securityRole: 'risky-when-true',
    advisoryRiskyZh: '⚠ 已开启：CLI 可执行任意 shell 命令。强烈建议启用沙箱。',
    advisoryRiskyEn: '⚠ Enabled — the CLI can run arbitrary shell commands. Strongly recommend enabling sandbox.',
  },

  // ─── OpenClaw ─────────────────────────────────────────────────────
  'openclaw.provider.type': {
    labelZh: '服务商',
    labelEn: 'Provider',
    descZh: 'anthropic / openai / custom。',
    descEn: 'anthropic / openai / custom.',
    defaultValue: 'anthropic',
  },
  'openclaw.provider.apiKey': {
    labelZh: 'API 密钥',
    labelEn: 'API Key',
    descZh: '所选服务商对应的 API key。',
    descEn: 'API key for the selected provider.',
  },
  'openclaw.provider.model': {
    labelZh: '模型',
    labelEn: 'Model',
    descZh: '完整模型 ID。',
    descEn: 'Full model ID.',
    defaultValue: 'claude-sonnet-4-20250514',
  },
  'openclaw.provider.baseUrl': {
    labelZh: 'Base URL',
    labelEn: 'Base URL',
    descZh: '可选。custom 模式必填。',
    descEn: 'Optional. Required for custom.',
  },
  'openclaw.gateway.port': {
    labelZh: 'Gateway 端口',
    labelEn: 'Gateway Port',
    descZh: 'OpenClaw 本地监听端口。',
    descEn: 'Local port OpenClaw listens on.',
    defaultValue: 18789,
  },
  'openclaw.channels.dmPolicy': {
    labelZh: '私聊策略',
    labelEn: 'DM Policy',
    descZh: 'all：任何人；registered：仅注册用户；none：不接受私聊。生产建议至少 registered。',
    descEn: 'all permits any user; registered limits to known users; none blocks DMs. Use at least registered in production.',
    defaultValue: 'all',
  },
  'openclaw.skills.enabled': {
    labelZh: '启用的能力',
    labelEn: 'Enabled Skills',
    descZh: '允许机器人使用的工具集。code-exec / file-read 涉及主机访问，谨慎开启。',
    descEn: 'Tools the bot may use. code-exec / file-read touch the host — enable with care.',
  },
}

// getMeta returns the metadata for a dotted field path or null when no
// entry is registered. Callers should fall back to rendering the raw input
// without the bilingual chrome when nil.
export function getMeta(path: string): FieldMeta | null {
  return FIELD_META[path] ?? null
}

// isEmptyValue is the canonical "user hasn't provided a value" check used
// across the field-status calculator. Treats undefined/null, empty string,
// empty array, and empty map as empty. Numbers (including 0) and booleans
// are NEVER empty — 0/false are considered explicit values.
function isEmptyValue(value: unknown): boolean {
  if (value === undefined || value === null) return true
  if (typeof value === 'string') return value === ''
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'object') return Object.keys(value as object).length === 0
  return false
}

export function getStatus(meta: FieldMeta | null, value: unknown): FieldStatus {
  if (!meta) {
    return isEmptyValue(value) ? 'unset' : 'set'
  }
  if (isEmptyValue(value)) {
    if (meta.required) return 'unset'
    if (meta.defaultValue !== undefined) return 'default'
    return 'unset'
  }
  // Security verdict short-circuits — risky/safe takes precedence over
  // the default-vs-set chrome since the user cares more about safety
  // than novelty.
  if (meta.securityRole) {
    const r = meta.securityRole
    const isRisky =
      (r === 'risky-when-true' && value === true) ||
      (r === 'risky-when-false' && value === false)
    if (isRisky) return 'risky'
    const isSafe =
      (r === 'safe-when-true' && value === true) ||
      (r === 'safe-when-false' && value === false)
    if (isSafe) return 'safe'
  }
  if (meta.defaultValue !== undefined && value === meta.defaultValue) {
    return 'default'
  }
  return 'set'
}
