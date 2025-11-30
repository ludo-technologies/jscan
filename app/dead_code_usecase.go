package app

import (
	"context"
	"fmt"
	"time"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/analyzer"
	"github.com/ludo-technologies/jscan/internal/parser"
	"github.com/ludo-technologies/jscan/internal/version"
)

// DeadCodeUseCase orchestrates the dead code analysis workflow
type DeadCodeUseCase struct {
	fileHelper *FileHelper
}

// NewDeadCodeUseCase creates a new dead code use case
func NewDeadCodeUseCase() *DeadCodeUseCase {
	return &DeadCodeUseCase{
		fileHelper: NewFileHelper(),
	}
}

// Execute performs the complete dead code analysis workflow
func (uc *DeadCodeUseCase) Execute(ctx context.Context, req domain.DeadCodeRequest) (*domain.DeadCodeResponse, error) {
	// Validate input
	if err := req.Validate(); err != nil {
		return nil, domain.NewInvalidInputError("invalid request", err)
	}

	// Resolve file paths
	files, err := ResolveFilePaths(
		uc.fileHelper,
		req.Paths,
		req.Recursive,
		req.IncludePatterns,
		req.ExcludePatterns,
	)
	if err != nil {
		return nil, domain.NewFileNotFoundError("failed to collect files", err)
	}

	if len(files) == 0 {
		return nil, domain.NewInvalidInputError("no JavaScript/TypeScript files found in the specified paths", nil)
	}

	// Analyze files
	return uc.analyzeFiles(ctx, files, req)
}

// AnalyzeFile analyzes a single file for dead code
func (uc *DeadCodeUseCase) AnalyzeFile(ctx context.Context, filePath string, req domain.DeadCodeRequest) (*domain.FileDeadCode, error) {
	// Validate file
	if !uc.fileHelper.IsValidJSFile(filePath) {
		return nil, domain.NewInvalidInputError(fmt.Sprintf("not a valid JavaScript/TypeScript file: %s", filePath), nil)
	}

	// Check if file exists
	exists, err := uc.fileHelper.FileExists(filePath)
	if err != nil {
		return nil, domain.NewFileNotFoundError(filePath, err)
	}
	if !exists {
		return nil, domain.NewFileNotFoundError(filePath, fmt.Errorf("file does not exist"))
	}

	// Read and parse file
	content, err := uc.fileHelper.ReadFile(filePath)
	if err != nil {
		return nil, domain.NewFileNotFoundError(filePath, err)
	}

	ast, err := parser.ParseForLanguage(filePath, content)
	if err != nil {
		return nil, domain.NewAnalysisError("failed to parse file", err)
	}

	// Build CFGs
	builder := analyzer.NewCFGBuilder()
	cfgs, err := builder.BuildAll(ast)
	if err != nil {
		return nil, domain.NewAnalysisError("failed to build CFG", err)
	}

	// Detect dead code
	results := analyzer.DetectAll(cfgs, filePath)

	// Convert to domain model
	return uc.convertToFileDeadCode(filePath, results, req.MinSeverity), nil
}

