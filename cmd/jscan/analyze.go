package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/analyzer"
	"github.com/ludo-technologies/jscan/internal/config"
	"github.com/ludo-technologies/jscan/internal/parser"
	"github.com/ludo-technologies/jscan/internal/version"
	"github.com/ludo-technologies/jscan/service"
	"github.com/spf13/cobra"
)

var (
	selectAnalyses []string
	outputFormat   string
	configPath     string
	jsonOutput     bool
	htmlOutput     bool
	noOpenBrowser  bool
	outputPath     string
)

func analyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [path...]",
		Short: "Analyze JavaScript/TypeScript files",
		Long: `Analyze JavaScript/TypeScript files for complexity, dead code, and other issues.

Examples:
  jscan analyze src/
  jscan analyze --select complexity src/
  jscan analyze --select complexity,deadcode --json src/
  jscan analyze --format json src/`,
		RunE: runAnalyze,
	}

	cmd.Flags().StringSliceVarP(&selectAnalyses, "select", "s", []string{"complexity", "deadcode"},
		"Analyses to run (comma-separated): complexity,deadcode")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text",
		"Output format: text, json, html")
	cmd.Flags().BoolVar(&jsonOutput, "json", false,
		"Output results as JSON (shorthand for --format json)")
	cmd.Flags().BoolVar(&htmlOutput, "html", false,
		"Output results as HTML (shorthand for --format html)")
	cmd.Flags().BoolVar(&noOpenBrowser, "no-open", false,
		"Don't auto-open HTML report in browser")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "",
		"Output file path (default: jscan-report.html for HTML)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "",
		"Path to config file")

	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no paths specified")
	}

	// Determine output format
	format := domain.OutputFormatText
	if jsonOutput || outputFormat == "json" {
		format = domain.OutputFormatJSON
	} else if htmlOutput || outputFormat == "html" {
		format = domain.OutputFormatHTML
	}

	// Load configuration
	cfg := config.DefaultConfig()
	if configPath != "" {
		// TODO: Load custom config
		if format != domain.OutputFormatJSON {
			fmt.Printf("Using config: %s\n", configPath)
		}
	}

	// Collect JavaScript/TypeScript files
	var files []string
	for _, path := range args {
		pathFiles, err := collectJSFiles(path)
		if err != nil {
			return fmt.Errorf("failed to collect files from %s: %w", path, err)
		}
		files = append(files, pathFiles...)
	}

	if len(files) == 0 {
		return fmt.Errorf("no JavaScript/TypeScript files found")
	}

	if format != domain.OutputFormatJSON {
		fmt.Printf("Analyzing %d files...\n", len(files))
	}

	// Create progress manager (auto-disabled for JSON output or non-TTY)
	pm := service.NewProgressManager(format != domain.OutputFormatJSON)
	defer pm.Close()

	// Start timing
	startTime := time.Now()

	// Initialize responses
	var complexityResponse *domain.ComplexityResponse
	var deadCodeResponse *domain.DeadCodeResponse

	// Run complexity analysis if selected
	if contains(selectAnalyses, "complexity") {
		resp, err := runComplexityAnalysis(files, cfg, pm)
		if err != nil {
			if format != domain.OutputFormatJSON {
				fmt.Fprintf(os.Stderr, "Complexity analysis error: %v\n", err)
			}
		} else {
			complexityResponse = resp
		}
	}

	// Run dead code analysis if selected
	if contains(selectAnalyses, "deadcode") {
		resp, err := runDeadCodeAnalysis(files, cfg, pm)
		if err != nil {
			if format != domain.OutputFormatJSON {
				fmt.Fprintf(os.Stderr, "Dead code analysis error: %v\n", err)
			}
		} else {
			deadCodeResponse = resp
		}
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Output results
	formatter := service.NewOutputFormatter()

	// Handle HTML output with file writing and browser opening
	if format == domain.OutputFormatHTML {
		// Determine output path
		htmlPath := outputPath
		if htmlPath == "" {
			htmlPath = "jscan-report.html"
		}

		// Create HTML file
		file, err := os.Create(htmlPath)
		if err != nil {
			return fmt.Errorf("failed to create HTML file: %w", err)
		}
		defer file.Close()

		// Write HTML
		if err := formatter.WriteAnalyze(complexityResponse, deadCodeResponse, format, file, duration); err != nil {
			return err
		}

		// Get absolute path for display
		absPath, _ := filepath.Abs(htmlPath)
		fmt.Printf("HTML report saved to: %s\n", absPath)

		// Open in browser unless disabled
		if !noOpenBrowser && !service.IsSSH() {
			if err := service.OpenBrowser("file://" + absPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not open browser: %v\n", err)
			}
		}

		return nil
	}

	// JSON or Text output to stdout
	return formatter.WriteAnalyze(complexityResponse, deadCodeResponse, format, os.Stdout, duration)
}

