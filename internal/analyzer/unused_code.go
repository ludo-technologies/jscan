package analyzer

import (
	"path/filepath"
	"strings"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/parser"
)

// DetectUnusedImports detects imported names that are never referenced in the file.
// It walks the AST to collect all identifier references (excluding import/export declarations)
// and compares them against the locally-bound import names.
func DetectUnusedImports(ast *parser.Node, moduleInfo *domain.ModuleInfo, filePath string) []*DeadCodeFinding {
	if ast == nil || moduleInfo == nil {
		return nil
	}

	// Collect local names from imports (skip side-effect, type-only, dynamic)
	type importEntry struct {
		localName string
		line      int
		source    string
	}
	var importedNames []importEntry

	for _, imp := range moduleInfo.Imports {
		// Skip side-effect imports (import 'polyfill')
		if imp.ImportType == domain.ImportTypeSideEffect {
			continue
		}
		// Skip type-only imports (import type { Foo } from 'bar')
		if imp.IsTypeOnly || imp.ImportType == domain.ImportTypeTypeOnly {
			continue
		}
		// Skip dynamic imports (import('foo'))
		if imp.IsDynamic || imp.ImportType == domain.ImportTypeDynamic {
			continue
		}

		for _, spec := range imp.Specifiers {
			// Skip type-only specifiers
			if spec.IsType {
				continue
			}
			if spec.Local != "" {
				importedNames = append(importedNames, importEntry{
					localName: spec.Local,
					line:      imp.Location.StartLine,
					source:    imp.Source,
				})
			}
		}
	}

	if len(importedNames) == 0 {
		return nil
	}

	// Walk the AST and collect all Identifier references,
	// skipping import declaration subtrees (which define the local bindings).
	// Export declarations are NOT skipped because they reference imported names
	// (e.g. `export { foo }` or `export default foo` means foo is used).
	referenced := make(map[string]bool)
	ast.Walk(func(n *parser.Node) bool {
		// Skip import declaration subtrees only
		if n.Type == parser.NodeImportDeclaration {
			return false
		}

		if n.Type == parser.NodeIdentifier && n.Name != "" {
			referenced[n.Name] = true
		}

		// ExportSpecifier nodes reference local names (e.g. `export { foo }`)
		// The Name field holds the local identifier being exported
		if n.Type == parser.NodeExportSpecifier && n.Name != "" {
			referenced[n.Name] = true
		}

		// Also check JSX element tags which reference identifiers
		if n.Type == parser.NodeJSXElement && n.Name != "" {
			referenced[n.Name] = true
		}

		return true
	})

	// Generate findings for unreferenced imports
	var findings []*DeadCodeFinding
	for _, entry := range importedNames {
		if !referenced[entry.localName] {
			findings = append(findings, &DeadCodeFinding{
				FilePath:  filePath,
				StartLine: entry.line,
				EndLine:   entry.line,
				Reason:    ReasonUnusedImport,
				Severity:  SeverityLevelWarning,
				Description: "Imported name '" + entry.localName + "' from '" +
					entry.source + "' is never used",
			})
		}
	}

	return findings
}