// analyzeFiles analyzes multiple files for dead code
func (uc *DeadCodeUseCase) analyzeFiles(ctx context.Context, files []string, req domain.DeadCodeRequest) (*domain.DeadCodeResponse, error) {
	var allFiles []domain.FileDeadCode
	var totalFindings, criticalFindings, warningFindings, infoFindings int
	var totalFunctions, functionsWithDeadCode, filesWithDeadCode int

	for _, filePath := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Read file
		content, err := uc.fileHelper.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Parse file
		ast, err := parser.ParseForLanguage(filePath, content)
		if err != nil {
			continue
		}

		// Build CFGs for all functions
		builder := analyzer.NewCFGBuilder()
		cfgs, err := builder.BuildAll(ast)
		if err != nil {
			continue
		}

		// Detect dead code
		results := analyzer.DetectAll(cfgs, filePath)

		// Convert to domain model
		fileDeadCode := uc.convertToFileDeadCode(filePath, results, req.MinSeverity)
		if fileDeadCode != nil {
			// Update counts
			for _, fn := range fileDeadCode.Functions {
				totalFunctions++
				if len(fn.Findings) > 0 {
					functionsWithDeadCode++
					for _, finding := range fn.Findings {
						switch finding.Severity {
						case domain.DeadCodeSeverityCritical:
							criticalFindings++
						case domain.DeadCodeSeverityWarning:
							warningFindings++
						case domain.DeadCodeSeverityInfo:
							infoFindings++
						}
						totalFindings++
					}
				}
			}

			if fileDeadCode.TotalFindings > 0 {
				filesWithDeadCode++
				allFiles = append(allFiles, *fileDeadCode)
			}
		}
	}

	response := &domain.DeadCodeResponse{
		Files: allFiles,
		Summary: domain.DeadCodeSummary{
			TotalFiles:            len(files),
			TotalFunctions:        totalFunctions,
			TotalFindings:         totalFindings,
			FilesWithDeadCode:     filesWithDeadCode,
			FunctionsWithDeadCode: functionsWithDeadCode,
			CriticalFindings:      criticalFindings,
			WarningFindings:       warningFindings,
			InfoFindings:          infoFindings,
		},
		GeneratedAt: time.Now().Format(time.RFC3339),
		Version:     version.Version,
	}

	return response, nil
}

// convertToFileDeadCode converts analyzer results to domain model
func (uc *DeadCodeUseCase) convertToFileDeadCode(filePath string, results map[string]*analyzer.DeadCodeResult, minSeverity domain.DeadCodeSeverity) *domain.FileDeadCode {
	var functions []domain.FunctionDeadCode
	var totalFindings int

	for funcName, result := range results {
		if funcName == "__main__" {
			continue
		}

		var findings []domain.DeadCodeFinding
		for _, finding := range result.Findings {
			severity := domain.DeadCodeSeverity(finding.Severity)
			if !severity.IsAtLeast(minSeverity) {
				continue
			}

			f := domain.DeadCodeFinding{
				Location: domain.DeadCodeLocation{
					FilePath:  filePath,
					StartLine: finding.StartLine,
					EndLine:   finding.EndLine,
				},
				FunctionName: funcName,
				Reason:       string(finding.Reason),
				Severity:     severity,
				Description:  finding.Description,
			}
			findings = append(findings, f)
		}

		if len(findings) > 0 {
			fn := domain.FunctionDeadCode{
				Name:     funcName,
				FilePath: filePath,
				Findings: findings,
			}
			fn.CalculateSeverityCounts()
			functions = append(functions, fn)
			totalFindings += len(findings)
		}
	}

	if len(functions) == 0 {
		return nil
	}

	return &domain.FileDeadCode{
		FilePath:      filePath,
		Functions:     functions,
		TotalFindings: totalFindings,
	}
}

// DeadCodeUseCaseBuilder provides a builder pattern for creating DeadCodeUseCase
type DeadCodeUseCaseBuilder struct {
	fileHelper *FileHelper
}

// NewDeadCodeUseCaseBuilder creates a new builder
func NewDeadCodeUseCaseBuilder() *DeadCodeUseCaseBuilder {
	return &DeadCodeUseCaseBuilder{}
}

// WithFileHelper sets the file helper
func (b *DeadCodeUseCaseBuilder) WithFileHelper(fileHelper *FileHelper) *DeadCodeUseCaseBuilder {
	b.fileHelper = fileHelper
	return b
}

// Build creates the DeadCodeUseCase with the configured dependencies
func (b *DeadCodeUseCaseBuilder) Build() (*DeadCodeUseCase, error) {
	uc := &DeadCodeUseCase{
		fileHelper: b.fileHelper,
	}

	if uc.fileHelper == nil {
		uc.fileHelper = NewFileHelper()
	}

	return uc, nil
}
