/**
 * OpenClaw memx Plugin
 *
 * Bridges OpenClaw's memory slot to the memx Python HTTP service
 * (UU114/memx — Adaptive Context Engine over mem0).
 *
 * Requires the memx-bridge service running at bridgeUrl (default http://127.0.0.1:7788).
 *
 * Features:
 *  - 5 tools: memory_search, memory_list, memory_store, memory_get, memory_forget
 *  - Auto-recall: inject relevant memories before each agent turn
 *  - Auto-capture: store conversation facts after each agent turn
 *  - CLI: openclaw memx search, openclaw memx stats
 */

import { Type } from "@sinclair/typebox";
import type { OpenClawPluginApi } from "openclaw/plugin-sdk";

// ── Config ────────────────────────────────────────────────────────────────

type MemxConfig = {
  bridgeUrl: string;      // URL of the memx FastAPI bridge
  userId: string;         // default user ID for scoping memories
  autoCapture: boolean;
  autoRecall: boolean;
  searchThreshold: number;
  topK: number;
};

interface MemoryItem {
  id: string;
  memory: string;
  user_id?: string;
  score?: number;
  metadata?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
}

interface AddResultItem {
  id: string;
  memory: string;
  event: string;
}

// ── HTTP helper ────────────────────────────────────────────────────────────

async function bridgePost<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`memx bridge error ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

async function bridgeGet<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`memx bridge error ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

async function bridgeDelete(url: string): Promise<void> {
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`memx bridge error ${res.status}: ${text}`);
  }
}

// ── Config parser ──────────────────────────────────────────────────────────

const memxConfigSchema = {
  parse(value: unknown): MemxConfig {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      throw new Error("openclaw-memx config required");
    }
    const cfg = value as Record<string, unknown>;
    return {
      bridgeUrl: typeof cfg.bridgeUrl === "string" ? cfg.bridgeUrl : "http://127.0.0.1:7788",
      userId: typeof cfg.userId === "string" && cfg.userId ? cfg.userId : "default",
      autoCapture: cfg.autoCapture !== false,
      autoRecall: cfg.autoRecall !== false,
      searchThreshold: typeof cfg.searchThreshold === "number" ? cfg.searchThreshold : 0.3,
      topK: typeof cfg.topK === "number" ? cfg.topK : 5,
    };
  },
};

// ── Plugin ─────────────────────────────────────────────────────────────────

