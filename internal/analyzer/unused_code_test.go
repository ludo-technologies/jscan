package analyzer

import (
	"testing"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/parser"
)

// helper to parse JS source and get module info + AST
func parseAndAnalyze(t *testing.T, source string) (*parser.Node, *domain.ModuleInfo) {
	t.Helper()
	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	ma := NewModuleAnalyzer(DefaultModuleAnalyzerConfig())
	info, err := ma.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze module: %v", err)
	}

	return ast, info
}

// --- Unused Import Tests ---

func TestDetectUnusedImports_AllUsed(t *testing.T) {
	source := `
import { useState, useEffect } from 'react';

const [count, setCount] = useState(0);
useEffect(() => {}, []);
`
	ast, info := parseAndAnalyze(t, source)
	findings := DetectUnusedImports(ast, info, "test.js")

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings when all imports are used, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s", f.Description)
		}
	}
}

func TestDetectUnusedImports_OneUnused(t *testing.T) {
	source := `
import { useState, useEffect } from 'react';

const [count, setCount] = useState(0);
`
	ast, info := parseAndAnalyze(t, source)
	findings := DetectUnusedImports(ast, info, "test.js")

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding for unused useEffect, got %d", len(findings))
	}

	if findings[0].Reason != ReasonUnusedImport {
		t.Errorf("Expected reason %s, got %s", ReasonUnusedImport, findings[0].Reason)
	}
	if findings[0].Severity != SeverityLevelWarning {
		t.Errorf("Expected severity warning, got %s", findings[0].Severity)
	}
}

func TestDetectUnusedImports_DefaultUnused(t *testing.T) {
	source := `
import React from 'react';
import { useState } from 'react';

const x = useState(0);
`
	ast, info := parseAndAnalyze(t, source)
	findings := DetectUnusedImports(ast, info, "test.js")

	// React (default import) should be unused
	found := false
	for _, f := range findings {
		if f.Reason == ReasonUnusedImport {
			found = true
		}
	}
	if !found {
		t.Error("Expected at least one unused import finding for default import 'React'")
	}
}

func TestDetectUnusedImports_SideEffectSkipped(t *testing.T) {
	source := `
import 'polyfill';

console.log('hello');
`
	ast, info := parseAndAnalyze(t, source)
	findings := DetectUnusedImports(ast, info, "test.js")

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for side-effect import, got %d", len(findings))
	}
}

func TestDetectUnusedImports_TypeOnlySkipped(t *testing.T) {
	// When IsTypeOnly is set on the import, it should be skipped.
	// We test this with manually constructed ModuleInfo since the parser
	// does not yet fully detect `import type` syntax.
	ast := &parser.Node{
		Type: parser.NodeProgram,
		Body: []*parser.Node{
			{Type: parser.NodeExpressionStatement, Children: []*parser.Node{
				{Type: parser.NodeIdentifier, Name: "x"},
			}},
		},
	}

	info := &domain.ModuleInfo{
		FilePath: "test.ts",
		Imports: []*domain.Import{
			{
				Source:     "./types",
				SourceType: domain.ModuleTypeRelative,
				ImportType: domain.ImportTypeTypeOnly,
				IsTypeOnly: true,
				Specifiers: []domain.ImportSpecifier{
					{Imported: "Foo", Local: "Foo"},
				},
			},
		},
	}

	findings := DetectUnusedImports(ast, info, "test.ts")

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for type-only import, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s", f.Description)
		}
	}
}

func TestDetectUnusedImports_NilInputs(t *testing.T) {
	findings := DetectUnusedImports(nil, nil, "test.js")
	if findings != nil {
		t.Errorf("Expected nil findings for nil inputs, got %d", len(findings))
	}
}

func TestDetectUnusedImports_NoImports(t *testing.T) {
	source := `
const x = 1;
const y = 2;
`
	ast, info := parseAndAnalyze(t, source)
	findings := DetectUnusedImports(ast, info, "test.js")

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings when no imports, got %d", len(findings))
	}
}

// --- Unused Export Tests ---

