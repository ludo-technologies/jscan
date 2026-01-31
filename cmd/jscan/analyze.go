package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

	// Determine which analyses to run
	runComplexity := contains(selectAnalyses, "complexity")
	runDeadCode := contains(selectAnalyses, "deadcode")

	// Single progress bar for all analyses
	task := pm.StartTask("Analyzing", 100)
	estimatedDuration := estimateAnalysisDuration(len(files), runComplexity, runDeadCode)
	progressDone := startTimeBasedProgressUpdater(task, estimatedDuration)

	// Run analyses in parallel
	var wg sync.WaitGroup
	var complexityErr, deadCodeErr error
	var mu sync.Mutex

	if runComplexity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runComplexityAnalysisInternal(files, cfg)
			mu.Lock()
			complexityResponse = resp
			complexityErr = err
			mu.Unlock()
		}()
	}

	if runDeadCode {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runDeadCodeAnalysisInternal(files, cfg)
			mu.Lock()
			deadCodeResponse = resp
			deadCodeErr = err
			mu.Unlock()
		}()
	}

	wg.Wait()
	close(progressDone)
	task.Complete()

	// Handle errors
	if complexityErr != nil && format != domain.OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "Complexity analysis error: %v\n", complexityErr)
	}
	if deadCodeErr != nil && format != domain.OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "Dead code analysis error: %v\n", deadCodeErr)
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

// runComplexityAnalysisInternal runs complexity analysis on the given files without progress tracking
func runComplexityAnalysisInternal(files []string, cfg *config.Config) (*domain.ComplexityResponse, error) {
	svc := service.NewComplexityService(&cfg.Complexity)

	req := domain.ComplexityRequest{
		Paths:           files,
		LowThreshold:    cfg.Complexity.LowThreshold,
		MediumThreshold: cfg.Complexity.MediumThreshold,
		SortBy:          domain.SortByComplexity,
	}

	ctx := context.Background()
	return svc.Analyze(ctx, req)
}

// runDeadCodeAnalysis runs dead code analysis on the given files with progress tracking
// This is used by check.go which has its own progress management
func runDeadCodeAnalysis(files []string, cfg *config.Config, pm domain.ProgressManager) (*domain.DeadCodeResponse, error) {
	task := pm.StartTask("Detecting dead code", len(files))
	defer task.Complete()

	return runDeadCodeAnalysisWithTask(files, cfg, task)
}

// runDeadCodeAnalysisInternal runs dead code analysis on the given files without progress tracking
func runDeadCodeAnalysisInternal(files []string, cfg *config.Config) (*domain.DeadCodeResponse, error) {
	return runDeadCodeAnalysisWithTask(files, cfg, nil)
}

// runDeadCodeAnalysisWithTask runs dead code analysis with optional task progress
func runDeadCodeAnalysisWithTask(files []string, _ *config.Config, task domain.TaskProgress) (*domain.DeadCodeResponse, error) {
	var allFiles []domain.FileDeadCode
	var totalFindings, criticalFindings, warningFindings, infoFindings int
	var totalFunctions, functionsWithDeadCode int

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

// estimateAnalysisDuration estimates total analysis time based on file count
func estimateAnalysisDuration(fileCount int, runComplexity, runDeadCode bool) time.Duration {
	perFileMs := 10.0
	analysisCount := 0
	if runComplexity {
		analysisCount++
	}
	if runDeadCode {
		analysisCount++
	}

	estimatedMs := float64(fileCount) * perFileMs * float64(analysisCount)
	if estimatedMs < 100 {
		estimatedMs = 100
	}
	estimatedMs *= 1.5 // buffer

	return time.Duration(estimatedMs) * time.Millisecond
}

// startTimeBasedProgressUpdater starts background progress updates
func startTimeBasedProgressUpdater(task domain.TaskProgress, estimatedDuration time.Duration) chan struct{} {
	done := make(chan struct{})
	startTime := time.Now()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime)
				progress := int((float64(elapsed) / float64(estimatedDuration)) * 100)
				if progress > 99 {
					progress = 99
				}
				task.Describe(fmt.Sprintf("Analyzing... %d%%", progress))
			case <-done:
				return
			}
		}
	}()

	return done
}