const memxPlugin = {
  id: "openclaw-memx",
  name: "Memory (memx)",
  description: "memx memory backend — UU114/memx via local HTTP bridge",
  kind: "memory" as const,
  configSchema: memxConfigSchema,

  register(api: OpenClawPluginApi) {
    const cfg = memxConfigSchema.parse(api.pluginConfig);
    const base = cfg.bridgeUrl;
    let currentSessionId: string | undefined;

    api.logger.info(
      `openclaw-memx: registered (bridge: ${base}, user: ${cfg.userId}, autoRecall: ${cfg.autoRecall}, autoCapture: ${cfg.autoCapture})`,
    );

    // ── Tools ───────────────────────────────────────────────────────────

    api.registerTool(
      {
        name: "memory_search",
        label: "Memory Search",
        description:
          "Search through long-term memories. Use when you need context about user preferences, past decisions, or previously discussed topics.",
        parameters: Type.Object({
          query: Type.String({ description: "Search query" }),
          limit: Type.Optional(Type.Number({ description: `Max results (default: ${cfg.topK})` })),
          userId: Type.Optional(Type.String({ description: "User ID (default: configured)" })),
        }),
        async execute(_id, params) {
          const { query, limit, userId } = params as { query: string; limit?: number; userId?: string };
          try {
            const resp = await bridgePost<{ results: MemoryItem[] }>(`${base}/search`, {
              query,
              user_id: userId || cfg.userId,
              limit: limit ?? cfg.topK,
              threshold: cfg.searchThreshold,
              run_id: currentSessionId,
            });
            const items = resp.results ?? [];
            if (!items.length) {
              return { content: [{ type: "text", text: "No matching memories found." }], details: { found: 0 } };
            }
            const text = items
              .map((r, i) => `${i + 1}. ${r.memory}${r.score != null ? ` (score: ${(r.score * 100).toFixed(0)}%)` : ""}`)
              .join("\n");
            return {
              content: [{ type: "text", text: `Found ${items.length} memories:\n\n${text}` }],
              details: { found: items.length, memories: items },
            };
          } catch (err) {
            return { content: [{ type: "text", text: `Memory search failed: ${String(err)}` }], details: { error: String(err) } };
          }
        },
      },
      { name: "memory_search" },
    );

    api.registerTool(
      {
        name: "memory_store",
        label: "Memory Store",
        description: "Explicitly store a specific fact or piece of information into long-term memory.",
        parameters: Type.Object({
          text: Type.String({ description: "The information to remember" }),
          userId: Type.Optional(Type.String()),
        }),
        async execute(_id, params) {
          const { text, userId } = params as { text: string; userId?: string };
          try {
            const resp = await bridgePost<{ results: AddResultItem[] }>(`${base}/add`, {
              messages: [{ role: "user", content: text }],
              user_id: userId || cfg.userId,
              run_id: currentSessionId,
            });
            const results = resp.results ?? [];
            const added = results.filter((r) => r.event === "ADD" || r.event === "add");
            const updated = results.filter((r) => r.event === "UPDATE" || r.event === "update");
            const summary = [];
            if (added.length) summary.push(`${added.length} added`);
            if (updated.length) summary.push(`${updated.length} updated`);
            if (!summary.length) summary.push("processed");
            return {
              content: [{ type: "text", text: `Stored: ${summary.join(", ")}` }],
              details: { results },
            };
          } catch (err) {
            return { content: [{ type: "text", text: `Memory store failed: ${String(err)}` }], details: { error: String(err) } };
          }
        },
      },
      { name: "memory_store" },
    );

    api.registerTool(
      {
        name: "memory_get",
        label: "Memory Get",
        description: "Retrieve a specific memory by its ID.",
        parameters: Type.Object({
          memoryId: Type.String({ description: "The memory ID to retrieve" }),
        }),
        async execute(_id, params) {
          const { memoryId } = params as { memoryId: string };
          try {
            const memory = await bridgeGet<MemoryItem>(`${base}/get/${encodeURIComponent(memoryId)}`);
            return {
              content: [{ type: "text", text: `Memory ${memory.id}:\n${memory.memory}\n\nCreated: ${memory.created_at ?? "unknown"}` }],
              details: { memory },
            };
          } catch (err) {
            return { content: [{ type: "text", text: `Memory get failed: ${String(err)}` }], details: { error: String(err) } };
          }
        },
      },
      { name: "memory_get" },
    );

    api.registerTool(
      {
        name: "memory_list",
        label: "Memory List",
        description: "List all stored memories for a user.",
        parameters: Type.Object({
          userId: Type.Optional(Type.String()),
        }),
        async execute(_id, params) {
          const { userId } = params as { userId?: string };
          try {
            const resp = await bridgePost<{ results: MemoryItem[] }>(`${base}/list`, {
              user_id: userId || cfg.userId,
            });
            const items = resp.results ?? [];
            if (!items.length) {
              return { content: [{ type: "text", text: "No memories stored yet." }], details: { count: 0 } };
            }
            const text = items.map((r, i) => `${i + 1}. ${r.memory} (id: ${r.id})`).join("\n");
            return {
              content: [{ type: "text", text: `${items.length} memories:\n\n${text}` }],
              details: { count: items.length, memories: items },
            };
          } catch (err) {
            return { content: [{ type: "text", text: `Memory list failed: ${String(err)}` }], details: { error: String(err) } };
          }
        },
      },
      { name: "memory_list" },
    );

    api.registerTool(
      {
        name: "memory_forget",
        label: "Memory Forget",
        description: "Delete a memory by ID, or search and delete by keyword.",
        parameters: Type.Object({
          memoryId: Type.Optional(Type.String({ description: "Memory ID to delete directly" })),
          query: Type.Optional(Type.String({ description: "Search query to find and delete memory" })),
        }),
        async execute(_id, params) {
          const { memoryId, query } = params as { memoryId?: string; query?: string };
          try {
            if (memoryId) {
              await bridgeDelete(`${base}/del/${encodeURIComponent(memoryId)}`);
              return { content: [{ type: "text", text: `Deleted memory ${memoryId}` }], details: { action: "deleted", id: memoryId } };
            }
            if (query) {
              const resp = await bridgePost<{ results: MemoryItem[] }>(`${base}/search`, {
                query,
                user_id: cfg.userId,
                limit: 5,
              });
              const items = resp.results ?? [];
              if (!items.length) {
                return { content: [{ type: "text", text: "No matching memories found." }], details: { found: 0 } };
              }
              if (items.length === 1 || (items[0].score ?? 0) > 0.9) {
                await bridgeDelete(`${base}/del/${encodeURIComponent(items[0].id)}`);
                return {
                  content: [{ type: "text", text: `Forgotten: "${items[0].memory}"` }],
                  details: { action: "deleted", id: items[0].id },
                };
              }
              const list = items.map((r) => `- [${r.id}] ${r.memory.slice(0, 80)} (score: ${((r.score ?? 0) * 100).toFixed(0)}%)`).join("\n");
              return {
                content: [{ type: "text", text: `Found ${items.length} candidates. Provide memoryId to delete:\n${list}` }],
                details: { action: "candidates", candidates: items },
              };
            }
            return { content: [{ type: "text", text: "Provide memoryId or query." }], details: { error: "missing_param" } };
          } catch (err) {
            return { content: [{ type: "text", text: `Memory forget failed: ${String(err)}` }], details: { error: String(err) } };
          }
        },
      },
      { name: "memory_forget" },
    );

    // ── CLI ──────────────────────────────────────────────────────────────

    api.registerCli(
      ({ program }) => {
        const memx = program.command("memx").description("memx memory plugin commands");

        memx
          .command("search")
          .description("Search memories in memx")
          .argument("<query>", "Search query")
          .option("--limit <n>", "Max results", String(cfg.topK))
          .action(async (query: string, opts: { limit: string }) => {
            try {
              const resp = await bridgePost<{ results: MemoryItem[] }>(`${base}/search`, {
                query,
                user_id: cfg.userId,
                limit: parseInt(opts.limit, 10),
              });
              const items = resp.results ?? [];
              if (!items.length) { console.log("No memories found."); return; }
              console.log(JSON.stringify(items.map((r) => ({ id: r.id, memory: r.memory, score: r.score, created_at: r.created_at })), null, 2));
            } catch (err) {
              console.error(`Search failed: ${String(err)}`);
            }
          });

        memx
          .command("stats")
          .description("Show memory statistics")
          .action(async () => {
            try {
              const resp = await bridgePost<{ results: MemoryItem[] }>(`${base}/list`, { user_id: cfg.userId });
              const total = (resp.results ?? []).length;
              const stats = await bridgeGet<Record<string, unknown>>(`${base}/stats`);
              console.log(`User: ${cfg.userId}`);
              console.log(`Total memories: ${total}`);
              console.log(`Bridge: ${base}`);
              console.log(`Auto-recall: ${cfg.autoRecall}, Auto-capture: ${cfg.autoCapture}`);
              if (stats && typeof stats === "object") {
                console.log("Backend stats:", JSON.stringify(stats, null, 2));
              }
            } catch (err) {
              console.error(`Stats failed: ${String(err)}`);
            }
          });
      },
      { commands: ["memx"] },
    );

    // ── Auto-recall hook ─────────────────────────────────────────────────

    if (cfg.autoRecall) {
      api.on("before_agent_start", async (event, ctx) => {
        if (!event.prompt || event.prompt.length < 5) return;

        const sessionId = (ctx as Record<string, unknown>)?.sessionKey as string | undefined;
        if (sessionId) currentSessionId = sessionId;

        try {
          const resp = await bridgePost<{ results: MemoryItem[] }>(`${base}/search`, {
            query: event.prompt,
            user_id: cfg.userId,
            limit: cfg.topK,
            threshold: cfg.searchThreshold,
          });
          const items = resp.results ?? [];
          if (!items.length) return;

          const memoryContext = items.map((r) => `- ${r.memory}`).join("\n");
          api.logger.info(`openclaw-memx: injecting ${items.length} memories into context`);

          return {
            prependContext: `<relevant-memories>\nThe following memories may be relevant to this conversation:\n${memoryContext}\n</relevant-memories>`,
          };
        } catch (err) {
          api.logger.warn(`openclaw-memx: recall failed: ${String(err)}`);
        }
      });
    }

    // ── Auto-capture hook ────────────────────────────────────────────────

    if (cfg.autoCapture) {
      api.on("agent_end", async (event, ctx) => {
        if (!event.success || !event.messages || event.messages.length === 0) return;

        const sessionId = (ctx as Record<string, unknown>)?.sessionKey as string | undefined;
        if (sessionId) currentSessionId = sessionId;

        try {
          const recentMessages = event.messages.slice(-10);
          const formatted: Array<{ role: string; content: string }> = [];

          for (const msg of recentMessages) {
            if (!msg || typeof msg !== "object") continue;
            const m = msg as Record<string, unknown>;
            const role = m.role;
            if (role !== "user" && role !== "assistant") continue;

            let text = "";
            const content = m.content;
            if (typeof content === "string") {
              text = content;
            } else if (Array.isArray(content)) {
              for (const block of content) {
                if (block && typeof block === "object" && "text" in block && typeof (block as Record<string, unknown>).text === "string") {
                  text += (text ? "\n" : "") + (block as Record<string, unknown>).text;
                }
              }
            }

            if (!text) continue;
            if (text.includes("<relevant-memories>")) {
              text = text.replace(/<relevant-memories>[\s\S]*?<\/relevant-memories>\s*/g, "").trim();
              if (!text) continue;
            }
            formatted.push({ role: role as string, content: text });
          }

          if (!formatted.length) return;

          const resp = await bridgePost<{ results: AddResultItem[] }>(`${base}/add`, {
            messages: formatted,
            user_id: cfg.userId,
            run_id: currentSessionId,
          });
          const count = (resp.results ?? []).length;
          if (count > 0) {
            api.logger.info(`openclaw-memx: auto-captured ${count} memories`);
          }
        } catch (err) {
          api.logger.warn(`openclaw-memx: capture failed: ${String(err)}`);
        }
      });
    }

    // ── Service lifecycle ────────────────────────────────────────────────

    api.registerService({
      id: "openclaw-memx",
      start: async () => {
        // Verify bridge is reachable
        try {
          const health = await bridgeGet<{ status: string }>(`${base}/health`);
          api.logger.info(
            `openclaw-memx: initialized (bridge: ${base}, status: ${health.status}, user: ${cfg.userId}, autoRecall: ${cfg.autoRecall}, autoCapture: ${cfg.autoCapture})`,
          );
        } catch (err) {
          api.logger.warn(`openclaw-memx: bridge unreachable at ${base} — ${String(err)}`);
        }
      },
      stop: () => {
        api.logger.info("openclaw-memx: stopped");
      },
    });
  },
};

export default memxPlugin;
