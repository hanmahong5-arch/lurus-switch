# Lurus Switch

AI CLI Configuration Generator - A desktop application for generating configuration packages for AI CLI tools.

## Supported Tools

| Tool | Repository | Config Format | Packaging |
|------|------------|---------------|-----------|
| **Claude Code** | [anthropics/claude-code](https://github.com/anthropics/claude-code) | JSON (settings.json) | Bun compile |
| **Codex** | [openai/codex](https://github.com/openai/codex) | TOML (config.toml) | Rust binary |
| **Gemini CLI** | [google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli) | Markdown (GEMINI.md) | Node.js pkg |

## Features

- Visual configuration editor for each AI CLI tool
- Real-time configuration preview
- Configuration validation
- Export configurations to files
- Save/load configuration presets
- Package configurations into standalone executables (experimental)

## Development

### Prerequisites

- Go 1.22+
- Node.js 18+ (or Bun)
- Wails CLI v2.x

### Setup

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install frontend dependencies
cd frontend && npm install

# Run in development mode
wails dev
```

### Build

```bash
# Build for production
wails build

# Build with debug info
wails build -debug
```

## Project Structure

```
lurus-switch/
├── main.go                      # Wails entry point
├── app.go                       # Go methods exposed to frontend
├── wails.json                   # Wails configuration
│
├── internal/
│   ├── config/                  # Configuration models
│   │   ├── store.go             # Config persistence
│   │   ├── claude.go            # Claude Code schema
│   │   ├── codex.go             # Codex schema
│   │   └── gemini.go            # Gemini CLI schema
│   │
│   ├── generator/               # Config file generators
│   │   ├── claude_generator.go  # JSON generator
│   │   ├── codex_generator.go   # TOML generator
│   │   └── gemini_generator.go  # Markdown generator
│   │
│   ├── packager/                # Executable packagers
│   │   ├── bun_packager.go      # Bun compile
│   │   ├── rust_packager.go     # Rust binary download
│   │   └── node_packager.go     # Node.js pkg
│   │
│   ├── downloader/              # GitHub Release downloader
│   └── validator/               # Config validation
│
├── frontend/                    # React + TypeScript
│   ├── src/
│   │   ├── pages/               # Page components
│   │   ├── components/          # UI components
│   │   └── stores/              # Zustand stores
│   └── package.json
│
└── doc/                         # Documentation
```

## Configuration Storage

User configurations are stored in platform-specific locations:

- **Windows**: `%APPDATA%\lurus-switch\configs\`
- **macOS**: `~/Library/Application Support/lurus-switch/configs/`
- **Linux**: `~/.config/lurus-switch/configs/`

## License

MIT
