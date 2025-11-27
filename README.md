# jscan - JavaScript/TypeScript Code Quality Analyzer

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/ludo-technologies/jscan)

## jscan is a code quality analyzer for JavaScript/TypeScript vibe coders.

Building with Cursor, Claude, or ChatGPT? jscan performs structural analysis to keep your codebase maintainable.

Sister project: [pyscn](https://github.com/ludo-technologies/pyscn) - Python Code Quality Analyzer

## Features

- **Complexity Analysis**: Calculate McCabe cyclomatic complexity for functions
- **Dead Code Detection**: Find unreachable code blocks
- **JavaScript Support**: Full support for ES6+ JavaScript syntax
- **TypeScript Support**: Full parser support for TypeScript
- **Fast Performance**: Built in Go with tree-sitter for efficient parsing

## Installation

### From Source

```bash
git clone https://github.com/ludo-technologies/jscan.git
cd jscan
go build -o jscan ./cmd/jscan
```

### Quick Install

```bash
go install github.com/ludo-technologies/jscan/cmd/jscan@latest
```

## Usage

### Analyze a file or directory

```bash
# Analyze a single file
jscan analyze src/index.js

# Analyze a directory
jscan analyze src/

# Run specific analyses
jscan analyze --select complexity src/
jscan analyze --select deadcode src/
jscan analyze --select complexity,deadcode src/
```

### Example Output

```
Analyzing 1 files...

src/index.js:
  Complexity Analysis:
    calculateTotal: complexity=5, risk=medium
    processData: complexity=2, risk=low
  Dead Code Analysis:
    calculateTotal: 1 dead code blocks found
      Line 42: Code after return statement is unreachable

Analysis complete!
Files analyzed: 1
```

## Architecture

jscan follows Clean Architecture principles:

```
jscan/
├── cmd/jscan/           # CLI entry point
├── domain/              # Domain models
├── internal/
│   ├── parser/         # tree-sitter JavaScript/TypeScript parser
│   ├── analyzer/       # CFG, complexity, dead code analysis
│   ├── config/         # Configuration management
│   └── reporter/       # Output formatting
└── testdata/           # Test fixtures
```

## Development Status

**Current Version: v0.1.0-alpha**

jscan is in early development. Currently implemented:
- ✅ Complexity Analysis
- ✅ Dead Code Detection
- ✅ Basic CLI

## Roadmap

Planned features:
- Clone Detection (APTED + LSH-based structural similarity)
- Module Analysis & Coupling Metrics
- TypeScript-specific analysis features
- Additional output formats (JSON, HTML, CSV)
- npm package distribution

## Configuration

jscan uses JSON-based configuration files, following JavaScript ecosystem conventions.

Create a `jscan.config.json` or `.jscanrc.json` file in your project root:

```json
{
  "complexity": {
    "lowThreshold": 10,
    "mediumThreshold": 20,
    "enabled": true
  },
  "deadCode": {
    "enabled": true,
    "minSeverity": "warning"
  },
  "output": {
    "format": "text",
    "showDetails": true
  }
}
```

See `jscan.config.example.json` for all available options.

## Testing

```bash
# Run unit tests
go test ./...

# Test on sample files
./jscan analyze testdata/javascript/simple/

# Run with Makefile
make test
make run
```

## License

MIT License - see [LICENSE](LICENSE) file for details

## Author

Created by [@daisukeyoda](https://github.com/daisukeyoda)

Sister project: [pyscn](https://github.com/ludo-technologies/pyscn) - Python Code Quality Analyzer
