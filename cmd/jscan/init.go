package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ludo-technologies/jscan/internal/config"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	initConfigPath  string
	initForce       bool
	initMinimal     bool
	initInteractive bool
)

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a jscan configuration file",
		Long: `Generate a documented jscan configuration file with sensible defaults.

By default, creates jscan.config.json in the current directory with full
documentation. Use --interactive for a guided setup wizard.

Examples:
  # Create jscan.config.json in current directory
  jscan init

  # Custom output path
  jscan init --config custom.json

  # Overwrite existing file
  jscan init --force

  # Generate smaller config with essential options only
  jscan init --minimal

  # Interactive setup wizard
  jscan init --interactive
  jscan init -i`,
		RunE: runInit,
	}

	cmd.Flags().StringVarP(&initConfigPath, "config", "c", "jscan.config.json",
		"Output path for the config file")
	cmd.Flags().BoolVarP(&initForce, "force", "f", false,
		"Overwrite existing config file")
	cmd.Flags().BoolVar(&initMinimal, "minimal", false,
		"Generate minimal config with essential options only")
	cmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false,
		"Interactive setup wizard")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	var projectType config.ProjectType = config.ProjectTypeGeneric
	var strictness config.Strictness = config.StrictnessStandard

	// Run interactive setup if requested
	if initInteractive {
		var err error
		projectType, strictness, err = runInteractiveSetup()
		if err != nil {
			return err
		}
	}

	// Check if file exists
	if !initForce {
		if _, err := os.Stat(initConfigPath); err == nil {
			return fmt.Errorf("%s already exists. Use --force to overwrite", initConfigPath)
		}
	}

	// Check if parent directory exists
	dir := filepath.Dir(initConfigPath)
	if dir != "." && dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", dir)
		}
	}

	// Generate config content
	var content string
	if initMinimal {
		content = config.GetMinimalConfigTemplate()
	} else {
		content = config.GetFullConfigTemplate(projectType, strictness)
	}

	// Write to file
	if err := os.WriteFile(initConfigPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Print success message
	absPath, _ := filepath.Abs(initConfigPath)
	fmt.Printf("Created %s\n", absPath)
	fmt.Println("\nRun 'jscan analyze .' to analyze your project.")

	return nil
}

func runInteractiveSetup() (config.ProjectType, config.Strictness, error) {
	fmt.Println()
	fmt.Println("jscan Configuration Setup")
	fmt.Println("=========================")
	fmt.Println()

	// Project type selection
	projectTypes := []struct {
		Label string
		Value config.ProjectType
	}{
		{"Generic JavaScript/TypeScript", config.ProjectTypeGeneric},
		{"React/Next.js", config.ProjectTypeReact},
		{"Vue/Nuxt", config.ProjectTypeVue},
		{"Node.js Backend", config.ProjectTypeNodeBackend},
	}

	projectTemplates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U0001F449 {{ .Label | cyan }}",
		Inactive: "   {{ .Label | white }}",
		Selected: "\U00002705 {{ .Label | green }}",
	}

	projectPrompt := promptui.Select{
		Label:     "What type of project is this?",
		Items:     projectTypes,
		Templates: projectTemplates,
	}

	projectIdx, _, err := projectPrompt.Run()
	if err != nil {
		return "", "", fmt.Errorf("project selection cancelled: %w", err)
	}
	selectedProject := projectTypes[projectIdx].Value

	fmt.Println()

	// Strictness selection
	strictnessLevels := []struct {
		Label       string
		Description string
		Value       config.Strictness
	}{
		{"Standard (recommended)", "Balanced thresholds for most projects", config.StrictnessStandard},
		{"Relaxed", "Higher thresholds, fewer warnings", config.StrictnessRelaxed},
		{"Strict", "Lower thresholds, CI/CD enforcement", config.StrictnessStrict},
	}

	strictnessTemplates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U0001F449 {{ .Label | cyan }} - {{ .Description | faint }}",
		Inactive: "   {{ .Label | white }} - {{ .Description | faint }}",
		Selected: "\U00002705 {{ .Label | green }}",
	}

	strictnessPrompt := promptui.Select{
		Label:     "How strict should the analysis be?",
		Items:     strictnessLevels,
		Templates: strictnessTemplates,
	}

	strictnessIdx, _, err := strictnessPrompt.Run()
	if err != nil {
		return "", "", fmt.Errorf("strictness selection cancelled: %w", err)
	}
	selectedStrictness := strictnessLevels[strictnessIdx].Value

	fmt.Println()

	// Output path prompt
	outputPrompt := promptui.Prompt{
		Label:   "Output file path",
		Default: initConfigPath,
	}

	outputPath, err := outputPrompt.Run()
	if err != nil {
		return "", "", fmt.Errorf("output path input cancelled: %w", err)
	}

	// Update the config path with user input
	if outputPath != "" {
		initConfigPath = outputPath
	}

	fmt.Println()
	fmt.Printf("Creating %s... ", initConfigPath)

	return selectedProject, selectedStrictness, nil
}
