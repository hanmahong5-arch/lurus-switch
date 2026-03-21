export namespace analytics {
	
	export class UsageReport {
	    toolActions: Record<string, any>;
	    dailyActive: Record<string, number>;
	    configCounts: Record<string, number>;
	    promptCount: number;
	
	    static createFrom(source: any = {}) {
	        return new UsageReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolActions = source["toolActions"];
	        this.dailyActive = source["dailyActive"];
	        this.configCounts = source["configCounts"];
	        this.promptCount = source["promptCount"];
	    }
	}

}

export namespace appconfig {
	
	export class AppSettings {
	    theme: string;
	    language: string;
	    autoUpdate: boolean;
	    editorFontSize: number;
	    startupPage: string;
	    onboardingCompleted: boolean;
	    appMode: string;
	    userLevel: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.autoUpdate = source["autoUpdate"];
	        this.editorFontSize = source["editorFontSize"];
	        this.startupPage = source["startupPage"];
	        this.onboardingCompleted = source["onboardingCompleted"];
	        this.appMode = source["appMode"];
	        this.userLevel = source["userLevel"];
	    }
	}

}

export namespace appreg {
	
	export class App {
	    id: string;
	    name: string;
	    kind: string;
	    tier: number;
	    token: string;
	    icon: string;
	    description: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    lastSeenAt?: any;
	    connected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new App(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.tier = source["tier"];
	        this.token = source["token"];
	        this.icon = source["icon"];
	        this.description = source["description"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.lastSeenAt = this.convertValues(source["lastSeenAt"], null);
	        this.connected = source["connected"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace billing {
	
	export class ConfigPreset {
	    id: string;
	    tool: string;
	    name: string;
	    description: string;
	    category: string;
	    config_json: Record<string, any>;
	    is_official: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tool = source["tool"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.config_json = source["config_json"];
	        this.is_official = source["is_official"];
	    }
	}
	export class IdentityOverview {
	    // Go type: struct { ID int64 "json:\"id\""; LurusID string "json:\"lurus_id\""; DisplayName string "json:\"display_name\""; AvatarURL string "json:\"avatar_url\"" }
	    account: any;
	    // Go type: struct { Level int16 "json:\"level\""; LevelName string "json:\"level_name\""; LevelEN string "json:\"level_en\""; Points int64 "json:\"points\""; LevelExpiresAt string "json:\"level_expires_at,omitempty\"" }
	    vip: any;
	    // Go type: struct { Balance float64 "json:\"balance\""; Frozen float64 "json:\"frozen\"" }
	    wallet: any;
	    // Go type: struct { ProductID string "json:\"product_id\""; PlanCode string "json:\"plan_code\""; Status string "json:\"status\""; ExpiresAt string "json:\"expires_at,omitempty\""; AutoRenew bool "json:\"auto_renew\"" }
	    subscription?: any;
	    topup_url: string;
	
	    static createFrom(source: any = {}) {
	        return new IdentityOverview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account = this.convertValues(source["account"], Object);
	        this.vip = this.convertValues(source["vip"], Object);
	        this.wallet = this.convertValues(source["wallet"], Object);
	        this.subscription = this.convertValues(source["subscription"], Object);
	        this.topup_url = source["topup_url"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PaymentResult {
	    trade_no: string;
	    payment_url: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new PaymentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trade_no = source["trade_no"];
	        this.payment_url = source["payment_url"];
	        this.message = source["message"];
	    }
	}
	export class QuotaSummary {
	    quota: number;
	    used_quota: number;
	    remaining_quota: number;
	    daily_quota: number;
	    daily_used: number;
	    username: string;
	
	    static createFrom(source: any = {}) {
	        return new QuotaSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.quota = source["quota"];
	        this.used_quota = source["used_quota"];
	        this.remaining_quota = source["remaining_quota"];
	        this.daily_quota = source["daily_quota"];
	        this.daily_used = source["daily_used"];
	        this.username = source["username"];
	    }
	}
	export class SubscriptionInfo {
	    id: number;
	    plan_code: string;
	    plan_name: string;
	    status: string;
	    expires_at: string;
	    auto_renew: boolean;
	    daily_quota: number;
	    total_quota: number;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.plan_code = source["plan_code"];
	        this.plan_name = source["plan_name"];
	        this.status = source["status"];
	        this.expires_at = source["expires_at"];
	        this.auto_renew = source["auto_renew"];
	        this.daily_quota = source["daily_quota"];
	        this.total_quota = source["total_quota"];
	    }
	}
	export class SubscriptionPlan {
	    code: string;
	    name: string;
	    currency: string;
	    duration: string;
	    price: number;
	    daily_quota: number;
	    total_quota: number;
	    features: string[];
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.currency = source["currency"];
	        this.duration = source["duration"];
	        this.price = source["price"];
	        this.daily_quota = source["daily_quota"];
	        this.total_quota = source["total_quota"];
	        this.features = source["features"];
	    }
	}
	export class TopUpInfo {
	    pay_methods: any[];
	    amount_options: number[];
	    min_topup: number;
	    discount: number;
	
	    static createFrom(source: any = {}) {
	        return new TopUpInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pay_methods = source["pay_methods"];
	        this.amount_options = source["amount_options"];
	        this.min_topup = source["min_topup"];
	        this.discount = source["discount"];
	    }
	}
	export class UserInfo {
	    quota: number;
	    used_quota: number;
	    remaining_quota: number;
	    daily_quota: number;
	    daily_used: number;
	    group: string;
	    username: string;
	    display_name: string;
	    aff_code: string;
	    subscription?: SubscriptionInfo;
	
	    static createFrom(source: any = {}) {
	        return new UserInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.quota = source["quota"];
	        this.used_quota = source["used_quota"];
	        this.remaining_quota = source["remaining_quota"];
	        this.daily_quota = source["daily_quota"];
	        this.daily_used = source["daily_used"];
	        this.group = source["group"];
	        this.username = source["username"];
	        this.display_name = source["display_name"];
	        this.aff_code = source["aff_code"];
	        this.subscription = this.convertValues(source["subscription"], SubscriptionInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace config {
	
	export class ClaudeAdvanced {
	    verbose?: boolean;
	    disableTelemetry?: boolean;
	    apiEndpoint?: string;
	    timeout?: number;
	    experimentalFeatures?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ClaudeAdvanced(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.verbose = source["verbose"];
	        this.disableTelemetry = source["disableTelemetry"];
	        this.apiEndpoint = source["apiEndpoint"];
	        this.timeout = source["timeout"];
	        this.experimentalFeatures = source["experimentalFeatures"];
	    }
	}
	export class SandboxMount {
	    source: string;
	    destination: string;
	    readOnly?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SandboxMount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.destination = source["destination"];
	        this.readOnly = source["readOnly"];
	    }
	}
	export class ClaudeSandbox {
	    enabled?: boolean;
	    type?: string;
	    dockerImage?: string;
	    mounts?: SandboxMount[];
	
	    static createFrom(source: any = {}) {
	        return new ClaudeSandbox(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.type = source["type"];
	        this.dockerImage = source["dockerImage"];
	        this.mounts = this.convertValues(source["mounts"], SandboxMount);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MCPServer {
	    command: string;
	    args?: string[];
	    env?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MCPServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	    }
	}
	export class ClaudePermissions {
	    allowBash?: boolean;
	    allowRead?: boolean;
	    allowWrite?: boolean;
	    allowWebFetch?: boolean;
	    trustedDirectories?: string[];
	    allowedBashCommands?: string[];
	    deniedBashCommands?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ClaudePermissions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowBash = source["allowBash"];
	        this.allowRead = source["allowRead"];
	        this.allowWrite = source["allowWrite"];
	        this.allowWebFetch = source["allowWebFetch"];
	        this.trustedDirectories = source["trustedDirectories"];
	        this.allowedBashCommands = source["allowedBashCommands"];
	        this.deniedBashCommands = source["deniedBashCommands"];
	    }
	}
	export class ClaudeConfig {
	    model?: string;
	    customInstructions?: string;
	    apiKey?: string;
	    maxTokens?: number;
	    permissions?: ClaudePermissions;
	    mcpServers?: Record<string, MCPServer>;
	    sandbox?: ClaudeSandbox;
	    advanced?: ClaudeAdvanced;
	
	    static createFrom(source: any = {}) {
	        return new ClaudeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.customInstructions = source["customInstructions"];
	        this.apiKey = source["apiKey"];
	        this.maxTokens = source["maxTokens"];
	        this.permissions = this.convertValues(source["permissions"], ClaudePermissions);
	        this.mcpServers = this.convertValues(source["mcpServers"], MCPServer, true);
	        this.sandbox = this.convertValues(source["sandbox"], ClaudeSandbox);
	        this.advanced = this.convertValues(source["advanced"], ClaudeAdvanced);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class CodexCommandExecution {
	    enabled: boolean;
	    allowedCommands?: string[];
	    deniedCommands?: string[];
	
	    static createFrom(source: any = {}) {
	        return new CodexCommandExecution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.allowedCommands = source["allowedCommands"];
	        this.deniedCommands = source["deniedCommands"];
	    }
	}
	export class CodexHistory {
	    enabled: boolean;
	    filePath?: string;
	    maxEntries?: number;
	
	    static createFrom(source: any = {}) {
	        return new CodexHistory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.filePath = source["filePath"];
	        this.maxEntries = source["maxEntries"];
	    }
	}
	export class CodexSandbox {
	    enabled: boolean;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new CodexSandbox(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.type = source["type"];
	    }
	}
	export class CodexMCPServer {
	    name: string;
	    command: string;
	    args?: string[];
	    env?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new CodexMCPServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	    }
	}
	export class CodexMCP {
	    enabled: boolean;
	    servers?: CodexMCPServer[];
	
	    static createFrom(source: any = {}) {
	        return new CodexMCP(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.servers = this.convertValues(source["servers"], CodexMCPServer);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CodexFileAccess {
	    allowedDirs?: string[];
	    deniedPatterns?: string[];
	    readOnlyDirs?: string[];
	
	    static createFrom(source: any = {}) {
	        return new CodexFileAccess(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowedDirs = source["allowedDirs"];
	        this.deniedPatterns = source["deniedPatterns"];
	        this.readOnlyDirs = source["readOnlyDirs"];
	    }
	}
	export class CodexSecurity {
	    networkAccess: string;
	    fileAccess: CodexFileAccess;
	    commandExecution: CodexCommandExecution;
	
	    static createFrom(source: any = {}) {
	        return new CodexSecurity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.networkAccess = source["networkAccess"];
	        this.fileAccess = this.convertValues(source["fileAccess"], CodexFileAccess);
	        this.commandExecution = this.convertValues(source["commandExecution"], CodexCommandExecution);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CodexProvider {
	    type: string;
	    baseUrl?: string;
	    azureDeployment?: string;
	    azureApiVersion?: string;
	
	    static createFrom(source: any = {}) {
	        return new CodexProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.baseUrl = source["baseUrl"];
	        this.azureDeployment = source["azureDeployment"];
	        this.azureApiVersion = source["azureApiVersion"];
	    }
	}
	export class CodexConfig {
	    model: string;
	    apiKey: string;
	    approvalMode: string;
	    provider: CodexProvider;
	    security: CodexSecurity;
	    mcp: CodexMCP;
	    sandbox: CodexSandbox;
	    history: CodexHistory;
	
	    static createFrom(source: any = {}) {
	        return new CodexConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.apiKey = source["apiKey"];
	        this.approvalMode = source["approvalMode"];
	        this.provider = this.convertValues(source["provider"], CodexProvider);
	        this.security = this.convertValues(source["security"], CodexSecurity);
	        this.mcp = this.convertValues(source["mcp"], CodexMCP);
	        this.sandbox = this.convertValues(source["sandbox"], CodexSandbox);
	        this.history = this.convertValues(source["history"], CodexHistory);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	export class GeminiAdvanced {
	    apiEndpoint?: string;
	
	    static createFrom(source: any = {}) {
	        return new GeminiAdvanced(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiEndpoint = source["apiEndpoint"];
	    }
	}
	export class GeminiAuth {
	    type: string;
	    oauthClientId?: string;
	    serviceAccountPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new GeminiAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.oauthClientId = source["oauthClientId"];
	        this.serviceAccountPath = source["serviceAccountPath"];
	    }
	}
	export class GeminiBehavior {
	    sandbox: boolean;
	    autoApprove?: string[];
	    yoloMode: boolean;
	    maxFileSize?: number;
	    allowedExtensions?: string[];
	
	    static createFrom(source: any = {}) {
	        return new GeminiBehavior(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sandbox = source["sandbox"];
	        this.autoApprove = source["autoApprove"];
	        this.yoloMode = source["yoloMode"];
	        this.maxFileSize = source["maxFileSize"];
	        this.allowedExtensions = source["allowedExtensions"];
	    }
	}
	export class GeminiDisplay {
	    theme: string;
	    syntaxHighlight: boolean;
	    markdownRender: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GeminiDisplay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.syntaxHighlight = source["syntaxHighlight"];
	        this.markdownRender = source["markdownRender"];
	    }
	}
	export class GeminiInstructions {
	    projectDescription?: string;
	    techStack?: string;
	    codeStyle?: string;
	    customRules?: string[];
	    fileStructure?: string;
	    testingGuidelines?: string;
	
	    static createFrom(source: any = {}) {
	        return new GeminiInstructions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectDescription = source["projectDescription"];
	        this.techStack = source["techStack"];
	        this.codeStyle = source["codeStyle"];
	        this.customRules = source["customRules"];
	        this.fileStructure = source["fileStructure"];
	        this.testingGuidelines = source["testingGuidelines"];
	    }
	}
	export class GeminiConfig {
	    model: string;
	    apiKey: string;
	    projectId?: string;
	    auth: GeminiAuth;
	    behavior: GeminiBehavior;
	    instructions: GeminiInstructions;
	    display: GeminiDisplay;
	    advanced: GeminiAdvanced;
	
	    static createFrom(source: any = {}) {
	        return new GeminiConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.apiKey = source["apiKey"];
	        this.projectId = source["projectId"];
	        this.auth = this.convertValues(source["auth"], GeminiAuth);
	        this.behavior = this.convertValues(source["behavior"], GeminiBehavior);
	        this.instructions = this.convertValues(source["instructions"], GeminiInstructions);
	        this.display = this.convertValues(source["display"], GeminiDisplay);
	        this.advanced = this.convertValues(source["advanced"], GeminiAdvanced);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class NullClawAgentDefaults {
	    model_name: string;
	
	    static createFrom(source: any = {}) {
	        return new NullClawAgentDefaults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_name = source["model_name"];
	    }
	}
	export class NullClawAgentSettings {
	    defaults: NullClawAgentDefaults;
	
	    static createFrom(source: any = {}) {
	        return new NullClawAgentSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaults = this.convertValues(source["defaults"], NullClawAgentDefaults);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NullClawModel {
	    name: string;
	    api_base: string;
	    api_key: string;
	    model_name: string;
	
	    static createFrom(source: any = {}) {
	        return new NullClawModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.api_base = source["api_base"];
	        this.api_key = source["api_key"];
	        this.model_name = source["model_name"];
	    }
	}
	export class NullClawConfig {
	    model_list: NullClawModel[];
	    agents: NullClawAgentSettings;
	
	    static createFrom(source: any = {}) {
	        return new NullClawConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_list = this.convertValues(source["model_list"], NullClawModel);
	        this.agents = this.convertValues(source["agents"], NullClawAgentSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class OpenClawChannels {
	    dm_policy: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenClawChannels(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dm_policy = source["dm_policy"];
	    }
	}
	export class OpenClawSkills {
	    enabled: string[];
	
	    static createFrom(source: any = {}) {
	        return new OpenClawSkills(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	    }
	}
	export class OpenClawProvider {
	    type: string;
	    api_key: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenClawProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	    }
	}
	export class OpenClawGateway {
	    port: number;
	    auth_token: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenClawGateway(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.auth_token = source["auth_token"];
	    }
	}
	export class OpenClawConfig {
	    gateway: OpenClawGateway;
	    provider: OpenClawProvider;
	    channels: OpenClawChannels;
	    skills: OpenClawSkills;
	
	    static createFrom(source: any = {}) {
	        return new OpenClawConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gateway = this.convertValues(source["gateway"], OpenClawGateway);
	        this.provider = this.convertValues(source["provider"], OpenClawProvider);
	        this.channels = this.convertValues(source["channels"], OpenClawChannels);
	        this.skills = this.convertValues(source["skills"], OpenClawSkills);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class PicoClawAgentDefaults {
	    model_name: string;
	
	    static createFrom(source: any = {}) {
	        return new PicoClawAgentDefaults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_name = source["model_name"];
	    }
	}
	export class PicoClawAgentSettings {
	    defaults: PicoClawAgentDefaults;
	
	    static createFrom(source: any = {}) {
	        return new PicoClawAgentSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaults = this.convertValues(source["defaults"], PicoClawAgentDefaults);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PicoClawModel {
	    name: string;
	    api_base: string;
	    api_key: string;
	    model_name: string;
	
	    static createFrom(source: any = {}) {
	        return new PicoClawModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.api_base = source["api_base"];
	        this.api_key = source["api_key"];
	        this.model_name = source["model_name"];
	    }
	}
	export class PicoClawConfig {
	    model_list: PicoClawModel[];
	    agents: PicoClawAgentSettings;
	
	    static createFrom(source: any = {}) {
	        return new PicoClawConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_list = this.convertValues(source["model_list"], PicoClawModel);
	        this.agents = this.convertValues(source["agents"], PicoClawAgentSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class ZeroClawSecurity {
	    sandbox: boolean;
	    audit_log: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ZeroClawSecurity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sandbox = source["sandbox"];
	        this.audit_log = source["audit_log"];
	    }
	}
	export class ZeroClawMemory {
	    backend: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ZeroClawMemory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backend = source["backend"];
	        this.path = source["path"];
	    }
	}
	export class ZeroClawGateway {
	    host: string;
	    port: number;
	    auth_token: string;
	
	    static createFrom(source: any = {}) {
	        return new ZeroClawGateway(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.auth_token = source["auth_token"];
	    }
	}
	export class ZeroClawProvider {
	    type: string;
	    api_key: string;
	    model: string;
	    base_url: string;
	
	    static createFrom(source: any = {}) {
	        return new ZeroClawProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	        this.base_url = source["base_url"];
	    }
	}
	export class ZeroClawConfig {
	    provider: ZeroClawProvider;
	    gateway: ZeroClawGateway;
	    memory: ZeroClawMemory;
	    security: ZeroClawSecurity;
	
	    static createFrom(source: any = {}) {
	        return new ZeroClawConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = this.convertValues(source["provider"], ZeroClawProvider);
	        this.gateway = this.convertValues(source["gateway"], ZeroClawGateway);
	        this.memory = this.convertValues(source["memory"], ZeroClawMemory);
	        this.security = this.convertValues(source["security"], ZeroClawSecurity);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	

}

export namespace docmgr {
	
	export class ContextFile {
	    tool: string;
	    scope: string;
	    path: string;
	    content: string;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContextFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.scope = source["scope"];
	        this.path = source["path"];
	        this.content = source["content"];
	        this.exists = source["exists"];
	    }
	}

}

export namespace envmgr {
	
	export class KeyEntry {
	    tool: string;
	    key: string;
	    maskedValue: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.key = source["key"];
	        this.maskedValue = source["maskedValue"];
	        this.source = source["source"];
	    }
	}

}

export namespace gateway {
	
	export class Config {
	    port: number;
	    upstreamUrl: string;
	    userToken: string;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.upstreamUrl = source["upstreamUrl"];
	        this.userToken = source["userToken"];
	        this.autoStart = source["autoStart"];
	    }
	}
	export class Status {
	    running: boolean;
	    port: number;
	    url: string;
	    uptime: number;
	    totalRequests: number;
	    activeConns: number;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.url = source["url"];
	        this.uptime = source["uptime"];
	        this.totalRequests = source["totalRequests"];
	        this.activeConns = source["activeConns"];
	    }
	}

}

export namespace gy {
	
	export class GYProduct {
	    id: string;
	    name: string;
	    description: string;
	    kind: string;
	    launchUrl?: string;
	    downloadUrl?: string;
	    serviceUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new GYProduct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.kind = source["kind"];
	        this.launchUrl = source["launchUrl"];
	        this.downloadUrl = source["downloadUrl"];
	        this.serviceUrl = source["serviceUrl"];
	    }
	}
	export class GYStatus {
	    productId: string;
	    available: boolean;
	    latencyMs: number;
	    version?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new GYStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productId = source["productId"];
	        this.available = source["available"];
	        this.latencyMs = source["latencyMs"];
	        this.version = source["version"];
	        this.error = source["error"];
	    }
	}

}

export namespace healthscore {
	
	export class CategoryScore {
	    category: string;
	    score: number;
	    max: number;
	    label: string;
	    issues: string[];
	
	    static createFrom(source: any = {}) {
	        return new CategoryScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.score = source["score"];
	        this.max = source["max"];
	        this.label = source["label"];
	        this.issues = source["issues"];
	    }
	}
	export class Suggestion {
	    id: string;
	    priority: number;
	    title: string;
	    action: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new Suggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.priority = source["priority"];
	        this.title = source["title"];
	        this.action = source["action"];
	        this.target = source["target"];
	    }
	}
	export class ScoreReport {
	    totalScore: number;
	    maxScore: number;
	    categories: CategoryScore[];
	    suggestions: Suggestion[];
	
	    static createFrom(source: any = {}) {
	        return new ScoreReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalScore = source["totalScore"];
	        this.maxScore = source["maxScore"];
	        this.categories = this.convertValues(source["categories"], CategoryScore);
	        this.suggestions = this.convertValues(source["suggestions"], Suggestion);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace installer {
	
	export class RuntimeStatus {
	    id: string;
	    name: string;
	    installed: boolean;
	    version: string;
	    path: string;
	    required: boolean;
	    tools: string[];
	
	    static createFrom(source: any = {}) {
	        return new RuntimeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.required = source["required"];
	        this.tools = source["tools"];
	    }
	}
	export class DepCheckResult {
	    runtimes: RuntimeStatus[];
	    allMet: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DepCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtimes = this.convertValues(source["runtimes"], RuntimeStatus);
	        this.allMet = source["allMet"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DepInstallResult {
	    runtimeId: string;
	    success: boolean;
	    version: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DepInstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtimeId = source["runtimeId"];
	        this.success = source["success"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}
	export class InstallResult {
	    tool: string;
	    success: boolean;
	    version: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.success = source["success"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}

}

export namespace main {
	
	export class RuntimeDiagnostic {
	    id: string;
	    name: string;
	    installed: boolean;
	    version: string;
	    required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.required = source["required"];
	    }
	}
	export class ToolDiagnostic {
	    tool: string;
	    installed: boolean;
	    version: string;
	    path: string;
	    configExists: boolean;
	    healthStatus: string;
	    healthIssues: string[];
	    gatewayBound: boolean;
	    connected: boolean;
	    currentEndpoint: string;
	    currentModel: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.configExists = source["configExists"];
	        this.healthStatus = source["healthStatus"];
	        this.healthIssues = source["healthIssues"];
	        this.gatewayBound = source["gatewayBound"];
	        this.connected = source["connected"];
	        this.currentEndpoint = source["currentEndpoint"];
	        this.currentModel = source["currentModel"];
	    }
	}
	export class EnvironmentCheck {
	    tools: ToolDiagnostic[];
	    runtimes: RuntimeDiagnostic[];
	    gatewayRunning: boolean;
	    gatewayUrl: string;
	    allToolsBound: boolean;
	    installedCount: number;
	    boundCount: number;
	
	    static createFrom(source: any = {}) {
	        return new EnvironmentCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools = this.convertValues(source["tools"], ToolDiagnostic);
	        this.runtimes = this.convertValues(source["runtimes"], RuntimeDiagnostic);
	        this.gatewayRunning = source["gatewayRunning"];
	        this.gatewayUrl = source["gatewayUrl"];
	        this.allToolsBound = source["allToolsBound"];
	        this.installedCount = source["installedCount"];
	        this.boundCount = source["boundCount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolConfigResult {
	    tool: string;
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolConfigResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class FullSetupResult {
	    gatewayStarted: boolean;
	    snapshotsTaken: number;
	    configResults: ToolConfigResult[];
	    gatewayUrl: string;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new FullSetupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gatewayStarted = source["gatewayStarted"];
	        this.snapshotsTaken = source["snapshotsTaken"];
	        this.configResults = this.convertValues(source["configResults"], ToolConfigResult);
	        this.gatewayUrl = source["gatewayUrl"];
	        this.errors = source["errors"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModelCostBreakdown {
	    model: string;
	    tokensIn: number;
	    tokensOut: number;
	    inputRatio: number;
	    outputRatio: number;
	    costUSD: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelCostBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	        this.inputRatio = source["inputRatio"];
	        this.outputRatio = source["outputRatio"];
	        this.costUSD = source["costUSD"];
	    }
	}
	export class RequestLogEntry {
	    id: string;
	    timestamp: string;
	    appId: string;
	    model: string;
	    tokensIn: number;
	    tokensOut: number;
	    latencyMs: number;
	    statusCode: number;
	    cached: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = source["timestamp"];
	        this.appId = source["appId"];
	        this.model = source["model"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	        this.latencyMs = source["latencyMs"];
	        this.statusCode = source["statusCode"];
	        this.cached = source["cached"];
	        this.error = source["error"];
	    }
	}
	
	export class SystemInfo {
	    appVersion: string;
	    goos: string;
	    goarch: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appVersion = source["appVersion"];
	        this.goos = source["goos"];
	        this.goarch = source["goarch"];
	    }
	}
	
	
	export class ToolSnapshotInfo {
	    id: string;
	    tool: string;
	    label: string;
	    createdAt: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new ToolSnapshotInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tool = source["tool"];
	        this.label = source["label"];
	        this.createdAt = source["createdAt"];
	        this.size = source["size"];
	    }
	}
	export class UpstreamHealthResult {
	    reachable: boolean;
	    latencyMs: number;
	    statusCode: number;
	    endpoint: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpstreamHealthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reachable = source["reachable"];
	        this.latencyMs = source["latencyMs"];
	        this.statusCode = source["statusCode"];
	        this.endpoint = source["endpoint"];
	        this.error = source["error"];
	    }
	}
	export class UsageInsight {
	    totalCalls: number;
	    totalTokensIn: number;
	    totalTokensOut: number;
	    cacheHitRate: number;
	    rateLimitEvents: number;
	    errorEvents: number;
	    avgLatencyMs: number;
	    totalCostUSD: number;
	    modelCosts: ModelCostBreakdown[];
	
	    static createFrom(source: any = {}) {
	        return new UsageInsight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalCalls = source["totalCalls"];
	        this.totalTokensIn = source["totalTokensIn"];
	        this.totalTokensOut = source["totalTokensOut"];
	        this.cacheHitRate = source["cacheHitRate"];
	        this.rateLimitEvents = source["rateLimitEvents"];
	        this.errorEvents = source["errorEvents"];
	        this.avgLatencyMs = source["avgLatencyMs"];
	        this.totalCostUSD = source["totalCostUSD"];
	        this.modelCosts = this.convertValues(source["modelCosts"], ModelCostBreakdown);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace mcp {
	
	export class MCPServer {
	    name: string;
	    command?: string;
	    args?: string[];
	    env?: Record<string, string>;
	    url?: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.url = source["url"];
	        this.type = source["type"];
	    }
	}
	export class MCPPreset {
	    id: string;
	    name: string;
	    description: string;
	    server: MCPServer;
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new MCPPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.server = this.convertValues(source["server"], MCPServer);
	        this.tags = source["tags"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace metering {
	
	export class ActivityEntry {
	    timestamp: string;
	    appId: string;
	    model: string;
	    tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivityEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.appId = source["appId"];
	        this.model = source["model"];
	        this.tokens = source["tokens"];
	    }
	}
	export class AppSummary {
	    appId: string;
	    totalCalls: number;
	    tokensIn: number;
	    tokensOut: number;
	    cacheHits: number;
	
	    static createFrom(source: any = {}) {
	        return new AppSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.totalCalls = source["totalCalls"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	        this.cacheHits = source["cacheHits"];
	    }
	}
	export class DailySummary {
	    date: string;
	    totalCalls: number;
	    tokensIn: number;
	    tokensOut: number;
	    cacheHits: number;
	
	    static createFrom(source: any = {}) {
	        return new DailySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.totalCalls = source["totalCalls"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	        this.cacheHits = source["cacheHits"];
	    }
	}
	export class ModelSummary {
	    model: string;
	    totalCalls: number;
	    tokensIn: number;
	    tokensOut: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.totalCalls = source["totalCalls"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	    }
	}

}

export namespace modelcatalog {
	
	export class Model {
	    id: string;
	    displayName: string;
	    provider: string;
	    inputRatio: number;
	    outputRatio: number;
	    tags: string[];
	    recommended: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Model(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.provider = source["provider"];
	        this.inputRatio = source["inputRatio"];
	        this.outputRatio = source["outputRatio"];
	        this.tags = source["tags"];
	        this.recommended = source["recommended"];
	    }
	}
	export class Catalog {
	    models: Model[];
	    // Go type: time
	    fetchedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Catalog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.models = this.convertValues(source["models"], Model);
	        this.fetchedAt = this.convertValues(source["fetchedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace optimizer {
	
	export class Optimization {
	    id: string;
	    category: string;
	    priority: number;
	    title: string;
	    description: string;
	    action: string;
	    target: string;
	    autoFixable: boolean;
	    status: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Optimization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.priority = source["priority"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.autoFixable = source["autoFixable"];
	        this.status = source["status"];
	        this.error = source["error"];
	    }
	}
	export class AnalysisResult {
	    optimizations: Optimization[];
	    fixableCount: number;
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.optimizations = this.convertValues(source["optimizations"], Optimization);
	        this.fixableCount = source["fixableCount"];
	        this.totalCount = source["totalCount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FixResult {
	    id: string;
	    status: string;
	    message?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new FixResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.error = source["error"];
	    }
	}

}

export namespace preset {
	
	export class Preset {
	    id: string;
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}

}

export namespace process {
	
	export class ProcessInfo {
	    pid: number;
	    tool: string;
	    command: string;
	    status: string;
	    memory: number;
	    since: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.tool = source["tool"];
	        this.command = source["command"];
	        this.status = source["status"];
	        this.memory = source["memory"];
	        this.since = source["since"];
	    }
	}

}

export namespace promoter {
	
	export class PromoterInfo {
	    aff_code: string;
	    share_link: string;
	    gateway_url: string;
	    total_referrals: number;
	    total_earned: number;
	    pending_earned: number;
	
	    static createFrom(source: any = {}) {
	        return new PromoterInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aff_code = source["aff_code"];
	        this.share_link = source["share_link"];
	        this.gateway_url = source["gateway_url"];
	        this.total_referrals = source["total_referrals"];
	        this.total_earned = source["total_earned"];
	        this.pending_earned = source["pending_earned"];
	    }
	}

}

export namespace promptlib {
	
	export class Prompt {
	    id: string;
	    name: string;
	    category: string;
	    tags: string[];
	    content: string;
	    targetTools: string[];
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Prompt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.content = source["content"];
	        this.targetTools = source["targetTools"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace provider {
	
	export class Preset {
	    id: string;
	    name: string;
	    icon: string;
	    iconColor: string;
	    category: string;
	    baseUrl: string;
	    keyFormat: string;
	    docsUrl: string;
	    models: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.iconColor = source["iconColor"];
	        this.category = source["category"];
	        this.baseUrl = source["baseUrl"];
	        this.keyFormat = source["keyFormat"];
	        this.docsUrl = source["docsUrl"];
	        this.models = source["models"];
	        this.description = source["description"];
	    }
	}

}

export namespace proxy {
	
	export class ProxySettings {
	    apiEndpoint: string;
	    apiKey: string;
	    registrationUrl?: string;
	    tenantSlug?: string;
	    userToken?: string;
	    model?: string;
	    toolModels?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new ProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiEndpoint = source["apiEndpoint"];
	        this.apiKey = source["apiKey"];
	        this.registrationUrl = source["registrationUrl"];
	        this.tenantSlug = source["tenantSlug"];
	        this.userToken = source["userToken"];
	        this.model = source["model"];
	        this.toolModels = source["toolModels"];
	    }
	}

}

export namespace proxydetect {
	
	export class DetectedProxy {
	    source: string;
	    host: string;
	    port: number;
	    type: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new DetectedProxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.type = source["type"];
	        this.url = source["url"];
	    }
	}

}

export namespace relay {
	
	export class RelayEndpoint {
	    id: string;
	    name: string;
	    kind: string;
	    url: string;
	    apiKey: string;
	    description?: string;
	    latencyMs: number;
	    healthy: boolean;
	    lastChecked?: string;
	
	    static createFrom(source: any = {}) {
	        return new RelayEndpoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.url = source["url"];
	        this.apiKey = source["apiKey"];
	        this.description = source["description"];
	        this.latencyMs = source["latencyMs"];
	        this.healthy = source["healthy"];
	        this.lastChecked = source["lastChecked"];
	    }
	}

}

export namespace serverctl {
	
	export class ServerConfig {
	    port: number;
	    session_secret: string;
	    admin_password: string;
	    admin_token: string;
	    auto_start: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.session_secret = source["session_secret"];
	        this.admin_password = source["admin_password"];
	        this.admin_token = source["admin_token"];
	        this.auto_start = source["auto_start"];
	    }
	}
	export class ServerStatus {
	    running: boolean;
	    port: number;
	    url: string;
	    uptime: number;
	    version: string;
	    binaryOk: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.url = source["url"];
	        this.uptime = source["uptime"];
	        this.version = source["version"];
	        this.binaryOk = source["binaryOk"];
	    }
	}

}

export namespace snapshot {
	
	export class SnapshotMeta {
	    id: string;
	    tool: string;
	    label: string;
	    createdAt: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tool = source["tool"];
	        this.label = source["label"];
	        this.createdAt = source["createdAt"];
	        this.size = source["size"];
	    }
	}

}

export namespace sysenv {
	
	export class AutostartConfig {
	    enabled: boolean;
	    args: string;
	
	    static createFrom(source: any = {}) {
	        return new AutostartConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.args = source["args"];
	    }
	}
	export class GitInfo {
	    installed: boolean;
	    version: string;
	    userName: string;
	    userEmail: string;
	
	    static createFrom(source: any = {}) {
	        return new GitInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.userName = source["userName"];
	        this.userEmail = source["userEmail"];
	    }
	}
	export class PathEntry {
	    dir: string;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PathEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dir = source["dir"];
	        this.exists = source["exists"];
	    }
	}
	export class RollbackEntry {
	    id: string;
	    action: string;
	    oldValue: string;
	    newValue: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new RollbackEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.action = source["action"];
	        this.oldValue = source["oldValue"];
	        this.newValue = source["newValue"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SystemEnvironment {
	    pathEntries: PathEntry[];
	    autostart: AutostartConfig;
	    git?: GitInfo;
	
	    static createFrom(source: any = {}) {
	        return new SystemEnvironment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pathEntries = this.convertValues(source["pathEntries"], PathEntry);
	        this.autostart = this.convertValues(source["autostart"], AutostartConfig);
	        this.git = this.convertValues(source["git"], GitInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace toolconfig {
	
	export class ToolConfigInfo {
	    tool: string;
	    path: string;
	    exists: boolean;
	    language: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolConfigInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.path = source["path"];
	        this.exists = source["exists"];
	        this.language = source["language"];
	        this.content = source["content"];
	    }
	}

}

export namespace toolhealth {
	
	export class HealthResult {
	    tool: string;
	    status: string;
	    issues: string[];
	
	    static createFrom(source: any = {}) {
	        return new HealthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.status = source["status"];
	        this.issues = source["issues"];
	    }
	}

}

export namespace toolmanifest {
	
	export class PlatformAsset {
	    url: string;
	    sha256?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlatformAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.sha256 = source["sha256"];
	    }
	}
	export class ToolEntry {
	    type: string;
	    npm_package?: string;
	    latest_version: string;
	    platforms?: Record<string, PlatformAsset>;
	
	    static createFrom(source: any = {}) {
	        return new ToolEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.npm_package = source["npm_package"];
	        this.latest_version = source["latest_version"];
	        this.platforms = this.convertValues(source["platforms"], PlatformAsset, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Manifest {
	    generated_at: string;
	    tools: Record<string, ToolEntry>;
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generated_at = source["generated_at"];
	        this.tools = this.convertValues(source["tools"], ToolEntry, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace updater {
	
	export class UpdateInfo {
	    name: string;
	    currentVersion: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	    downloadUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.downloadUrl = source["downloadUrl"];
	    }
	}

}

export namespace validator {
	
	export class ValidationError {
	    field: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.message = source["message"];
	    }
	}
	export class ValidationResult {
	    valid: boolean;
	    errors: ValidationError[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.errors = this.convertValues(source["errors"], ValidationError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