func TestDetectUnusedExports_AllImported(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/utils.js": {
			FilePath: "/src/utils.js",
			Exports: []*domain.Export{
				{
					ExportType: "declaration",
					Name:       "helper",
					Location:   domain.SourceLocation{StartLine: 1, EndLine: 1},
				},
			},
		},
		"/src/app.js": {
			FilePath: "/src/app.js",
			Imports: []*domain.Import{
				{
					Source:     "./utils",
					SourceType: domain.ModuleTypeRelative,
					ImportType: domain.ImportTypeNamed,
					Specifiers: []domain.ImportSpecifier{
						{Imported: "helper", Local: "helper"},
					},
				},
			},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/utils.js": true,
		"/src/app.js":   true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings when export is imported, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s", f.Description)
		}
	}
}

func TestDetectUnusedExports_NeverImported(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/utils.js": {
			FilePath: "/src/utils.js",
			Exports: []*domain.Export{
				{
					ExportType: "declaration",
					Name:       "unusedHelper",
					Location:   domain.SourceLocation{StartLine: 5, EndLine: 5},
				},
			},
		},
		"/src/app.js": {
			FilePath: "/src/app.js",
			Imports:  []*domain.Import{},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/utils.js": true,
		"/src/app.js":   true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding for unused export, got %d", len(findings))
	}

	if findings[0].Reason != ReasonUnusedExport {
		t.Errorf("Expected reason %s, got %s", ReasonUnusedExport, findings[0].Reason)
	}
	if findings[0].Severity != SeverityLevelInfo {
		t.Errorf("Expected severity info, got %s", findings[0].Severity)
	}
}

func TestDetectUnusedExports_ReExportSkipped(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/utils.js": {
			FilePath: "/src/utils.js",
			Exports: []*domain.Export{
				{
					ExportType: "named",
					Source:     "./other", // re-export
					Name:       "foo",
					Specifiers: []domain.ExportSpecifier{
						{Local: "foo", Exported: "foo"},
					},
					Location: domain.SourceLocation{StartLine: 1, EndLine: 1},
				},
			},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/utils.js": true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for re-export, got %d", len(findings))
	}
}

func TestDetectUnusedExports_IndexFileSkipped(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/index.js": {
			FilePath: "/src/index.js",
			Exports: []*domain.Export{
				{
					ExportType: "declaration",
					Name:       "App",
					Location:   domain.SourceLocation{StartLine: 1, EndLine: 1},
				},
			},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/index.js": true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for index file exports, got %d", len(findings))
	}
}

func TestDetectUnusedExports_TestFileSkipped(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/utils.test.js": {
			FilePath: "/src/utils.test.js",
			Exports: []*domain.Export{
				{
					ExportType: "declaration",
					Name:       "testHelper",
					Location:   domain.SourceLocation{StartLine: 1, EndLine: 1},
				},
			},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/utils.test.js": true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for test file exports, got %d", len(findings))
	}
}

func TestDetectUnusedExports_SpecFileSkipped(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/utils.spec.ts": {
			FilePath: "/src/utils.spec.ts",
			Exports: []*domain.Export{
				{
					ExportType: "declaration",
					Name:       "testHelper",
					Location:   domain.SourceLocation{StartLine: 1, EndLine: 1},
				},
			},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/utils.spec.ts": true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for spec file exports, got %d", len(findings))
	}
}

func TestDetectUnusedExports_NilInput(t *testing.T) {
	findings := DetectUnusedExports(nil, nil)
	if findings != nil {
		t.Errorf("Expected nil findings for nil input, got %d", len(findings))
	}
}

// --- Path Resolution Tests ---

func TestResolveImportPath_BasicResolution(t *testing.T) {
	knownFiles := map[string]bool{
		"/src/utils.js": true,
		"/src/app.js":   true,
	}

	resolved := resolveImportPath("/src/app.js", "./utils", knownFiles)
	if resolved != "/src/utils.js" {
		t.Errorf("Expected /src/utils.js, got %s", resolved)
	}
}

func TestResolveImportPath_WithExtension(t *testing.T) {
	knownFiles := map[string]bool{
		"/src/utils.ts": true,
	}

	resolved := resolveImportPath("/src/app.ts", "./utils", knownFiles)
	if resolved != "/src/utils.ts" {
		t.Errorf("Expected /src/utils.ts, got %s", resolved)
	}
}