// DetectUnusedExports detects exported names that are not imported by any other analyzed file.
// It builds a reverse index of all imports across files and checks each export against it.
func DetectUnusedExports(allModuleInfos map[string]*domain.ModuleInfo, analyzedFiles map[string]bool) []*DeadCodeFinding {
	if len(allModuleInfos) == 0 {
		return nil
	}

	// Build reverse index: resolved source path → set of imported names
	// This tells us which names are imported from each file
	importedNamesFromFile := make(map[string]map[string]bool)

	for importingFile, info := range allModuleInfos {
		for _, imp := range info.Imports {
			if imp.SourceType != domain.ModuleTypeRelative {
				continue
			}

			resolvedPath := resolveImportPath(importingFile, imp.Source, analyzedFiles)
			if resolvedPath == "" {
				continue
			}

			if importedNamesFromFile[resolvedPath] == nil {
				importedNamesFromFile[resolvedPath] = make(map[string]bool)
			}

			// Track which names are imported
			switch imp.ImportType {
			case domain.ImportTypeNamespace:
				// import * as X — marks the whole module as "used"
				importedNamesFromFile[resolvedPath]["*"] = true
			case domain.ImportTypeDefault, domain.ImportTypeNamed:
				for _, spec := range imp.Specifiers {
					name := spec.Imported
					if name == "" {
						name = spec.Local
					}
					importedNamesFromFile[resolvedPath][name] = true
				}
			case domain.ImportTypeSideEffect:
				// Side-effect import means the file is "used"
				importedNamesFromFile[resolvedPath]["*"] = true
			}
		}
	}

	var findings []*DeadCodeFinding

	for filePath, info := range allModuleInfos {
		// Skip entry-point files whose exports are meant to be public
		if isEntryPointFile(filePath) {
			continue
		}
		// Skip test files
		if isTestFile(filePath) {
			continue
		}

		importedNames := importedNamesFromFile[filePath]

		// If namespace import (*) exists, all exports are considered used
		if importedNames != nil && importedNames["*"] {
			continue
		}

		for _, exp := range info.Exports {
			// Skip re-exports (export { x } from './other')
			if exp.Source != "" {
				continue
			}
			// Skip type-only exports
			if exp.IsTypeOnly {
				continue
			}
			// Skip export * (re-export all)
			if exp.ExportType == "all" {
				continue
			}

			// Determine the exported name(s)
			exportedNames := getExportedNames(exp)

			for _, name := range exportedNames {
				if importedNames == nil || !importedNames[name] {
					findings = append(findings, &DeadCodeFinding{
						FilePath:    filePath,
						StartLine:   exp.Location.StartLine,
						EndLine:     exp.Location.EndLine,
						Reason:      ReasonUnusedExport,
						Severity:    SeverityLevelInfo,
						Description: "Export '" + name + "' is not imported by any other analyzed file",
					})
				}
			}
		}
	}

	return findings
}

// getExportedNames extracts the exported name(s) from an export declaration.
func getExportedNames(exp *domain.Export) []string {
	var names []string

	// Named exports with specifiers: export { foo, bar }
	if len(exp.Specifiers) > 0 {
		for _, spec := range exp.Specifiers {
			if spec.IsType {
				continue
			}
			name := spec.Exported
			if name == "" {
				name = spec.Local
			}
			if name != "" {
				names = append(names, name)
			}
		}
		return names
	}

	// Default export
	if exp.ExportType == "default" {
		return []string{"default"}
	}

	// Declaration export: export function foo() / export const bar
	if exp.Name != "" {
		return []string{exp.Name}
	}

	return nil
}

// resolveImportPath resolves a relative import source to an actual file path.
// It tries the raw path, then common extensions, then index files.
func resolveImportPath(importingFile, source string, knownFiles map[string]bool) string {
	// Only handle relative imports
	if !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "../") {
		return ""
	}

	dir := filepath.Dir(importingFile)
	resolved := filepath.Join(dir, source)
	resolved = filepath.Clean(resolved)

	// Try exact path first
	if knownFiles[resolved] {
		return resolved
	}

	// Try adding extensions
	extensions := []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".cts", ".mjs", ".cjs"}
	for _, ext := range extensions {
		candidate := resolved + ext
		if knownFiles[candidate] {
			return candidate
		}
	}

	// Try as directory with index files
	indexFiles := []string{
		"index.ts", "index.tsx", "index.js", "index.jsx",
		"index.mts", "index.cts", "index.mjs", "index.cjs",
	}
	for _, idx := range indexFiles {
		candidate := filepath.Join(resolved, idx)
		if knownFiles[candidate] {
			return candidate
		}
	}

	return ""
}

// isEntryPointFile checks if a file is an entry point (barrel file / index file).
func isEntryPointFile(filePath string) bool {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	return nameWithoutExt == "index"
}

// isTestFile checks if a file is a test file.
func isTestFile(filePath string) bool {
	base := filepath.Base(filePath)

	// Check for *.test.* and *.spec.* patterns
	parts := strings.Split(base, ".")
	for _, part := range parts {
		if part == "test" || part == "spec" {
			return true
		}
	}

	// Check for __tests__ directory
	if strings.Contains(filePath, "__tests__") {
		return true
	}

	return false
}
