# CLAUDE.md - Lurus Switch

## Project Overview

Lurus Switch is a Wails desktop application for generating configuration packages for AI CLI tools (Claude Code, Codex, Gemini CLI).

## Tech Stack

- **Backend**: Go 1.22+ with Wails v2
- **Frontend**: React 18 + TypeScript + Tailwind CSS
- **State Management**: Zustand
- **UI Components**: Radix UI + Lucide Icons
- **Code Editor**: Monaco Editor

## Key Commands

```bash
# Development
wails dev

# Build
wails build

# Generate Wails bindings
wails generate module

# Run Go tests
go test -v ./...

# Build frontend only
cd frontend && npm run build
```

## Architecture

```
app.go          → Exposes Go methods to frontend via Wails
internal/
  config/       → Data models for Claude/Codex/Gemini configs
  generator/    → Generates config files (JSON/TOML/Markdown)
  packager/     → Creates standalone executables
  validator/    → Validates configurations
frontend/
  src/pages/    → ClaudePage, CodexPage, GeminiPage
  src/stores/   → Zustand store for state management
```

## Code Conventions

- Go: Standard Go formatting (`gofmt`)
- TypeScript: ESLint + Prettier
- Comments in English
- Documentation in Chinese & English

## Testing

```bash
# Run all Go tests
go test -v ./...

# Run frontend type check
cd frontend && npm run build
```
