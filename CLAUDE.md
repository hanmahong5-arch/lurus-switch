# Lurus Switch

Wails desktop app — AI CLI config package generator.

Go 1.22+ / Wails v2 / React 18 + TypeScript + Tailwind + Zustand + Monaco Editor.

## Commands

```bash
wails dev                          # Dev with hot reload
wails build                        # Build
go test -v ./...                   # Backend tests
cd frontend && bun run build       # Frontend only
```

## Structure

```
app.go              # Wails bindings (Go ↔ Frontend)
internal/
  config/           # Data models (Claude/Codex/Gemini configs)
  generator/        # Config file generation (JSON/TOML/Markdown)
  packager/         # Standalone executable packaging
  validator/        # Config validation
frontend/
  src/pages/        # ClaudePage, CodexPage, GeminiPage
  src/stores/       # Zustand state management
```

## BMAD

| Resource | Path |
|----------|------|
| Architecture | `./_bmad-output/planning-artifacts/architecture.md` |
