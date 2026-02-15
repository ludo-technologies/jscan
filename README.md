<div align="center">

# jscan

**A code quality analyzer for JavaScript/TypeScript vibe coders.**

Building with Cursor, Claude, or ChatGPT? jscan performs structural analysis to keep your codebase maintainable.

[![CI](https://github.com/ludo-technologies/jscan/actions/workflows/ci.yml/badge.svg)](https://github.com/ludo-technologies/jscan/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/jscan?style=flat-square&logo=npm)](https://www.npmjs.com/package/jscan)
[![Downloads](https://img.shields.io/npm/dm/jscan?style=flat-square&logo=npm&label=downloads)](https://www.npmjs.com/package/jscan)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/ludo-technologies/jscan?style=flat-square)](LICENSE)

*Working with Python? Check out [pyscn](https://github.com/ludo-technologies/pyscn)*

</div>

## Quick Start

```bash
# Run analysis without installation
npx jscan analyze src/
```

## Demo

https://github.com/user-attachments/assets/6c491b52-99d3-4fa4-b628-e09c0b61451d

## Features

- 🔍 **Dead code detection** – Unreachable code, unused imports/exports, and orphan files
- 📋 **Multi-algorithm clone detection** – Duplicate code identification using APTED tree edit distance + MinHash/LSH
- 🔗 **Coupling metrics (CBO)** – Track architecture quality and module dependencies
- 📊 **Cyclomatic complexity analysis** – McCabe complexity with risk-level scoring
- 🏥 **Health score** – Lighthouse-style overall project health scoring
- 🔄 **Dependency analysis** – Module dependency graph with circular dependency detection

**Parallel execution** • **Multiple output formats (HTML, JSON, CSV, DOT)** • Built with Go + tree-sitter

## Installation

```bash
# Install globally with npm (recommended)
npm install -g jscan
```

<details>
<summary>Alternative installation methods</summary>

### Build from source

```bash
git clone https://github.com/ludo-technologies/jscan.git
cd jscan
go build -o jscan ./cmd/jscan
```

### Go install

```bash
go install github.com/ludo-technologies/jscan/cmd/jscan@latest
```

</details>

## Common Commands

### `jscan analyze`

Run comprehensive analysis with HTML report

```bash
jscan analyze src/                              # All analyses with HTML report
jscan analyze --format json src/                # Generate JSON report
jscan analyze --select complexity src/          # Only complexity analysis
jscan analyze --select deadcode src/            # Only dead code analysis
jscan analyze --select complexity,deadcode,clones src/  # Multiple analyses
```

### `jscan check`

Fast CI-friendly quality gate

```bash
jscan check src/                         # Quick pass/fail check
```

### `jscan init`

Create configuration file

```bash
jscan init                               # Generate jscan.config.json
```

### `jscan deps`

Dependency visualization

```bash
jscan deps src/ --format dot | dot -Tsvg -o deps.svg
```

> 💡 Run `jscan --help` or `jscan <command> --help` for complete options

## Configuration

Create a `jscan.config.json` or `.jscanrc.json` in your project root:

```json
{
  "complexity": {
    "low_threshold": 10,
    "medium_threshold": 20,
    "enabled": true
  },
  "dead_code": {
    "enabled": true,
    "min_severity": "warning"
  },
  "output": {
    "format": "text",
    "show_details": true
  }
}
```

> ⚙️ Run `jscan init` to generate a configuration file with all available options

## Roadmap

- TypeScript-specific analysis features (type-aware dead code, generic complexity)
- Vue / JSX single-file component support
- IDE / editor integrations
- Watch mode for continuous analysis

---

## Documentation

📚 **[Development Guide](docs/DEVELOPMENT.md)** • **[Architecture](docs/ARCHITECTURE.md)** • **[Testing](docs/TESTING.md)** • **[Contributing](CONTRIBUTING.md)**

## Enterprise Support

For commercial support, custom integrations, or consulting services, contact us at contact@ludo-tech.org

## License

MIT License — see [LICENSE](LICENSE)

---

*Built with ❤️ using Go and tree-sitter*
