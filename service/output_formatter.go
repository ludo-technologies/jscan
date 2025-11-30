package service

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/version"
)

// OutputFormatterImpl implements the OutputFormatter interface
type OutputFormatterImpl struct{}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatterImpl {
	return &OutputFormatterImpl{}
}

// FormatUtils provides formatting helper functions
type FormatUtils struct{}

// NewFormatUtils creates a new FormatUtils instance
func NewFormatUtils() *FormatUtils {
	return &FormatUtils{}
}

// WriteJSON writes data as JSON to the writer
func WriteJSON(writer io.Writer, data interface{}) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// ComplexityResponseJSON wraps ComplexityResponse with JSON metadata
type ComplexityResponseJSON struct {
	Version     string                     `json:"version"`
	GeneratedAt string                     `json:"generated_at"`
	DurationMs  int64                      `json:"duration_ms,omitempty"`
	Functions   []domain.FunctionComplexity `json:"functions"`
	Summary     domain.ComplexitySummary   `json:"summary"`
	Warnings    []string                   `json:"warnings,omitempty"`
	Errors      []string                   `json:"errors,omitempty"`
	Config      interface{}                `json:"config,omitempty"`
}

// DeadCodeResponseJSON wraps DeadCodeResponse with JSON metadata
type DeadCodeResponseJSON struct {
	Version     string                  `json:"version"`
	GeneratedAt string                  `json:"generated_at"`
	DurationMs  int64                   `json:"duration_ms,omitempty"`
	Files       []domain.FileDeadCode   `json:"files"`
	Summary     domain.DeadCodeSummary  `json:"summary"`
	Warnings    []string                `json:"warnings,omitempty"`
	Errors      []string                `json:"errors,omitempty"`
	Config      interface{}             `json:"config,omitempty"`
}

// AnalyzeResponseJSON represents the unified analysis response for JSON output
type AnalyzeResponseJSON struct {
	Version     string                      `json:"version"`
	GeneratedAt string                      `json:"generated_at"`
	DurationMs  int64                       `json:"duration_ms"`
	Complexity  *ComplexityResponseJSON     `json:"complexity,omitempty"`
	DeadCode    *DeadCodeResponseJSON       `json:"dead_code,omitempty"`
	Summary     *domain.AnalyzeSummary      `json:"summary,omitempty"`
}

