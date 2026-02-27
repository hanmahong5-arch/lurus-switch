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

}

export namespace billing {

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
	    pay_methods: Record<string, string>[];
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

}

export namespace installer {

	export class ToolStatus {
	    name: string;
	    installed: boolean;
	    version: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	    path: string;

	    static createFrom(source: any = {}) {
	        return new ToolStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.path = source["path"];
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

export namespace proxy {

	export class ProxySettings {
	    apiEndpoint: string;
	    apiKey: string;
	    registrationUrl?: string;
	    tenantSlug?: string;
	    userToken?: string;

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