// runComplexityAnalysis runs complexity analysis on the given files
func runComplexityAnalysis(files []string, cfg *config.Config, pm domain.ProgressManager) (*domain.ComplexityResponse, error) {
	svc := service.NewComplexityServiceWithProgress(&cfg.Complexity, pm)

	req := domain.ComplexityRequest{
		Paths:           files,
		LowThreshold:    cfg.Complexity.LowThreshold,
		MediumThreshold: cfg.Complexity.MediumThreshold,
		SortBy:          domain.SortByComplexity,
	}

	ctx := context.Background()
	return svc.Analyze(ctx, req)
}

// runDeadCodeAnalysis runs dead code analysis on the given files
func runDeadCodeAnalysis(files []string, _ *config.Config, pm domain.ProgressManager) (*domain.DeadCodeResponse, error) {
	var allFiles []domain.FileDeadCode
	var totalFindings, criticalFindings, warningFindings, infoFindings int
	var totalFunctions, functionsWithDeadCode int

	// Set up progress tracking
	var task domain.TaskProgress
	if pm != nil {
		task = pm.StartTask("Detecting dead code", len(files))
		defer task.Complete()
	}

	for _, filePath := range files {
		// Read file
		content, err := os.ReadFile(filePath)
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
		var functions []domain.FunctionDeadCode
		for funcName, result := range results {
			if funcName == "__main__" {
				continue
			}

			totalFunctions++

			var findings []domain.DeadCodeFinding
			for _, finding := range result.Findings {
				f := domain.DeadCodeFinding{
					Location: domain.DeadCodeLocation{
						FilePath:  filePath,
						StartLine: finding.StartLine,
						EndLine:   finding.EndLine,
					},
					FunctionName: funcName,
					Reason:       string(finding.Reason),
					Severity:     domain.DeadCodeSeverity(finding.Severity),
					Description:  finding.Description,
				}
				findings = append(findings, f)

				// Count by severity
				switch f.Severity {
				case domain.DeadCodeSeverityCritical:
					criticalFindings++
				case domain.DeadCodeSeverityWarning:
					warningFindings++
				case domain.DeadCodeSeverityInfo:
					infoFindings++
				}
				totalFindings++
			}

			if len(findings) > 0 {
				functionsWithDeadCode++
				fn := domain.FunctionDeadCode{
					Name:     funcName,
					FilePath: filePath,
					Findings: findings,
				}
				fn.CalculateSeverityCounts()
				functions = append(functions, fn)
			}
		}

		if len(functions) > 0 {
			fileDeadCode := domain.FileDeadCode{
				FilePath:      filePath,
				Functions:     functions,
				TotalFindings: len(functions),
			}
			allFiles = append(allFiles, fileDeadCode)
		}

		if task != nil {
			task.Increment(1)
		}
	}

	response := &domain.DeadCodeResponse{
		Files: allFiles,
		Summary: domain.DeadCodeSummary{
			TotalFiles:            len(files),
			TotalFunctions:        totalFunctions,
			TotalFindings:         totalFindings,
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

func collectJSFiles(path string) ([]string, error) {
	var files []string

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		if isJSFile(path) {
			return []string{path}, nil
		}
		return nil, nil
	}

	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isJSFile(filePath) {
			files = append(files, filePath)
		}

		return nil
	})

	return files, err
}

func isJSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" ||
		ext == ".mjs" || ext == ".cjs" || ext == ".mts" || ext == ".cts"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
