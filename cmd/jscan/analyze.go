package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ludo-technologies/jscan/internal/analyzer"
	"github.com/ludo-technologies/jscan/internal/config"
	"github.com/ludo-technologies/jscan/internal/parser"
	"github.com/spf13/cobra"
)

var (
	selectAnalyses []string
	outputFormat   string
	configPath     string
)

func analyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [path...]",
		Short: "Analyze JavaScript/TypeScript files",
		Long: `Analyze JavaScript/TypeScript files for complexity, dead code, and other issues.

Examples:
  jscan analyze src/
  jscan analyze --select complexity src/
  jscan analyze --select complexity,deadcode --format json src/`,
		RunE: runAnalyze,
	}

	cmd.Flags().StringSliceVarP(&selectAnalyses, "select", "s", []string{"complexity", "deadcode"},
		"Analyses to run (comma-separated): complexity,deadcode")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text",
		"Output format: text, json")
	cmd.Flags().StringVarP(&configPath, "config", "c", "",
		"Path to config file")

	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no paths specified")
	}

	// Load configuration
	cfg := config.DefaultConfig()
	if configPath != "" {
		// TODO: Load custom config
		fmt.Printf("Using config: %s\n", configPath)
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

	fmt.Printf("Analyzing %d files...\n", len(files))

	// Analyze each file
	var totalComplexity int
	var totalFunctions int
	var totalDeadCodeFindings int

	for _, file := range files {
		if err := analyzeFile(file, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing %s: %v\n", file, err)
			continue
		}
	}

	// Print summary
	fmt.Printf("\nAnalysis complete!\n")
	fmt.Printf("Files analyzed: %d\n", len(files))
	fmt.Printf("Total functions: %d\n", totalFunctions)

	if contains(selectAnalyses, "complexity") {
		fmt.Printf("Total complexity: %d\n", totalComplexity)
	}

	if contains(selectAnalyses, "deadcode") {
		fmt.Printf("Dead code findings: %d\n", totalDeadCodeFindings)
	}

	return nil
}

func analyzeFile(filePath string, cfg *config.Config) error {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse file
	ast, err := parser.ParseForLanguage(filePath, content)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Build CFGs for all functions
	builder := analyzer.NewCFGBuilder()
	cfgs, err := builder.BuildAll(ast)
	if err != nil {
		return fmt.Errorf("failed to build CFGs: %w", err)
	}

	fmt.Printf("\n%s:\n", filePath)

	// Run selected analyses
	if contains(selectAnalyses, "complexity") {
		analyzeComplexity(cfgs, cfg)
	}

	if contains(selectAnalyses, "deadcode") {
		analyzeDeadCode(cfgs, filePath)
	}

	return nil
}

func analyzeComplexity(cfgs map[string]*analyzer.CFG, cfg *config.Config) {
	fmt.Println("  Complexity Analysis:")

	for name, cfgItem := range cfgs {
		if name == "__main__" {
			continue // Skip main module
		}

		result := analyzer.CalculateComplexityWithConfig(cfgItem, &cfg.Complexity)

		fmt.Printf("    %s: complexity=%d, risk=%s\n",
			name, result.Complexity, result.RiskLevel)

		if result.Complexity > cfg.Complexity.MediumThreshold {
			fmt.Printf("      ⚠ High complexity detected!\n")
		}
	}
}

func analyzeDeadCode(cfgs map[string]*analyzer.CFG, filePath string) {
	fmt.Println("  Dead Code Analysis:")

	results := analyzer.DetectAll(cfgs, filePath)

	totalFindings := 0
	for name, result := range results {
		if name == "__main__" {
			continue // Skip main module
		}

		if result.HasFindings() {
			fmt.Printf("    %s: %d dead code blocks found\n",
				name, len(result.Findings))

			for _, finding := range result.Findings {
				fmt.Printf("      Line %d: %s (%s)\n",
					finding.StartLine, finding.Description, finding.Reason)
			}

			totalFindings += len(result.Findings)
		}
	}

	if totalFindings == 0 {
		fmt.Println("    ✓ No dead code found")
	}
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
