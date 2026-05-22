export namespace admin {
	
	export class Channel {
	    id: number;
	    name: string;
	    type: number;
	    status: number;
	    group: string;
	    models: string;
	    weight?: number;
	    priority?: number;
	    base_url?: string;
	    balance: number;
	    used_quota: number;
	    tag?: string;
	    remark?: string;
	    created_time: number;
	    test_time: number;
	    response_time: number;
	
	    static createFrom(source: any = {}) {
	        return new Channel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.group = source["group"];
	        this.models = source["models"];
	        this.weight = source["weight"];
	        this.priority = source["priority"];
	        this.base_url = source["base_url"];
	        this.balance = source["balance"];
	        this.used_quota = source["used_quota"];
	        this.tag = source["tag"];
	        this.remark = source["remark"];
	        this.created_time = source["created_time"];
	        this.test_time = source["test_time"];
	        this.response_time = source["response_time"];
	    }
	}
	export class ChannelPage {
	    items: Channel[];
	    page: number;
	    page_size: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new ChannelPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Channel);
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	        this.total = source["total"];
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
	export class DashboardSummary {
	    user_count: number;
	    channel_count: number;
	    token_count: number;
	    today_request: number;
	    today_quota: number;
	    today_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_count = source["user_count"];
	        this.channel_count = source["channel_count"];
	        this.token_count = source["token_count"];
	        this.today_request = source["today_request"];
	        this.today_quota = source["today_quota"];
	        this.today_tokens = source["today_tokens"];
	    }
	}
	export class LogEntry {
	    id: number;
	    user_id: number;
	    username: string;
	    created_at: number;
	    type: number;
	    content: string;
	    model_name: string;
	    token_name: string;
	    quota: number;
	    prompt_tokens: number;
	    completion_tokens: number;
	    use_time: number;
	    is_stream: boolean;
	    channel: number;
	    ip: string;
	    group: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.created_at = source["created_at"];
	        this.type = source["type"];
	        this.content = source["content"];
	        this.model_name = source["model_name"];
	        this.token_name = source["token_name"];
	        this.quota = source["quota"];
	        this.prompt_tokens = source["prompt_tokens"];
	        this.completion_tokens = source["completion_tokens"];
	        this.use_time = source["use_time"];
	        this.is_stream = source["is_stream"];
	        this.channel = source["channel"];
	        this.ip = source["ip"];
	        this.group = source["group"];
	    }
	}
	export class LogPage {
	    items: LogEntry[];
	    page: number;
	    page_size: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new LogPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], LogEntry);
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	        this.total = source["total"];
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
	export class LogQuery {
	    Page: number;
	    PageSize: number;
	    Username: string;
	    TokenName: string;
	    ModelName: string;
	    Type: number;
	    // Go type: time
	    StartAt: any;
	    // Go type: time
	    EndAt: any;
	    ChannelID: number;
	    Group: string;
	    IP: string;
	    OnlyMine: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LogQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Page = source["Page"];
	        this.PageSize = source["PageSize"];
	        this.Username = source["Username"];
	        this.TokenName = source["TokenName"];
	        this.ModelName = source["ModelName"];
	        this.Type = source["Type"];
	        this.StartAt = this.convertValues(source["StartAt"], null);
	        this.EndAt = this.convertValues(source["EndAt"], null);
	        this.ChannelID = source["ChannelID"];
	        this.Group = source["Group"];
	        this.IP = source["IP"];
	        this.OnlyMine = source["OnlyMine"];
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
	export class PerformanceStats {
	    goroutines: number;
	    memory_alloc: number;
	    uptime: number;
	    requests_total: number;
	    requests_per_sec: number;
	
	    static createFrom(source: any = {}) {
	        return new PerformanceStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.goroutines = source["goroutines"];
	        this.memory_alloc = source["memory_alloc"];
	        this.uptime = source["uptime"];
	        this.requests_total = source["requests_total"];
	        this.requests_per_sec = source["requests_per_sec"];
	    }
	}
	export class QuotaDate {
	    date: string;
	    quota: number;
	    request_count: number;
	    token_count: number;
	    model_usage: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new QuotaDate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.quota = source["quota"];
	        this.request_count = source["request_count"];
	        this.token_count = source["token_count"];
	        this.model_usage = source["model_usage"];
	    }
	}
	export class Redemption {
	    id: number;
	    user_id: number;
	    key: string;
	    name: string;
	    status: number;
	    quota: number;
	    used_id: number;
	    used_time: number;
	    expired_time: number;
	    created_time: number;
	
	    static createFrom(source: any = {}) {
	        return new Redemption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.user_id = source["user_id"];
	        this.key = source["key"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.quota = source["quota"];
	        this.used_id = source["used_id"];
	        this.used_time = source["used_time"];
	        this.expired_time = source["expired_time"];
	        this.created_time = source["created_time"];
	    }
	}
	export class RedemptionPage {
	    items: Redemption[];
	    page: number;
	    page_size: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new RedemptionPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Redemption);
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	        this.total = source["total"];
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
	export class SwitchPreset {
	    id: string;
	    name: string;
	    provider: string;
	    description: string;
	    logo: string;
	    config: Record<string, any>;
	    is_official: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SwitchPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.description = source["description"];
	        this.logo = source["logo"];
	        this.config = source["config"];
	        this.is_official = source["is_official"];
	    }
	}
	export class Tenant {
	    id: string;
	    slug: string;
	    name: string;
	    status: string;
	    created_at: number;
	
	    static createFrom(source: any = {}) {
	        return new Tenant(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.created_at = source["created_at"];
	    }
	}
	export class TenantList {
	    items: Tenant[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TenantList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Tenant);
	        this.total = source["total"];
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
	export class Token {
	    id: number;
	    user_id: number;
	    key: string;
	    name: string;
	    status: number;
	    used_quota: number;
	    remain_quota: number;
	    unlimited_quota: boolean;
	    expired_time: number;
	    created_time: number;
	    accessed_time: number;
	    model_limits: string;
	    group: string;
	
	    static createFrom(source: any = {}) {
	        return new Token(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.user_id = source["user_id"];
	        this.key = source["key"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.used_quota = source["used_quota"];
	        this.remain_quota = source["remain_quota"];
	        this.unlimited_quota = source["unlimited_quota"];
	        this.expired_time = source["expired_time"];
	        this.created_time = source["created_time"];
	        this.accessed_time = source["accessed_time"];
	        this.model_limits = source["model_limits"];
	        this.group = source["group"];
	    }
	}
	export class TokenPage {
	    items: Token[];
	    page: number;
	    page_size: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TokenPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Token);
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	        this.total = source["total"];
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

export namespace agent {
	
	export class Permissions {
	    allowShell: boolean;
	    allowFiles: boolean;
	    allowNetwork: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Permissions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowShell = source["allowShell"];
	        this.allowFiles = source["allowFiles"];
	        this.allowNetwork = source["allowNetwork"];
	    }
	}
	export class CreateParams {
	    name: string;
	    icon: string;
	    tags: string[];
	    toolType: string;
	    modelId: string;
	    systemPrompt: string;
	    mcpServers: string[];
	    permissions: Permissions;
	    projectId: string;
	    budgetLimitTokens?: number;
	    budgetLimitCurrency?: number;
	    budgetPeriod: string;
	    budgetPolicy: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.tags = source["tags"];
	        this.toolType = source["toolType"];
	        this.modelId = source["modelId"];
	        this.systemPrompt = source["systemPrompt"];
	        this.mcpServers = source["mcpServers"];
	        this.permissions = this.convertValues(source["permissions"], Permissions);
	        this.projectId = source["projectId"];
	        this.budgetLimitTokens = source["budgetLimitTokens"];
	        this.budgetLimitCurrency = source["budgetLimitCurrency"];
	        this.budgetPeriod = source["budgetPeriod"];
	        this.budgetPolicy = source["budgetPolicy"];
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
	export class ListFilter {
	    status?: string;
	    toolType?: string;
	    projectId?: string;
	    tag?: string;
	
	    static createFrom(source: any = {}) {
	        return new ListFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.toolType = source["toolType"];
	        this.projectId = source["projectId"];
	        this.tag = source["tag"];
	    }
	}
	
	export class Profile {
	    id: string;
	    name: string;
	    icon: string;
	    tags: string[];
	    toolType: string;
	    modelId: string;
	    systemPrompt: string;
	    mcpServers: string[];
	    permissions: Permissions;
	    projectId?: string;
	    status: string;
	    configDir?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    budgetLimitTokens?: number;
	    budgetLimitCurrency?: number;
	    budgetPeriod: string;
	    budgetPolicy: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.tags = source["tags"];
	        this.toolType = source["toolType"];
	        this.modelId = source["modelId"];
	        this.systemPrompt = source["systemPrompt"];
	        this.mcpServers = source["mcpServers"];
	        this.permissions = this.convertValues(source["permissions"], Permissions);
	        this.projectId = source["projectId"];
	        this.status = source["status"];
	        this.configDir = source["configDir"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.budgetLimitTokens = source["budgetLimitTokens"];
	        this.budgetLimitCurrency = source["budgetLimitCurrency"];
	        this.budgetPeriod = source["budgetPeriod"];
	        this.budgetPolicy = source["budgetPolicy"];
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
	export class UpdateParams {
	    name?: string;
	    icon?: string;
	    tags?: string[];
	    modelId?: string;
	    systemPrompt?: string;
	    mcpServers?: string[];
	    permissions?: Permissions;
	    projectId?: string;
	    status?: string;
	    budgetLimitTokens?: number;
	    budgetLimitCurrency?: number;
	    budgetPeriod?: string;
	    budgetPolicy?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.tags = source["tags"];
	        this.modelId = source["modelId"];
	        this.systemPrompt = source["systemPrompt"];
	        this.mcpServers = source["mcpServers"];
	        this.permissions = this.convertValues(source["permissions"], Permissions);
	        this.projectId = source["projectId"];
	        this.status = source["status"];
	        this.budgetLimitTokens = source["budgetLimitTokens"];
	        this.budgetLimitCurrency = source["budgetLimitCurrency"];
	        this.budgetPeriod = source["budgetPeriod"];
	        this.budgetPolicy = source["budgetPolicy"];
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

export namespace agenttemplate {
	
	export class Template {
	    id: string;
	    displayName: string;
	    icon: string;
	    toolType: string;
	    modelId: string;
	    systemPrompt: string;
	    tags: string[];
	    mcpServers: string[];
	    capabilities: string[];
	    budgetTokens: number;
	    budgetUsd: number;
	    budgetPeriod: string;
	    budgetPolicy: string;
	    guardrails: string[];
	    useCases: string[];
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.icon = source["icon"];
	        this.toolType = source["toolType"];
	        this.modelId = source["modelId"];
	        this.systemPrompt = source["systemPrompt"];
	        this.tags = source["tags"];
	        this.mcpServers = source["mcpServers"];
	        this.capabilities = source["capabilities"];
	        this.budgetTokens = source["budgetTokens"];
	        this.budgetUsd = source["budgetUsd"];
	        this.budgetPeriod = source["budgetPeriod"];
	        this.budgetPolicy = source["budgetPolicy"];
	        this.guardrails = source["guardrails"];
	        this.useCases = source["useCases"];
	        this.notes = source["notes"];
	    }
	}

}

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
	
	export class ResellerConfig {
	    hubUrl?: string;
	    adminToken?: string;
	    tenantSlug?: string;
	    displayName?: string;
	
	    static createFrom(source: any = {}) {
	        return new ResellerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hubUrl = source["hubUrl"];
	        this.adminToken = source["adminToken"];
	        this.tenantSlug = source["tenantSlug"];
	        this.displayName = source["displayName"];
	    }
	}
	export class AppSettings {
	    theme: string;
	    language: string;
	    autoUpdate: boolean;
	    editorFontSize: number;
	    startupPage: string;
	    onboardingCompleted: boolean;
	    featureTourSeen: boolean;
	    appMode: string;
	    userLevel: string;
	    lockedHubUrl?: string;
	    brandName?: string;
	    brandLogoBase64?: string;
	    brandPrimaryColor?: string;
	    brandSupportContact?: string;
	    reseller?: ResellerConfig;
	    authClientId?: string;
	    authIssuer?: string;
	    authPlatformUrl?: string;
	
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
	        this.featureTourSeen = source["featureTourSeen"];
	        this.appMode = source["appMode"];
	        this.userLevel = source["userLevel"];
	        this.lockedHubUrl = source["lockedHubUrl"];
	        this.brandName = source["brandName"];
	        this.brandLogoBase64 = source["brandLogoBase64"];
	        this.brandPrimaryColor = source["brandPrimaryColor"];
	        this.brandSupportContact = source["brandSupportContact"];
	        this.reseller = this.convertValues(source["reseller"], ResellerConfig);
	        this.authClientId = source["authClientId"];
	        this.authIssuer = source["authIssuer"];
	        this.authPlatformUrl = source["authPlatformUrl"];
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
	    ownerEmployeeId?: string;
	    costCenter?: string;
	
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
	        this.ownerEmployeeId = source["ownerEmployeeId"];
	        this.costCenter = source["costCenter"];
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

export namespace audit {
	
	export class Entry {
	    id: string;
	    // Go type: time
	    timestamp: any;
	    principal: string;
	    capsHeld: string[];
	    operation: string;
	    target: string;
	    before?: any;
	    after?: any;
	    outcome: string;
	    error?: string;
	    // Go type: time
	    undoneAt?: any;
	    undoneBy?: string;
	    reversible: boolean;
	    metadata?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.principal = source["principal"];
	        this.capsHeld = source["capsHeld"];
	        this.operation = source["operation"];
	        this.target = source["target"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.outcome = source["outcome"];
	        this.error = source["error"];
	        this.undoneAt = this.convertValues(source["undoneAt"], null);
	        this.undoneBy = source["undoneBy"];
	        this.reversible = source["reversible"];
	        this.metadata = source["metadata"];
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
	export class Stats {
	    total: number;
	    ok: number;
	    denied: number;
	    error: number;
	    undone: number;
	    byPrincipal: Record<string, number>;
	    byOperation: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.ok = source["ok"];
	        this.denied = source["denied"];
	        this.error = source["error"];
	        this.undone = source["undone"];
	        this.byPrincipal = source["byPrincipal"];
	        this.byOperation = source["byOperation"];
	    }
	}

}

export namespace auth {
	
	export class PlatformAccount {
	    account_id: number;
	    lurus_id: string;
	    display_name?: string;
	    email?: string;
	    vip_level: number;
	    wallet_balance: number;
	    wallet_frozen: number;
	
	    static createFrom(source: any = {}) {
	        return new PlatformAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_id = source["account_id"];
	        this.lurus_id = source["lurus_id"];
	        this.display_name = source["display_name"];
	        this.email = source["email"];
	        this.vip_level = source["vip_level"];
	        this.wallet_balance = source["wallet_balance"];
	        this.wallet_frozen = source["wallet_frozen"];
	    }
	}
	export class UserInfo {
	    sub: string;
	    name: string;
	    email: string;
	    picture: string;
	
	    static createFrom(source: any = {}) {
	        return new UserInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sub = source["sub"];
	        this.name = source["name"];
	        this.email = source["email"];
	        this.picture = source["picture"];
	    }
	}
	export class AuthState {
	    is_logged_in: boolean;
	    user?: UserInfo;
	    platform?: PlatformAccount;
	    expires_at?: string;
	    has_gateway_token: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuthState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_logged_in = source["is_logged_in"];
	        this.user = this.convertValues(source["user"], UserInfo);
	        this.platform = this.convertValues(source["platform"], PlatformAccount);
	        this.expires_at = source["expires_at"];
	        this.has_gateway_token = source["has_gateway_token"];
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

export namespace bashguard {
	
	export class BlockEntry {
	    // Go type: time
	    time: any;
	    tool: string;
	    command: string;
	    ruleId: string;
	    reason: string;
	    severity: string;
	    cwd?: string;
	
	    static createFrom(source: any = {}) {
	        return new BlockEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.tool = source["tool"];
	        this.command = source["command"];
	        this.ruleId = source["ruleId"];
	        this.reason = source["reason"];
	        this.severity = source["severity"];
	        this.cwd = source["cwd"];
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
	export class HookInstallStatus {
	    tool: string;
	    installed: boolean;
	    hookCmd: string;
	    configPath: string;
	    issue?: string;
	
	    static createFrom(source: any = {}) {
	        return new HookInstallStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.installed = source["installed"];
	        this.hookCmd = source["hookCmd"];
	        this.configPath = source["configPath"];
	        this.issue = source["issue"];
	    }
	}
	export class Rule {
	    id: string;
	    pattern: string;
	    severity: string;
	    reasonZh: string;
	    reasonEn: string;
	    reference?: string;
	
	    static createFrom(source: any = {}) {
	        return new Rule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pattern = source["pattern"];
	        this.severity = source["severity"];
	        this.reasonZh = source["reasonZh"];
	        this.reasonEn = source["reasonEn"];
	        this.reference = source["reference"];
	    }
	}
	export class MatchResult {
	    allowed: boolean;
	    rule?: Rule;
	    reason?: string;
	    normalizedCommand: string;
	
	    static createFrom(source: any = {}) {
	        return new MatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowed = source["allowed"];
	        this.rule = this.convertValues(source["rule"], Rule);
	        this.reason = source["reason"];
	        this.normalizedCommand = source["normalizedCommand"];
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
	    role: number;
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
	        this.role = source["role"];
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

export namespace budget {
	
	export class Config {
	    enabled: boolean;
	    dailyTokens: number;
	    sessionTokens: number;
	    softWarnPct: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.dailyTokens = source["dailyTokens"];
	        this.sessionTokens = source["sessionTokens"];
	        this.softWarnPct = source["softWarnPct"];
	    }
	}
	export class Status {
	    enabled: boolean;
	    dailyTokens: number;
	    sessionTokens: number;
	    dailyUsed: number;
	    sessionUsed: number;
	    dailyPct: number;
	    sessionPct: number;
	    // Go type: time
	    sessionStart: any;
	    softWarnPct: number;
	    hitDaily: boolean;
	    hitSession: boolean;
	    warnDaily: boolean;
	    warnSession: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.dailyTokens = source["dailyTokens"];
	        this.sessionTokens = source["sessionTokens"];
	        this.dailyUsed = source["dailyUsed"];
	        this.sessionUsed = source["sessionUsed"];
	        this.dailyPct = source["dailyPct"];
	        this.sessionPct = source["sessionPct"];
	        this.sessionStart = this.convertValues(source["sessionStart"], null);
	        this.softWarnPct = source["softWarnPct"];
	        this.hitDaily = source["hitDaily"];
	        this.hitSession = source["hitSession"];
	        this.warnDaily = source["warnDaily"];
	        this.warnSession = source["warnSession"];
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

export namespace connectivity {
	
	export class LocalProxy {
	    host: string;
	    port: number;
	    url: string;
	    guessedName?: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalProxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.url = source["url"];
	        this.guessedName = source["guessedName"];
	    }
	}
	export class Provider {
	    id: string;
	    label: string;
	    url: string;
	    tier: string;
	
	    static createFrom(source: any = {}) {
	        return new Provider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.url = source["url"];
	        this.tier = source["tier"];
	    }
	}
	export class ProviderResult {
	    provider: Provider;
	    dnsOK: boolean;
	    dnsError?: string;
	    directOK: boolean;
	    directMs?: number;
	    directError?: string;
	    upstreamOK?: boolean;
	    upstreamMs?: number;
	    upstreamError?: string;
	    upstreamTried: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = this.convertValues(source["provider"], Provider);
	        this.dnsOK = source["dnsOK"];
	        this.dnsError = source["dnsError"];
	        this.directOK = source["directOK"];
	        this.directMs = source["directMs"];
	        this.directError = source["directError"];
	        this.upstreamOK = source["upstreamOK"];
	        this.upstreamMs = source["upstreamMs"];
	        this.upstreamError = source["upstreamError"];
	        this.upstreamTried = source["upstreamTried"];
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
	export class Suggestion {
	    kind: string;
	    title: string;
	    detail: string;
	    payload?: string;
	
	    static createFrom(source: any = {}) {
	        return new Suggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.payload = source["payload"];
	    }
	}
	export class SystemProxy {
	    httpProxy?: string;
	    httpsProxy?: string;
	    allProxy?: string;
	    noProxy?: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemProxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.httpProxy = source["httpProxy"];
	        this.httpsProxy = source["httpsProxy"];
	        this.allProxy = source["allProxy"];
	        this.noProxy = source["noProxy"];
	    }
	}
	export class Report {
	    // Go type: time
	    generatedAt: any;
	    providers: ProviderResult[];
	    localProxies: LocalProxy[];
	    systemProxy: SystemProxy;
	    suggestions: Suggestion[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generatedAt = this.convertValues(source["generatedAt"], null);
	        this.providers = this.convertValues(source["providers"], ProviderResult);
	        this.localProxies = this.convertValues(source["localProxies"], LocalProxy);
	        this.systemProxy = this.convertValues(source["systemProxy"], SystemProxy);
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

export namespace conversation {
	
	export class ContextFile {
	    path: string;
	    name: string;
	    content: string;
	    size: number;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContextFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.content = source["content"];
	        this.size = source["size"];
	        this.truncated = source["truncated"];
	    }
	}
	export class ConversationFilter {
	    tool: string;
	    cwdSubstring: string;
	    model: string;
	    startAfter: string;
	    endBefore: string;
	    onlyDLPHits: boolean;
	    search: string;
	
	    static createFrom(source: any = {}) {
	        return new ConversationFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.cwdSubstring = source["cwdSubstring"];
	        this.model = source["model"];
	        this.startAfter = source["startAfter"];
	        this.endBefore = source["endBefore"];
	        this.onlyDLPHits = source["onlyDLPHits"];
	        this.search = source["search"];
	    }
	}
	export class ConversationMeta {
	    tool: string;
	    sessionID: string;
	    cwd?: string;
	    path: string;
	    model?: string;
	    // Go type: time
	    startedAt?: any;
	    // Go type: time
	    endedAt?: any;
	    messageCount: number;
	    userMessages: number;
	    assistantMessages: number;
	    totalTokens: number;
	    toolList?: string[];
	    hasErrors: boolean;
	    hasDLPHits?: boolean;
	    parentSessionID?: string;
	    forkPointUUID?: string;
	    fileModTime: number;
	    fileSize: number;
	
	    static createFrom(source: any = {}) {
	        return new ConversationMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.sessionID = source["sessionID"];
	        this.cwd = source["cwd"];
	        this.path = source["path"];
	        this.model = source["model"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.endedAt = this.convertValues(source["endedAt"], null);
	        this.messageCount = source["messageCount"];
	        this.userMessages = source["userMessages"];
	        this.assistantMessages = source["assistantMessages"];
	        this.totalTokens = source["totalTokens"];
	        this.toolList = source["toolList"];
	        this.hasErrors = source["hasErrors"];
	        this.hasDLPHits = source["hasDLPHits"];
	        this.parentSessionID = source["parentSessionID"];
	        this.forkPointUUID = source["forkPointUUID"];
	        this.fileModTime = source["fileModTime"];
	        this.fileSize = source["fileSize"];
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
	export class Event {
	    type: string;
	    messageUUID?: string;
	    parentUUID?: string;
	    // Go type: time
	    timestamp: any;
	    content?: string;
	    toolName?: string;
	    toolArgs?: number[];
	    model?: string;
	    inputTokens?: number;
	    outputTokens?: number;
	    cacheCreationTokens?: number;
	    cacheReadTokens?: number;
	    raw?: number[];
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.messageUUID = source["messageUUID"];
	        this.parentUUID = source["parentUUID"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.content = source["content"];
	        this.toolName = source["toolName"];
	        this.toolArgs = source["toolArgs"];
	        this.model = source["model"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.cacheCreationTokens = source["cacheCreationTokens"];
	        this.cacheReadTokens = source["cacheReadTokens"];
	        this.raw = source["raw"];
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
	export class ForkResult {
	    newSessionID: string;
	    newPath: string;
	    parentPath: string;
	    forkPointUUID: string;
	    messagesKept: number;
	
	    static createFrom(source: any = {}) {
	        return new ForkResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newSessionID = source["newSessionID"];
	        this.newPath = source["newPath"];
	        this.parentPath = source["parentPath"];
	        this.forkPointUUID = source["forkPointUUID"];
	        this.messagesKept = source["messagesKept"];
	    }
	}
	export class ReindexResult {
	    scanned: number;
	    added: number;
	    updated: number;
	    removed: number;
	    errors: number;
	
	    static createFrom(source: any = {}) {
	        return new ReindexResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanned = source["scanned"];
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.removed = source["removed"];
	        this.errors = source["errors"];
	    }
	}

}

export namespace deploy {
	
	export class Result {
	    Kind: string;
	    HubURL: string;
	    AdminToken: string;
	    TenantSlug: string;
	    DisplayName: string;
	    Notes: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Kind = source["Kind"];
	        this.HubURL = source["HubURL"];
	        this.AdminToken = source["AdminToken"];
	        this.TenantSlug = source["TenantSlug"];
	        this.DisplayName = source["DisplayName"];
	        this.Notes = source["Notes"];
	    }
	}

}

export namespace dlp {
	
	export class Hit {
	    patternName: string;
	    severity: string;
	    policy: string;
	    start: number;
	    end: number;
	    snippet: string;
	
	    static createFrom(source: any = {}) {
	        return new Hit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.patternName = source["patternName"];
	        this.severity = source["severity"];
	        this.policy = source["policy"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.snippet = source["snippet"];
	    }
	}
	export class HitRecord {
	    // Go type: time
	    timestamp: any;
	    source: string;
	    path: string;
	    hit: Hit;
	
	    static createFrom(source: any = {}) {
	        return new HitRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.source = source["source"];
	        this.path = source["path"];
	        this.hit = this.convertValues(source["hit"], Hit);
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
	export class HitStats {
	    total: number;
	    bySeverity: Record<string, number>;
	    byPolicy: Record<string, number>;
	    byPattern: Record<string, number>;
	    bySource: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new HitStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.bySeverity = source["bySeverity"];
	        this.byPolicy = source["byPolicy"];
	        this.byPattern = source["byPattern"];
	        this.bySource = source["bySource"];
	    }
	}
	export class Pattern {
	    name: string;
	    description: string;
	    regex: string;
	    severity: string;
	    policy: string;
	    tags?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Pattern(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.regex = source["regex"];
	        this.severity = source["severity"];
	        this.policy = source["policy"];
	        this.tags = source["tags"];
	    }
	}
	export class Result {
	    hits: Hit[];
	    highestPolicy: string;
	    blocked: boolean;
	    redacted: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], Hit);
	        this.highestPolicy = source["highestPolicy"];
	        this.blocked = source["blocked"];
	        this.redacted = source["redacted"];
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

export namespace feishu {
	
	export class Config {
	    webhookUrl: string;
	    secret?: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.webhookUrl = source["webhookUrl"];
	        this.secret = source["secret"];
	    }
	}

}

export namespace gateway {
	
	export class FallbackEntry {
	    name: string;
	    url: string;
	    token: string;
	    priority: number;
	
	    static createFrom(source: any = {}) {
	        return new FallbackEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.token = source["token"];
	        this.priority = source["priority"];
	    }
	}
	export class Config {
	    port: number;
	    upstreamUrl: string;
	    userToken: string;
	    autoStart: boolean;
	    fallbacks: FallbackEntry[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.upstreamUrl = source["upstreamUrl"];
	        this.userToken = source["userToken"];
	        this.autoStart = source["autoStart"];
	        this.fallbacks = this.convertValues(source["fallbacks"], FallbackEntry);
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

export namespace livesession {
	
	export class EventSummary {
	    // Go type: time
	    time: any;
	    kind: string;
	    label: string;
	    details?: string;
	
	    static createFrom(source: any = {}) {
	        return new EventSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.details = source["details"];
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
	export class FileTouch {
	    path: string;
	    count: number;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new FileTouch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.count = source["count"];
	        this.kind = source["kind"];
	    }
	}
	export class PendingTool {
	    name: string;
	    preview: string;
	    // Go type: time
	    startedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new PendingTool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.preview = source["preview"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
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
	export class LiveSession {
	    sessionId: string;
	    tool: string;
	    cwd: string;
	    projectName: string;
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    lastActivity: any;
	    model?: string;
	    transcriptPath: string;
	    status: string;
	    pendingTool?: PendingTool;
	    recent: EventSummary[];
	    messageCount: number;
	    toolCallCount: number;
	    inputTokens: number;
	    outputTokens: number;
	    cacheCreateTokens: number;
	    cacheReadTokens: number;
	    estimatedUsd: number;
	    modelsSeen?: string[];
	    bashCommands?: string[];
	    filesTouched?: FileTouch[];
	
	    static createFrom(source: any = {}) {
	        return new LiveSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.tool = source["tool"];
	        this.cwd = source["cwd"];
	        this.projectName = source["projectName"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.lastActivity = this.convertValues(source["lastActivity"], null);
	        this.model = source["model"];
	        this.transcriptPath = source["transcriptPath"];
	        this.status = source["status"];
	        this.pendingTool = this.convertValues(source["pendingTool"], PendingTool);
	        this.recent = this.convertValues(source["recent"], EventSummary);
	        this.messageCount = source["messageCount"];
	        this.toolCallCount = source["toolCallCount"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.cacheCreateTokens = source["cacheCreateTokens"];
	        this.cacheReadTokens = source["cacheReadTokens"];
	        this.estimatedUsd = source["estimatedUsd"];
	        this.modelsSeen = source["modelsSeen"];
	        this.bashCommands = source["bashCommands"];
	        this.filesTouched = this.convertValues(source["filesTouched"], FileTouch);
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

export namespace main {
	
	export class ActivationStatus {
	    state: string;
	    stateReason?: string;
	    activated: boolean;
	    hubUrl?: string;
	    tenantSlug?: string;
	    userId?: number;
	    quota?: number;
	    // Go type: time
	    expiresAt?: any;
	    // Go type: time
	    activatedAt?: any;
	    // Go type: time
	    lastHeartbeat?: any;
	    fingerprint?: string;
	
	    static createFrom(source: any = {}) {
	        return new ActivationStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.stateReason = source["stateReason"];
	        this.activated = source["activated"];
	        this.hubUrl = source["hubUrl"];
	        this.tenantSlug = source["tenantSlug"];
	        this.userId = source["userId"];
	        this.quota = source["quota"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
	        this.activatedAt = this.convertValues(source["activatedAt"], null);
	        this.lastHeartbeat = this.convertValues(source["lastHeartbeat"], null);
	        this.fingerprint = source["fingerprint"];
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
	export class AgentStats {
	    total: number;
	    running: number;
	    stopped: number;
	    error: number;
	    created: number;
	
	    static createFrom(source: any = {}) {
	        return new AgentStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.running = source["running"];
	        this.stopped = source["stopped"];
	        this.error = source["error"];
	        this.created = source["created"];
	    }
	}
	export class AuditFilter {
	    principal: string;
	    operation: string;
	    outcome: string;
	    onlyReversible: boolean;
	    onlyUndone: boolean;
	    onlyNotUndone: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuditFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.principal = source["principal"];
	        this.operation = source["operation"];
	        this.outcome = source["outcome"];
	        this.onlyReversible = source["onlyReversible"];
	        this.onlyUndone = source["onlyUndone"];
	        this.onlyNotUndone = source["onlyNotUndone"];
	    }
	}
	export class BuildHistoryEntry {
	    // Go type: time
	    builtAt: any;
	    brandName: string;
	    hubUrl: string;
	    binaryPath: string;
	    sha256: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildHistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.builtAt = this.convertValues(source["builtAt"], null);
	        this.brandName = source["brandName"];
	        this.hubUrl = source["hubUrl"];
	        this.binaryPath = source["binaryPath"];
	        this.sha256 = source["sha256"];
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
	export class ChargebackRow {
	    kind: string;
	    deptId?: string;
	    deptName?: string;
	    employeeId?: string;
	    email?: string;
	    displayName?: string;
	    costCenter?: string;
	    totalCalls: number;
	    tokensIn: number;
	    tokensOut: number;
	    uniqueEmployees?: number;
	
	    static createFrom(source: any = {}) {
	        return new ChargebackRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.deptId = source["deptId"];
	        this.deptName = source["deptName"];
	        this.employeeId = source["employeeId"];
	        this.email = source["email"];
	        this.displayName = source["displayName"];
	        this.costCenter = source["costCenter"];
	        this.totalCalls = source["totalCalls"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	        this.uniqueEmployees = source["uniqueEmployees"];
	    }
	}
	export class ChargebackReport {
	    fromMs: number;
	    toMs: number;
	    byDepartment: ChargebackRow[];
	    byEmployee: ChargebackRow[];
	
	    static createFrom(source: any = {}) {
	        return new ChargebackReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fromMs = source["fromMs"];
	        this.toMs = source["toMs"];
	        this.byDepartment = this.convertValues(source["byDepartment"], ChargebackRow);
	        this.byEmployee = this.convertValues(source["byEmployee"], ChargebackRow);
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
	
	export class CompetingInstall {
	    id: string;
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new CompetingInstall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class ConversationEvents {
	    meta: conversation.ConversationMeta;
	    events: conversation.Event[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationEvents(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], conversation.ConversationMeta);
	        this.events = this.convertValues(source["events"], conversation.Event);
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
	export class DiagnosticCheck {
	    id: string;
	    label: string;
	    status: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	    }
	}
	export class DiagnosticsReport {
	    generatedAt: string;
	    appVersion: string;
	    os: string;
	    arch: string;
	    configDir: string;
	    checks: DiagnosticCheck[];
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticsReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generatedAt = source["generatedAt"];
	        this.appVersion = source["appVersion"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.configDir = source["configDir"];
	        this.checks = this.convertValues(source["checks"], DiagnosticCheck);
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
	export class PreflightCheck {
	    id: string;
	    pass: boolean;
	    titleZh: string;
	    titleEn: string;
	    detailZh?: string;
	    detailEn?: string;
	
	    static createFrom(source: any = {}) {
	        return new PreflightCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pass = source["pass"];
	        this.titleZh = source["titleZh"];
	        this.titleEn = source["titleEn"];
	        this.detailZh = source["detailZh"];
	        this.detailEn = source["detailEn"];
	    }
	}
	export class PreflightReport {
	    ok: boolean;
	    checks: PreflightCheck[];
	
	    static createFrom(source: any = {}) {
	        return new PreflightReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.checks = this.convertValues(source["checks"], PreflightCheck);
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
	
	
	export class ToolManifestRow {
	    name: string;
	    type: string;
	    npmPackage?: string;
	    latestVersion: string;
	    status: string;
	    platforms: Record<string, toolmanifest.PlatformAsset>;
	    overridden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolManifestRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.npmPackage = source["npmPackage"];
	        this.latestVersion = source["latestVersion"];
	        this.status = source["status"];
	        this.platforms = this.convertValues(source["platforms"], toolmanifest.PlatformAsset, true);
	        this.overridden = source["overridden"];
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
	export class ToolManifestAdminView {
	    rows: ToolManifestRow[];
	    upstreamSource: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolManifestAdminView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], ToolManifestRow);
	        this.upstreamSource = source["upstreamSource"];
	        this.updatedAt = source["updatedAt"];
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
	export class WhiteLabelInputs {
	    brandName: string;
	    hubUrl: string;
	    tenantSlug?: string;
	    primaryColor?: string;
	    logoBase64?: string;
	    supportContact?: string;
	    outputDir?: string;
	    iconPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new WhiteLabelInputs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.brandName = source["brandName"];
	        this.hubUrl = source["hubUrl"];
	        this.tenantSlug = source["tenantSlug"];
	        this.primaryColor = source["primaryColor"];
	        this.logoBase64 = source["logoBase64"];
	        this.supportContact = source["supportContact"];
	        this.outputDir = source["outputDir"];
	        this.iconPath = source["iconPath"];
	    }
	}
	export class WhiteLabelOutput {
	    outputDir: string;
	    binaryPath: string;
	    sidecarPath: string;
	    binarySha256: string;
	    sidecarSha256: string;
	    notes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new WhiteLabelOutput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputDir = source["outputDir"];
	        this.binaryPath = source["binaryPath"];
	        this.sidecarPath = source["sidecarPath"];
	        this.binarySha256 = source["binarySha256"];
	        this.sidecarSha256 = source["sidecarSha256"];
	        this.notes = source["notes"];
	    }
	}
	export class resellerKindEntry {
	    kind: string;
	    implemented: boolean;
	    labelZh: string;
	    labelEn: string;
	    descriptionZh: string;
	    descriptionEn: string;
	
	    static createFrom(source: any = {}) {
	        return new resellerKindEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.implemented = source["implemented"];
	        this.labelZh = source["labelZh"];
	        this.labelEn = source["labelEn"];
	        this.descriptionZh = source["descriptionZh"];
	        this.descriptionEn = source["descriptionEn"];
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

export namespace netproxy {
	
	export class Settings {
	    enabled: boolean;
	    url: string;
	    noProxy?: string;
	    testUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.url = source["url"];
	        this.noProxy = source["noProxy"];
	        this.testUrl = source["testUrl"];
	    }
	}
	export class TestResult {
	    ok: boolean;
	    statusCode?: number;
	    latencyMs?: number;
	    error?: string;
	    probedUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.statusCode = source["statusCode"];
	        this.latencyMs = source["latencyMs"];
	        this.error = source["error"];
	        this.probedUrl = source["probedUrl"];
	    }
	}

}

export namespace notify {
	
	export class ApprovalRequest {
	    Command: string;
	    Reason: string;
	    RuleID: string;
	
	    static createFrom(source: any = {}) {
	        return new ApprovalRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Command = source["Command"];
	        this.Reason = source["Reason"];
	        this.RuleID = source["RuleID"];
	    }
	}
	export class Event {
	    id: string;
	    // Go type: time
	    time: any;
	    kind: string;
	    severity: string;
	    title: string;
	    body: string;
	    project?: string;
	    tool?: string;
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = this.convertValues(source["time"], null);
	        this.kind = source["kind"];
	        this.severity = source["severity"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.project = source["project"];
	        this.tool = source["tool"];
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

export namespace orgsync {
	
	export class CSVImportError {
	    lineNumber: number;
	    email: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CSVImportError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lineNumber = source["lineNumber"];
	        this.email = source["email"];
	        this.reason = source["reason"];
	    }
	}
	export class CSVImportResult {
	    created: number;
	    updated: number;
	    skipped: number;
	    errorRows: CSVImportError[];
	
	    static createFrom(source: any = {}) {
	        return new CSVImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = source["created"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	        this.errorRows = this.convertValues(source["errorRows"], CSVImportError);
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
	export class Department {
	    id: string;
	    parentId: string;
	    name: string;
	    costCenter: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Department(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentId = source["parentId"];
	        this.name = source["name"];
	        this.costCenter = source["costCenter"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class Employee {
	    id: string;
	    ssoSubject?: string;
	    email: string;
	    displayName: string;
	    departmentId: string;
	    role: string;
	    managerId?: string;
	    active: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Employee(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ssoSubject = source["ssoSubject"];
	        this.email = source["email"];
	        this.displayName = source["displayName"];
	        this.departmentId = source["departmentId"];
	        this.role = source["role"];
	        this.managerId = source["managerId"];
	        this.active = source["active"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class TreeNode {
	    department: Department;
	    children: TreeNode[];
	    employeeCount: number;
	
	    static createFrom(source: any = {}) {
	        return new TreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.department = this.convertValues(source["department"], Department);
	        this.children = this.convertValues(source["children"], TreeNode);
	        this.employeeCount = source["employeeCount"];
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
	    freeTier: boolean;
	    needsProxy: boolean;
	
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
	        this.freeTier = source["freeTier"];
	        this.needsProxy = source["needsProxy"];
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
	    upstreamProxy?: netproxy.Settings;
	
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
	        this.upstreamProxy = this.convertValues(source["upstreamProxy"], netproxy.Settings);
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
	
	export class CircuitState {
	    endpointID: string;
	    status: string;
	    consecutiveFailures: number;
	    lastFailureMs?: number;
	    nextProbeMs?: number;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new CircuitState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.endpointID = source["endpointID"];
	        this.status = source["status"];
	        this.consecutiveFailures = source["consecutiveFailures"];
	        this.lastFailureMs = source["lastFailureMs"];
	        this.nextProbeMs = source["nextProbeMs"];
	        this.lastError = source["lastError"];
	    }
	}
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
	export class PickResult {
	    Endpoint: RelayEndpoint;
	    MatchedBy: string;
	    Healthy: RelayEndpoint[];
	
	    static createFrom(source: any = {}) {
	        return new PickResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Endpoint = this.convertValues(source["Endpoint"], RelayEndpoint);
	        this.MatchedBy = source["MatchedBy"];
	        this.Healthy = this.convertValues(source["Healthy"], RelayEndpoint);
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

export namespace repoaudit {
	
	export class Finding {
	    file: string;
	    fullPath: string;
	    tool: string;
	    field: string;
	    severity: string;
	    issueZh: string;
	    issueEn: string;
	    detailValue: string;
	    suggestedAction: string;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.fullPath = source["fullPath"];
	        this.tool = source["tool"];
	        this.field = source["field"];
	        this.severity = source["severity"];
	        this.issueZh = source["issueZh"];
	        this.issueEn = source["issueEn"];
	        this.detailValue = source["detailValue"];
	        this.suggestedAction = source["suggestedAction"];
	    }
	}
	export class AuditReport {
	    path: string;
	    // Go type: time
	    scannedAt: any;
	    findings: Finding[];
	    filesFound: string[];
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.scannedAt = this.convertValues(source["scannedAt"], null);
	        this.findings = this.convertValues(source["findings"], Finding);
	        this.filesFound = source["filesFound"];
	        this.verdict = source["verdict"];
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

export namespace store {
	
	export class RulesPersist {
	    stuckAfterSec: number;
	    stuckEscalateSec: number;
	    idleAfterSec: number;
	    notifyStuck: boolean;
	    notifyDone: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RulesPersist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stuckAfterSec = source["stuckAfterSec"];
	        this.stuckEscalateSec = source["stuckEscalateSec"];
	        this.idleAfterSec = source["idleAfterSec"];
	        this.notifyStuck = source["notifyStuck"];
	        this.notifyDone = source["notifyDone"];
	    }
	}
	export class AppConfig {
	    enabled: boolean;
	    feishu: feishu.Config;
	    rules: RulesPersist;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.feishu = this.convertValues(source["feishu"], feishu.Config);
	        this.rules = this.convertValues(source["rules"], RulesPersist);
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
	    status?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.npm_package = source["npm_package"];
	        this.latest_version = source["latest_version"];
	        this.platforms = this.convertValues(source["platforms"], PlatformAsset, true);
	        this.status = source["status"];
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

export namespace toolruntime {
	
	export class ToolRuntime {
	    tool: string;
	    installed: boolean;
	    configPath: string;
	    model: string;
	    endpoint: string;
	    endpointKind: string;
	    hasApiKey: boolean;
	    processRunning: boolean;
	    processPID?: number;
	    connState: string;
	    latencyMs?: number;
	    probeError?: string;
	    // Go type: time
	    checkedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ToolRuntime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.installed = source["installed"];
	        this.configPath = source["configPath"];
	        this.model = source["model"];
	        this.endpoint = source["endpoint"];
	        this.endpointKind = source["endpointKind"];
	        this.hasApiKey = source["hasApiKey"];
	        this.processRunning = source["processRunning"];
	        this.processPID = source["processPID"];
	        this.connState = source["connState"];
	        this.latencyMs = source["latencyMs"];
	        this.probeError = source["probeError"];
	        this.checkedAt = this.convertValues(source["checkedAt"], null);
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

export namespace topology {
	
	export class Edge {
	    from: string;
	    to: string;
	    credential?: string;
	    status?: string;
	    label?: string;
	
	    static createFrom(source: any = {}) {
	        return new Edge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.credential = source["credential"];
	        this.status = source["status"];
	        this.label = source["label"];
	    }
	}
	export class Node {
	    id: string;
	    kind: string;
	    label: string;
	    status: string;
	    detail?: string;
	    hint?: string;
	    navPage?: string;
	    navSubTab?: string;
	    fixAction?: string;
	    latencyMs?: number;
	    badge?: string;
	    highlight?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Node(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	        this.hint = source["hint"];
	        this.navPage = source["navPage"];
	        this.navSubTab = source["navSubTab"];
	        this.fixAction = source["fixAction"];
	        this.latencyMs = source["latencyMs"];
	        this.badge = source["badge"];
	        this.highlight = source["highlight"];
	    }
	}
	export class Summary {
	    ok: number;
	    degraded: number;
	    down: number;
	    notconfigured: number;
	    unknown: number;
	    headline: string;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.degraded = source["degraded"];
	        this.down = source["down"];
	        this.notconfigured = source["notconfigured"];
	        this.unknown = source["unknown"];
	        this.headline = source["headline"];
	    }
	}
	export class Snapshot {
	    mode: string;
	    // Go type: time
	    generatedAt: any;
	    nodes: Node[];
	    edges: Edge[];
	    summary: Summary;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.generatedAt = this.convertValues(source["generatedAt"], null);
	        this.nodes = this.convertValues(source["nodes"], Node);
	        this.edges = this.convertValues(source["edges"], Edge);
	        this.summary = this.convertValues(source["summary"], Summary);
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