// Write writes the complexity response in the specified format
func (f *OutputFormatterImpl) Write(response *domain.ComplexityResponse, format domain.OutputFormat, writer io.Writer) error {
	switch format {
	case domain.OutputFormatJSON:
		return f.writeComplexityJSON(response, writer)
	case domain.OutputFormatText:
		return f.writeComplexityText(response, writer)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteDeadCode writes the dead code response in the specified format
func (f *OutputFormatterImpl) WriteDeadCode(response *domain.DeadCodeResponse, format domain.OutputFormat, writer io.Writer) error {
	switch format {
	case domain.OutputFormatJSON:
		return f.writeDeadCodeJSON(response, writer)
	case domain.OutputFormatText:
		return f.writeDeadCodeText(response, writer)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteAnalyze writes the unified analysis response in the specified format
func (f *OutputFormatterImpl) WriteAnalyze(
	complexityResponse *domain.ComplexityResponse,
	deadCodeResponse *domain.DeadCodeResponse,
	format domain.OutputFormat,
	writer io.Writer,
	duration time.Duration,
) error {
	switch format {
	case domain.OutputFormatJSON:
		return f.writeAnalyzeJSON(complexityResponse, deadCodeResponse, writer, duration)
	case domain.OutputFormatText:
		return f.writeAnalyzeText(complexityResponse, deadCodeResponse, writer, duration)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// writeComplexityJSON writes complexity response as JSON
func (f *OutputFormatterImpl) writeComplexityJSON(response *domain.ComplexityResponse, writer io.Writer) error {
	jsonResponse := ComplexityResponseJSON{
		Version:     version.Version,
		GeneratedAt: response.GeneratedAt,
		Functions:   response.Functions,
		Summary:     response.Summary,
		Warnings:    response.Warnings,
		Errors:      response.Errors,
		Config:      response.Config,
	}
	return WriteJSON(writer, jsonResponse)
}

// writeDeadCodeJSON writes dead code response as JSON
func (f *OutputFormatterImpl) writeDeadCodeJSON(response *domain.DeadCodeResponse, writer io.Writer) error {
	jsonResponse := DeadCodeResponseJSON{
		Version:     version.Version,
		GeneratedAt: response.GeneratedAt,
		Files:       response.Files,
		Summary:     response.Summary,
		Warnings:    response.Warnings,
		Errors:      response.Errors,
		Config:      response.Config,
	}
	return WriteJSON(writer, jsonResponse)
}

// writeAnalyzeJSON writes unified analysis response as JSON
func (f *OutputFormatterImpl) writeAnalyzeJSON(
	complexityResponse *domain.ComplexityResponse,
	deadCodeResponse *domain.DeadCodeResponse,
	writer io.Writer,
	duration time.Duration,
) error {
	now := time.Now()

	response := AnalyzeResponseJSON{
		Version:     version.Version,
		GeneratedAt: now.Format(time.RFC3339),
		DurationMs:  duration.Milliseconds(),
	}

	// Build summary
	summary := &domain.AnalyzeSummary{}

	// Add complexity data if available
	if complexityResponse != nil {
		response.Complexity = &ComplexityResponseJSON{
			Version:     version.Version,
			GeneratedAt: complexityResponse.GeneratedAt,
			Functions:   complexityResponse.Functions,
			Summary:     complexityResponse.Summary,
			Warnings:    complexityResponse.Warnings,
			Errors:      complexityResponse.Errors,
			Config:      complexityResponse.Config,
		}
		summary.ComplexityEnabled = true
		summary.TotalFunctions = complexityResponse.Summary.TotalFunctions
		summary.AverageComplexity = complexityResponse.Summary.AverageComplexity
		summary.HighComplexityCount = complexityResponse.Summary.HighRiskFunctions
		summary.AnalyzedFiles = complexityResponse.Summary.FilesAnalyzed
	}

	// Add dead code data if available
	if deadCodeResponse != nil {
		response.DeadCode = &DeadCodeResponseJSON{
			Version:     version.Version,
			GeneratedAt: deadCodeResponse.GeneratedAt,
			Files:       deadCodeResponse.Files,
			Summary:     deadCodeResponse.Summary,
			Warnings:    deadCodeResponse.Warnings,
			Errors:      deadCodeResponse.Errors,
			Config:      deadCodeResponse.Config,
		}
		summary.DeadCodeEnabled = true
		summary.DeadCodeCount = deadCodeResponse.Summary.TotalFindings
		summary.CriticalDeadCode = deadCodeResponse.Summary.CriticalFindings
		summary.WarningDeadCode = deadCodeResponse.Summary.WarningFindings
		summary.InfoDeadCode = deadCodeResponse.Summary.InfoFindings
		if deadCodeResponse.Summary.TotalFiles > summary.TotalFiles {
			summary.TotalFiles = deadCodeResponse.Summary.TotalFiles
		}
	}

	// Calculate health score
	_ = summary.CalculateHealthScore()
	response.Summary = summary

	return WriteJSON(writer, response)
}

// writeComplexityText writes complexity response as plain text
func (f *OutputFormatterImpl) writeComplexityText(response *domain.ComplexityResponse, writer io.Writer) error {
	fmt.Fprintf(writer, "\n=== Complexity Analysis ===\n\n")
	fmt.Fprintf(writer, "Generated: %s\n", response.GeneratedAt)
	fmt.Fprintf(writer, "Version: %s\n\n", response.Version)

	// Summary
	fmt.Fprintf(writer, "Summary:\n")
	fmt.Fprintf(writer, "  Files analyzed: %d\n", response.Summary.FilesAnalyzed)
	fmt.Fprintf(writer, "  Total functions: %d\n", response.Summary.TotalFunctions)
	fmt.Fprintf(writer, "  Average complexity: %.2f\n", response.Summary.AverageComplexity)
	fmt.Fprintf(writer, "  Max complexity: %d\n", response.Summary.MaxComplexity)
	fmt.Fprintf(writer, "  Min complexity: %d\n", response.Summary.MinComplexity)
	fmt.Fprintf(writer, "\n")

	// Risk distribution
	fmt.Fprintf(writer, "Risk Distribution:\n")
	fmt.Fprintf(writer, "  High risk: %d\n", response.Summary.HighRiskFunctions)
	fmt.Fprintf(writer, "  Medium risk: %d\n", response.Summary.MediumRiskFunctions)
	fmt.Fprintf(writer, "  Low risk: %d\n", response.Summary.LowRiskFunctions)
	fmt.Fprintf(writer, "\n")

	// Function details
	if len(response.Functions) > 0 {
		fmt.Fprintf(writer, "Functions (sorted by complexity):\n")
		for _, fn := range response.Functions {
			riskIndicator := ""
			switch fn.RiskLevel {
			case domain.RiskLevelHigh:
				riskIndicator = " [HIGH]"
			case domain.RiskLevelMedium:
				riskIndicator = " [MEDIUM]"
			}
			fmt.Fprintf(writer, "  %s: %d%s\n", fn.Name, fn.Metrics.Complexity, riskIndicator)
			fmt.Fprintf(writer, "    File: %s:%d-%d\n", fn.FilePath, fn.StartLine, fn.EndLine)
		}
	}

	// Warnings
	if len(response.Warnings) > 0 {
		fmt.Fprintf(writer, "\nWarnings:\n")
		for _, w := range response.Warnings {
			fmt.Fprintf(writer, "  - %s\n", w)
		}
	}

	// Errors
	if len(response.Errors) > 0 {
		fmt.Fprintf(writer, "\nErrors:\n")
		for _, e := range response.Errors {
			fmt.Fprintf(writer, "  - %s\n", e)
		}
	}

	return nil
}

// writeDeadCodeText writes dead code response as plain text
func (f *OutputFormatterImpl) writeDeadCodeText(response *domain.DeadCodeResponse, writer io.Writer) error {
	fmt.Fprintf(writer, "\n=== Dead Code Analysis ===\n\n")
	fmt.Fprintf(writer, "Generated: %s\n", response.GeneratedAt)
	fmt.Fprintf(writer, "Version: %s\n\n", response.Version)

	// Summary
	fmt.Fprintf(writer, "Summary:\n")
	fmt.Fprintf(writer, "  Total files: %d\n", response.Summary.TotalFiles)
	fmt.Fprintf(writer, "  Total functions: %d\n", response.Summary.TotalFunctions)
	fmt.Fprintf(writer, "  Total findings: %d\n", response.Summary.TotalFindings)
	fmt.Fprintf(writer, "\n")

	// Severity distribution
	fmt.Fprintf(writer, "Severity Distribution:\n")
	fmt.Fprintf(writer, "  Critical: %d\n", response.Summary.CriticalFindings)
	fmt.Fprintf(writer, "  Warning: %d\n", response.Summary.WarningFindings)
	fmt.Fprintf(writer, "  Info: %d\n", response.Summary.InfoFindings)
	fmt.Fprintf(writer, "\n")

	// File details
	for _, file := range response.Files {
		if file.TotalFindings > 0 {
			fmt.Fprintf(writer, "%s:\n", file.FilePath)
			for _, fn := range file.Functions {
				if len(fn.Findings) > 0 {
					fmt.Fprintf(writer, "  %s:\n", fn.Name)
					for _, finding := range fn.Findings {
						severityIndicator := ""
						switch finding.Severity {
						case domain.DeadCodeSeverityCritical:
							severityIndicator = " [CRITICAL]"
						case domain.DeadCodeSeverityWarning:
							severityIndicator = " [WARNING]"
						case domain.DeadCodeSeverityInfo:
							severityIndicator = " [INFO]"
						}
						fmt.Fprintf(writer, "    Line %d-%d: %s%s\n",
							finding.Location.StartLine, finding.Location.EndLine,
							finding.Reason, severityIndicator)
						if finding.Description != "" {
							fmt.Fprintf(writer, "      %s\n", finding.Description)
						}
					}
				}
			}
		}
	}

	if response.Summary.TotalFindings == 0 {
		fmt.Fprintf(writer, "No dead code found.\n")
	}

	return nil
}

// writeAnalyzeText writes unified analysis response as plain text
func (f *OutputFormatterImpl) writeAnalyzeText(
	complexityResponse *domain.ComplexityResponse,
	deadCodeResponse *domain.DeadCodeResponse,
	writer io.Writer,
	duration time.Duration,
) error {
	fmt.Fprintf(writer, "\n=== jscan Analysis Report ===\n")
	fmt.Fprintf(writer, "Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(writer, "Duration: %dms\n", duration.Milliseconds())
	fmt.Fprintf(writer, "Version: %s\n\n", version.Version)

	// Complexity results
	if complexityResponse != nil {
		if err := f.writeComplexityText(complexityResponse, writer); err != nil {
			return err
		}
	}

	// Dead code results
	if deadCodeResponse != nil {
		if err := f.writeDeadCodeText(deadCodeResponse, writer); err != nil {
			return err
		}
	}

	return nil
}