func TestResolveImportPath_IndexFile(t *testing.T) {
	knownFiles := map[string]bool{
		"/src/components/index.ts": true,
	}

	resolved := resolveImportPath("/src/app.ts", "./components", knownFiles)
	if resolved != "/src/components/index.ts" {
		t.Errorf("Expected /src/components/index.ts, got %s", resolved)
	}
}

func TestResolveImportPath_ParentDirectory(t *testing.T) {
	knownFiles := map[string]bool{
		"/src/utils.js": true,
	}

	resolved := resolveImportPath("/src/sub/app.js", "../utils", knownFiles)
	if resolved != "/src/utils.js" {
		t.Errorf("Expected /src/utils.js, got %s", resolved)
	}
}

func TestResolveImportPath_NonRelative(t *testing.T) {
	knownFiles := map[string]bool{
		"/node_modules/react/index.js": true,
	}

	resolved := resolveImportPath("/src/app.js", "react", knownFiles)
	if resolved != "" {
		t.Errorf("Expected empty string for non-relative import, got %s", resolved)
	}
}

func TestResolveImportPath_NotFound(t *testing.T) {
	knownFiles := map[string]bool{
		"/src/app.js": true,
	}

	resolved := resolveImportPath("/src/app.js", "./nonexistent", knownFiles)
	if resolved != "" {
		t.Errorf("Expected empty string for unresolved path, got %s", resolved)
	}
}

// --- Entry Point / Test File Helpers ---

func TestIsEntryPointFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/src/index.ts", true},
		{"/src/index.js", true},
		{"/src/index.tsx", true},
		{"/src/utils.ts", false},
		{"/src/app.js", false},
	}

	for _, tc := range tests {
		result := isEntryPointFile(tc.path)
		if result != tc.expected {
			t.Errorf("isEntryPointFile(%q) = %v, want %v", tc.path, result, tc.expected)
		}
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/src/utils.test.ts", true},
		{"/src/utils.spec.js", true},
		{"/src/__tests__/utils.js", true},
		{"/src/utils.ts", false},
		{"/src/app.js", false},
	}

	for _, tc := range tests {
		result := isTestFile(tc.path)
		if result != tc.expected {
			t.Errorf("isTestFile(%q) = %v, want %v", tc.path, result, tc.expected)
		}
	}
}

// --- Integration-style Test ---

func TestDetectUnusedExports_DefaultExport(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/component.js": {
			FilePath: "/src/component.js",
			Exports: []*domain.Export{
				{
					ExportType: "default",
					Name:       "MyComponent",
					Location:   domain.SourceLocation{StartLine: 10, EndLine: 10},
				},
			},
		},
		"/src/app.js": {
			FilePath: "/src/app.js",
			Imports: []*domain.Import{
				{
					Source:     "./component",
					SourceType: domain.ModuleTypeRelative,
					ImportType: domain.ImportTypeDefault,
					Specifiers: []domain.ImportSpecifier{
						{Imported: "default", Local: "MyComponent"},
					},
				},
			},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/component.js": true,
		"/src/app.js":       true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings when default export is imported, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s", f.Description)
		}
	}
}

func TestDetectUnusedExports_NamespaceImportCoversAll(t *testing.T) {
	allInfos := map[string]*domain.ModuleInfo{
		"/src/utils.js": {
			FilePath: "/src/utils.js",
			Exports: []*domain.Export{
				{
					ExportType: "declaration",
					Name:       "foo",
					Location:   domain.SourceLocation{StartLine: 1, EndLine: 1},
				},
				{
					ExportType: "declaration",
					Name:       "bar",
					Location:   domain.SourceLocation{StartLine: 2, EndLine: 2},
				},
			},
		},
		"/src/app.js": {
			FilePath: "/src/app.js",
			Imports: []*domain.Import{
				{
					Source:     "./utils",
					SourceType: domain.ModuleTypeRelative,
					ImportType: domain.ImportTypeNamespace,
					Specifiers: []domain.ImportSpecifier{
						{Imported: "*", Local: "utils"},
					},
				},
			},
		},
	}
	analyzedFiles := map[string]bool{
		"/src/utils.js": true,
		"/src/app.js":   true,
	}

	findings := DetectUnusedExports(allInfos, analyzedFiles)

	if len(findings) != 0 {
		t.Errorf("Expected 0 findings when namespace import covers all exports, got %d", len(findings))
	}
}
