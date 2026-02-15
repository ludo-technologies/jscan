# jscan - JavaScript/TypeScript Code Quality Analyzer

[![CI](https://github.com/ludo-technologies/jscan/actions/workflows/ci.yml/badge.svg)](https://github.com/ludo-technologies/jscan/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Active_Development-brightgreen.svg)](https://github.com/ludo-technologies/jscan)

## jscan is a code quality analyzer for JavaScript/TypeScript vibe coders.

Building with Cursor, Claude, or ChatGPT? jscan performs structural analysis to keep your codebase maintainable.

Sister project: [pyscn](https://github.com/ludo-technologies/pyscn) - Python Code Quality Analyzer

## Features

- **Complexity Analysis** - McCabe cyclomatic complexity with risk-level scoring
- **Dead Code Detection** - Unreachable code, unused imports/exports, and orphan files
- **Clone Detection** - Duplicate code identification using APTED tree edit distance + MinHash/LSH
- **Dependency Analysis** - Module dependency graph with circular dependency detection
- **CBO Metrics** - Coupling Between Objects measurement for module health
- **Health Score** - Lighthouse-style overall project health scoring
- **Multiple Output Formats** - HTML, JSON, CSV, and DOT (for graph visualization)
- **`init` / `check` Commands** - Config scaffolding and CI/CD threshold enforcement
- **Parallel Execution** - Concurrent file analysis for fast performance on large codebases
- **Built with Go + tree-sitter** - Fast, error-tolerant parsing for ES6+ JavaScript and TypeScript

## Installation

### npm (recommended)

```bash
# Run without installing
npx jscan analyze src/

# Install globally
npm install -g jscan
```

### From Source

```bash
git clone https://github.com/ludo-technologies/jscan.git
cd jscan
go build -o jscan ./cmd/jscan
```

### Go Install

```bash
go install github.com/ludo-technologies/jscan/cmd/jscan@latest
```

## Usage

### Analyze a file or directory

```bash
# Analyze a directory (default: HTML report)
jscan analyze src/

# Analyze a single file
jscan analyze src/index.js

# Run specific analyses
jscan analyze --select complexity src/
jscan analyze --select deadcode src/
jscan analyze --select complexity,deadcode,clones src/

# Choose output format
jscan analyze --format json src/
jscan analyze --format csv src/
jscan analyze --format text src/
```

### Initialize configuration

```bash
# Generate a jscan.config.json with defaults
jscan init
```

### CI/CD health check

```bash
# Fail if health score drops below thresholds
jscan check src/
```

### Dependency visualization

```bash
# Output DOT graph for dependency analysis
jscan deps src/ --format dot | dot -Tsvg -o deps.svg
```

### Example Output

```
Analyzing 3 files...

src/index.js:
  Complexity Analysis:
    calculateTotal: complexity=5, risk=medium
    processData: complexity=2, risk=low
  Dead Code Analysis:
    calculateTotal: 1 dead code blocks found
      Line 42: Code after return statement is unreachable

Health Score: 72/100

Analysis complete!
Files analyzed: 3
```

## Configuration

jscan uses JSON-based configuration files. Run `jscan init` to generate one, or create a `jscan.config.json` / `.jscanrc.json` in your project root:

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

See `jscan.config.example.json` for all available options.

## Architecture

jscan uses a layered architecture inspired by Clean Architecture:

```
cmd -> service -> internal -> domain
cmd -> app -> service -> internal -> domain
```

The `domain` package stays dependency-free and is shared across all layers.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full design documentation.

## Roadmap

- TypeScript-specific analysis features (type-aware dead code, generic complexity)
- Vue / JSX single-file component support
- IDE / editor integrations
- Watch mode for continuous analysis

## Development

```bash
# Run unit tests
go test ./...

# Lint
make lint

# Build
make build

# Test on sample files
./jscan analyze testdata/javascript/simple/
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding conventions, and pull request guidelines.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

Created by [@daisukeyoda](https://github.com/daisukeyoda)

Sister project: [pyscn](https://github.com/ludo-technologies/pyscn) - Python Code Quality Analyzer
