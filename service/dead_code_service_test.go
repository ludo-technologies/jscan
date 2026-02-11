package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ludo-technologies/jscan/domain"
)

func TestDeadCodeServiceAnalyze(t *testing.T) {
	// Create a temp file with dead code
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.js")
	content := `
function hasDeadCode() {
    return 42;
    console.log("never executed");
}

function noDeadCode() {
    const x = 1;
    return x;
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	svc := NewDeadCodeService()
	ctx := context.Background()

	req := domain.DeadCodeRequest{
		Paths:       []string{testFile},
		MinSeverity: domain.DeadCodeSeverityInfo,
		SortBy:      domain.DeadCodeSortBySeverity,
	}

	response, err := svc.Analyze(ctx, req)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	// Verify summary was generated
	if response.Summary.TotalFiles != 1 {
		t.Errorf("Expected 1 file processed, got %d", response.Summary.TotalFiles)
	}
}

func TestDeadCodeServiceAnalyzeFile(t *testing.T) {
	// Create a temp file with dead code
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.js")
	content := `
function example() {
    if (true) {
        return 1;
    }
    return 2;
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	svc := NewDeadCodeService()
	ctx := context.Background()

	req := domain.DeadCodeRequest{
		MinSeverity: domain.DeadCodeSeverityInfo,
	}

	result, err := svc.AnalyzeFile(ctx, testFile, req)
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.FilePath != testFile {
		t.Errorf("Expected file path %s, got %s", testFile, result.FilePath)
	}
}

func TestDeadCodeServiceSorting(t *testing.T) {
	svc := NewDeadCodeService()

	files := []domain.FileDeadCode{
		{
			FilePath:      "b.js",
			TotalFindings: 5,
			Functions: []domain.FunctionDeadCode{
				{
					Name: "func1",
					Findings: []domain.DeadCodeFinding{
						{Severity: domain.DeadCodeSeverityWarning},
					},
				},
			},
		},
		{
			FilePath:      "a.js",
			TotalFindings: 10,
			Functions: []domain.FunctionDeadCode{
				{
					Name: "func2",
					Findings: []domain.DeadCodeFinding{
						{Severity: domain.DeadCodeSeverityCritical},
					},
				},
			},
		},
	}

	// Test sort by file
	sortedByFile := svc.sortFiles(files, domain.DeadCodeSortByFile)
	if sortedByFile[0].FilePath != "a.js" {
		t.Error("Expected files to be sorted by file path")
	}

	// Test sort by severity
	sortedBySeverity := svc.sortFiles(files, domain.DeadCodeSortBySeverity)
	if sortedBySeverity[0].FilePath != "a.js" {
		t.Error("Expected files to be sorted by severity (critical first)")
	}
}

func TestDeadCodeServiceFiltering(t *testing.T) {
	svc := NewDeadCodeService()

	files := []domain.FileDeadCode{
		{
			FilePath: "test.js",
			Functions: []domain.FunctionDeadCode{
				{
					Name: "func1",
					Findings: []domain.DeadCodeFinding{
						{Severity: domain.DeadCodeSeverityInfo},
						{Severity: domain.DeadCodeSeverityWarning},
					},
				},
				{
					Name: "func2",
					Findings: []domain.DeadCodeFinding{
						{Severity: domain.DeadCodeSeverityInfo},
					},
				},
			},
		},
	}

	// Filter with warning minimum severity
	req := domain.DeadCodeRequest{
		MinSeverity: domain.DeadCodeSeverityWarning,
	}

	filtered := svc.filterFiles(files, req)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 file after filtering, got %d", len(filtered))
	}

	// func2 should be filtered out (only has info severity)
	if len(filtered[0].Functions) != 1 {
		t.Errorf("Expected 1 function after filtering, got %d", len(filtered[0].Functions))
	}
	if filtered[0].Functions[0].Name != "func1" {
		t.Errorf("Expected func1 to remain, got %s", filtered[0].Functions[0].Name)
	}
}

func TestDeadCodeServiceSummaryGeneration(t *testing.T) {
	svc := NewDeadCodeService()

	files := []domain.FileDeadCode{
		{
			FilePath:          "test1.js",
			TotalFunctions:    3,
			AffectedFunctions: 2,
			TotalFindings:     5,
			Functions: []domain.FunctionDeadCode{
				{
					CriticalCount: 2,
					WarningCount:  2,
					InfoCount:     1,
					Findings: []domain.DeadCodeFinding{
						{Reason: "unreachable_after_return", Severity: domain.DeadCodeSeverityCritical},
						{Reason: "unreachable_after_return", Severity: domain.DeadCodeSeverityCritical},
						{Reason: "unreachable_branch", Severity: domain.DeadCodeSeverityWarning},
						{Reason: "unreachable_branch", Severity: domain.DeadCodeSeverityWarning},
						{Reason: "dead_assignment", Severity: domain.DeadCodeSeverityInfo},
					},
				},
			},
		},
	}

	req := domain.DeadCodeRequest{
		MinSeverity: domain.DeadCodeSeverityInfo,
	}

	summary := svc.generateSummary(files, 2, req)

	if summary.TotalFiles != 2 {
		t.Errorf("Expected TotalFiles to be 2, got %d", summary.TotalFiles)
	}
	if summary.FilesWithDeadCode != 1 {
		t.Errorf("Expected FilesWithDeadCode to be 1, got %d", summary.FilesWithDeadCode)
	}
	if summary.TotalFindings != 5 {
		t.Errorf("Expected TotalFindings to be 5, got %d", summary.TotalFindings)
	}
	if summary.CriticalFindings != 2 {
		t.Errorf("Expected CriticalFindings to be 2, got %d", summary.CriticalFindings)
	}
	if summary.WarningFindings != 2 {
		t.Errorf("Expected WarningFindings to be 2, got %d", summary.WarningFindings)
	}
	if summary.FindingsByReason["unreachable_after_return"] != 2 {
		t.Errorf("Expected 2 unreachable_after_return findings, got %d", summary.FindingsByReason["unreachable_after_return"])
	}
}

func TestDeadCodeServiceBuildConfig(t *testing.T) {
	svc := NewDeadCodeService()

	req := domain.DeadCodeRequest{
		MinSeverity:  domain.DeadCodeSeverityWarning,
		SortBy:       domain.DeadCodeSortByFile,
		ShowContext:  domain.BoolPtr(true),
		ContextLines: 5,
	}

	config := svc.buildConfigForResponse(req)

	if config["min_severity"] != domain.DeadCodeSeverityWarning {
		t.Errorf("Expected min_severity to be warning, got %v", config["min_severity"])
	}
	if config["sort_by"] != domain.DeadCodeSortByFile {
		t.Errorf("Expected sort_by to be file, got %v", config["sort_by"])
	}
	if config["show_context"] != true {
		t.Errorf("Expected show_context to be true, got %v", config["show_context"])
	}
	if config["context_lines"] != 5 {
		t.Errorf("Expected context_lines to be 5, got %v", config["context_lines"])
	}
}
