package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ludo-technologies/jscan/app"
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
	textOutput     bool
	noOpenBrowser  bool
	outputPath     string
)

func analyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [path...]",
		Short: "Analyze JavaScript/TypeScript files",
		Long: `Analyze JavaScript/TypeScript files for complexity, dead code, code clones, and coupling.

By default, generates an HTML report and opens it in your browser.

Examples:
  jscan analyze src/                              # All analyses (default)
  jscan analyze --select complexity,deadcode src/ # Complexity + dead code only
  jscan analyze --select clone src/               # Clone detection only
  jscan analyze --select cbo src/                 # CBO coupling analysis only
  jscan analyze --json src/                       # Output JSON to stdout
  jscan analyze --text src/                       # Output text to stdout
  jscan analyze --no-open src/                    # Generate HTML without opening browser
  jscan analyze -o report.html src/               # Custom output path`,
		RunE: runAnalyze,
	}

	cmd.Flags().StringSliceVarP(&selectAnalyses, "select", "s", []string{"complexity", "deadcode", "clone", "cbo", "deps"},
		"Analyses to run (comma-separated): complexity,deadcode,clone,cbo,deps")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "html",
		"Output format: html, json, text (default: html)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false,
		"Output results as JSON to stdout")
	cmd.Flags().BoolVar(&textOutput, "text", false,
		"Output results as text to stdout")
	cmd.Flags().BoolVar(&htmlOutput, "html", false,
		"Output results as HTML report (default)")
	cmd.Flags().BoolVar(&noOpenBrowser, "no-open", false,
		"Don't auto-open HTML report in browser")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "",
		"Output file path (default: jscan-report.html)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "",
		"Path to config file")

	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no paths specified")
	}

	// Determine output format (default: HTML)
	format := domain.OutputFormatHTML
	if jsonOutput || outputFormat == "json" {
		format = domain.OutputFormatJSON
	} else if textOutput || outputFormat == "text" {
		format = domain.OutputFormatText
	}

	// Load configuration
	cfg := config.DefaultConfig()
	if configPath != "" {
		// TODO: Load custom config
		if format != domain.OutputFormatJSON {
			fmt.Printf("Using config: %s\n", configPath)
		}
	}

	// Collect JavaScript/TypeScript files (using exclude patterns from config)
	var files []string
	for _, path := range args {
		pathFiles, err := collectJSFiles(path, cfg.Analysis.ExcludePatterns)
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
	var cloneResponse *domain.CloneResponse
	var cboResponse *domain.CBOResponse
	var depsResponse *domain.DependencyGraphResponse

	// Determine which analyses to run
	runComplexity := contains(selectAnalyses, "complexity")
	runDeadCode := contains(selectAnalyses, "deadcode")
	runClone := contains(selectAnalyses, "clone")
	runCBO := contains(selectAnalyses, "cbo")
	runDeps := contains(selectAnalyses, "deps")

	// Single progress bar for all analyses (only when interactive)
	var task domain.TaskProgress
	var progressDone chan struct{}
	if pm.IsInteractive() {
		task = pm.StartTask("Analyzing", 100)
		estimatedDuration := estimateAnalysisDuration(len(files), runComplexity, runDeadCode)
		progressDone = startTimeBasedProgressUpdater(task, estimatedDuration)
	}

	// Run analyses in parallel
	var wg sync.WaitGroup
	var complexityErr, deadCodeErr, cloneErr, cboErr, depsErr error
	var mu sync.Mutex
	ctx := context.Background()

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
			resp, err := runDeadCodeAnalysisInternal(files)
			mu.Lock()
			deadCodeResponse = resp
			deadCodeErr = err
			mu.Unlock()
		}()
	}

	if runClone {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runCloneAnalysisInternal(ctx, files)
			mu.Lock()
			cloneResponse = resp
			cloneErr = err
			mu.Unlock()
		}()
	}

	if runCBO {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runCBOAnalysisInternal(ctx, files)
			mu.Lock()
			cboResponse = resp
			cboErr = err
			mu.Unlock()
		}()
	}

	if runDeps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runDepsAnalysisInternal(ctx, files)
			mu.Lock()
			depsResponse = resp
			depsErr = err
			mu.Unlock()
		}()
	}

	wg.Wait()
	if progressDone != nil {
		close(progressDone)
	}
	if task != nil {
		task.Complete()
	}

	// Handle errors
	if complexityErr != nil && format != domain.OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "Complexity analysis error: %v\n", complexityErr)
	}
	if deadCodeErr != nil && format != domain.OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "Dead code analysis error: %v\n", deadCodeErr)
	}
	if cloneErr != nil && format != domain.OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "Clone analysis error: %v\n", cloneErr)
	}
	if cboErr != nil && format != domain.OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "CBO analysis error: %v\n", cboErr)
	}
	if depsErr != nil && format != domain.OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "Dependency analysis error: %v\n", depsErr)
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
		if err := formatter.WriteAnalyze(complexityResponse, deadCodeResponse, cloneResponse, cboResponse, depsResponse, format, file, duration); err != nil {
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
	return formatter.WriteAnalyze(complexityResponse, deadCodeResponse, cloneResponse, cboResponse, depsResponse, format, os.Stdout, duration)
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
func runDeadCodeAnalysis(files []string, _ *config.Config, pm domain.ProgressManager) (*domain.DeadCodeResponse, error) {
	task := pm.StartTask("Detecting dead code", len(files))
	defer task.Complete()

	return runDeadCodeAnalysisWithTask(files, task)
}

// runDeadCodeAnalysisInternal runs dead code analysis on the given files without progress tracking
func runDeadCodeAnalysisInternal(files []string) (*domain.DeadCodeResponse, error) {
	return runDeadCodeAnalysisWithTask(files, nil)
}

// runDeadCodeAnalysisWithTask runs dead code analysis with optional task progress
func runDeadCodeAnalysisWithTask(files []string, task domain.TaskProgress) (*domain.DeadCodeResponse, error) {
	var allFiles []domain.FileDeadCode
	var totalFindings, criticalFindings, warningFindings, infoFindings int
	var totalFunctions, functionsWithDeadCode int

	// Track module info and ASTs for cross-file unused export detection
	allModuleInfos := make(map[string]*domain.ModuleInfo)
	analyzedFiles := make(map[string]bool)
	// Map filePath → index in allFiles for merging file-level findings later
	fileIndexMap := make(map[string]int)

	moduleAnalyzer := analyzer.NewModuleAnalyzer(nil)

	for _, filePath := range files {
		analyzedFiles[filePath] = true

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

		// Detect dead code (CFG-based)
		results := analyzer.DetectAll(cfgs, filePath)

		// Analyze module imports/exports
		moduleInfo, moduleErr := moduleAnalyzer.AnalyzeFile(ast, filePath)
		if moduleErr == nil && moduleInfo != nil {
			allModuleInfos[filePath] = moduleInfo
		}

		// Detect unused imports (per-file)
		var fileLevelFindings []domain.DeadCodeFinding
		if moduleInfo != nil {
			unusedImports := analyzer.DetectUnusedImports(ast, moduleInfo, filePath)
			for _, finding := range unusedImports {
				f := domain.DeadCodeFinding{
					Location: domain.DeadCodeLocation{
						FilePath:  filePath,
						StartLine: finding.StartLine,
						EndLine:   finding.EndLine,
					},
					Reason:      string(finding.Reason),
					Severity:    domain.DeadCodeSeverity(finding.Severity),
					Description: finding.Description,
				}
				fileLevelFindings = append(fileLevelFindings, f)
				warningFindings++
				totalFindings++
			}
		}

		// Convert CFG-based results to domain model
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

		if len(functions) > 0 || len(fileLevelFindings) > 0 {
			findingsCount := len(fileLevelFindings)
			for _, fn := range functions {
				findingsCount += len(fn.Findings)
			}
			fileDeadCode := domain.FileDeadCode{
				FilePath:          filePath,
				Functions:         functions,
				FileLevelFindings: fileLevelFindings,
				TotalFindings:     findingsCount,
			}
			fileIndexMap[filePath] = len(allFiles)
			allFiles = append(allFiles, fileDeadCode)
		}

		if task != nil {
			task.Increment(1)
		}
	}

	// Post-step: Detect unused exports (cross-file)
	unusedExports := analyzer.DetectUnusedExports(allModuleInfos, analyzedFiles)
	for _, finding := range unusedExports {
		f := domain.DeadCodeFinding{
			Location: domain.DeadCodeLocation{
				FilePath:  finding.FilePath,
				StartLine: finding.StartLine,
				EndLine:   finding.EndLine,
			},
			Reason:      string(finding.Reason),
			Severity:    domain.DeadCodeSeverity(finding.Severity),
			Description: finding.Description,
		}

		// Merge into existing file entry or create new one
		if idx, ok := fileIndexMap[finding.FilePath]; ok {
			allFiles[idx].FileLevelFindings = append(allFiles[idx].FileLevelFindings, f)
			allFiles[idx].TotalFindings++
		} else {
			fileDeadCode := domain.FileDeadCode{
				FilePath:          finding.FilePath,
				FileLevelFindings: []domain.DeadCodeFinding{f},
				TotalFindings:     1,
			}
			fileIndexMap[finding.FilePath] = len(allFiles)
			allFiles = append(allFiles, fileDeadCode)
		}

		infoFindings++
		totalFindings++
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

// collectJSFiles collects JavaScript/TypeScript files from a path using FileHelper
func collectJSFiles(path string, excludePatterns []string) ([]string, error) {
	helper := app.NewFileHelper()
	return helper.CollectJSFiles([]string{path}, true, nil, excludePatterns)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// estimateAnalysisDuration estimates total analysis time based on file count.
// Since analyses run in parallel, the time is based on the slower analysis (not the sum).
func estimateAnalysisDuration(fileCount int, _, _ bool) time.Duration {
	perFileMs := 10.0

	estimatedMs := float64(fileCount) * perFileMs
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

// runCloneAnalysisInternal runs clone detection without progress tracking
func runCloneAnalysisInternal(ctx context.Context, files []string) (*domain.CloneResponse, error) {
	svc := service.NewCloneServiceWithDefaults()

	req := domain.DefaultCloneRequest()
	req.Paths = files

	return svc.DetectClones(ctx, req)
}

// runCBOAnalysisInternal runs CBO analysis without progress tracking
func runCBOAnalysisInternal(ctx context.Context, files []string) (*domain.CBOResponse, error) {
	svc := service.NewCBOServiceWithDefaults()

	req := domain.CBORequest{
		Paths: files,
	}

	return svc.Analyze(ctx, req)
}

// runDepsAnalysisInternal runs dependency analysis without progress tracking
func runDepsAnalysisInternal(ctx context.Context, files []string) (*domain.DependencyGraphResponse, error) {
	svc := service.NewDependencyGraphServiceWithDefaults()

	req := domain.DependencyGraphRequest{
		Paths:        files,
		DetectCycles: domain.BoolPtr(true),
	}

	return svc.Analyze(ctx, req)
}
