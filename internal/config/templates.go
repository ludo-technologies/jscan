package config

import "strconv"

// ProjectType represents the type of JavaScript/TypeScript project
type ProjectType string

const (
	ProjectTypeGeneric     ProjectType = "generic"
	ProjectTypeReact       ProjectType = "react"
	ProjectTypeVue         ProjectType = "vue"
	ProjectTypeNodeBackend ProjectType = "node"
)

// Strictness represents the analysis strictness level
type Strictness string

const (
	StrictnessRelaxed  Strictness = "relaxed"
	StrictnessStandard Strictness = "standard"
	StrictnessStrict   Strictness = "strict"
)

// ProjectPreset holds configuration presets for different project types
type ProjectPreset struct {
	IncludePatterns []string
	ExcludePatterns []string
}

// StrictnessPreset holds threshold values for different strictness levels
type StrictnessPreset struct {
	LowThreshold    int
	MediumThreshold int
	MaxComplexity   int
}

// GetProjectPresets returns presets for different project types
func GetProjectPresets() map[ProjectType]ProjectPreset {
	return map[ProjectType]ProjectPreset{
		ProjectTypeGeneric: {
			IncludePatterns: []string{
				"**/*.js",
				"**/*.ts",
				"**/*.jsx",
				"**/*.tsx",
			},
			ExcludePatterns: []string{
				"**/node_modules/**",
				"**/dist/**",
				"**/build/**",
				"**/*.min.js",
				"**/*.bundle.js",
			},
		},
		ProjectTypeReact: {
			IncludePatterns: []string{
				"**/*.js",
				"**/*.ts",
				"**/*.jsx",
				"**/*.tsx",
			},
			ExcludePatterns: []string{
				"**/node_modules/**",
				"**/dist/**",
				"**/build/**",
				"**/.next/**",
				"**/coverage/**",
				"**/*.min.js",
				"**/*.bundle.js",
			},
		},
		ProjectTypeVue: {
			IncludePatterns: []string{
				"**/*.js",
				"**/*.ts",
				"**/*.jsx",
				"**/*.tsx",
				"**/*.vue",
			},
			ExcludePatterns: []string{
				"**/node_modules/**",
				"**/dist/**",
				"**/build/**",
				"**/.nuxt/**",
				"**/coverage/**",
				"**/*.min.js",
				"**/*.bundle.js",
			},
		},
		ProjectTypeNodeBackend: {
			IncludePatterns: []string{
				"**/*.js",
				"**/*.ts",
				"**/*.mjs",
				"**/*.cjs",
			},
			ExcludePatterns: []string{
				"**/node_modules/**",
				"**/dist/**",
				"**/build/**",
				"**/test/**",
				"**/tests/**",
				"**/__tests__/**",
				"**/*.min.js",
				"**/*.bundle.js",
			},
		},
	}
}

// GetStrictnessPresets returns presets for different strictness levels
func GetStrictnessPresets() map[Strictness]StrictnessPreset {
	return map[Strictness]StrictnessPreset{
		StrictnessRelaxed: {
			LowThreshold:    15,
			MediumThreshold: 30,
			MaxComplexity:   0, // No limit
		},
		StrictnessStandard: {
			LowThreshold:    10,
			MediumThreshold: 20,
			MaxComplexity:   0, // No limit
		},
		StrictnessStrict: {
			LowThreshold:    5,
			MediumThreshold: 10,
			MaxComplexity:   15,
		},
	}
}

// GetFullConfigTemplate returns the documented config template as JSONC
func GetFullConfigTemplate(projectType ProjectType, strictness Strictness) string {
	projectPresets := GetProjectPresets()
	strictnessPresets := GetStrictnessPresets()

	preset := projectPresets[projectType]
	strict := strictnessPresets[strictness]

	// Build include patterns string
	includePatterns := formatJSONArray(preset.IncludePatterns)
	excludePatterns := formatJSONArray(preset.ExcludePatterns)

	return `{
  // jscan Configuration
  // Documentation: https://github.com/ludo-technologies/jscan

  // ============================================================================
  // COMPLEXITY ANALYSIS
  // ============================================================================
  // Analyzes cyclomatic complexity of functions to identify hard-to-maintain code
  "complexity": {
    // Enable/disable complexity analysis
    "enabled": true,

    // Threshold for LOW risk (recommended: 10)
    // Functions with complexity <= this value are considered easy to maintain
    "lowThreshold": ` + strconv.Itoa(strict.LowThreshold) + `,

    // Threshold for MEDIUM risk
    // Functions above lowThreshold but <= this value need attention
    // Functions above this are considered HIGH risk
    "mediumThreshold": ` + strconv.Itoa(strict.MediumThreshold) + `,

    // Maximum allowed complexity (0 = no limit)
    // Set this for CI/CD enforcement to fail builds on complex functions
    "maxComplexity": ` + strconv.Itoa(strict.MaxComplexity) + `,

    // Report functions with complexity = 1 (simple functions)
    "reportUnchanged": false
  },

  // ============================================================================
  // DEAD CODE DETECTION
  // ============================================================================
  // Detects unreachable code that will never execute
  "deadCode": {
    // Enable/disable dead code detection
    "enabled": true,

    // Minimum severity level to report: "info", "warning", "critical"
    "minSeverity": "warning",

    // Detection options - enable/disable specific dead code patterns
    "detectAfterReturn": true,
    "detectAfterBreak": true,
    "detectAfterContinue": true,
    "detectAfterThrow": true,
    "detectUnreachableBranches": true
  },

  // ============================================================================
  // OUTPUT SETTINGS
  // ============================================================================
  "output": {
    // Output format: "text", "json", "html"
    "format": "text",

    // Show detailed breakdown of results
    "showDetails": true,

    // Use colors in terminal output (disable for CI logs)
    "colorize": true
  },

  // ============================================================================
  // ANALYSIS SCOPE
  // ============================================================================
  // Controls which files are analyzed
  "analysis": {
    // File patterns to include (glob patterns)
    "include": ` + includePatterns + `,

    // File patterns to exclude (glob patterns)
    "exclude": ` + excludePatterns + `,

    // Maximum file size to analyze in KB (0 = no limit)
    "maxFileSize": 1000,

    // Number of parallel workers (0 = auto-detect based on CPU)
    "workers": 0
  }
}
`
}

// GetMinimalConfigTemplate returns a minimal config template
func GetMinimalConfigTemplate() string {
	return `{
  // jscan Configuration (minimal)
  // See full options: https://github.com/ludo-technologies/jscan

  "complexity": {
    "enabled": true,
    "lowThreshold": 10,
    "mediumThreshold": 20
  },

  "deadCode": {
    "enabled": true,
    "minSeverity": "warning"
  },

  "analysis": {
    "include": ["**/*.js", "**/*.ts", "**/*.jsx", "**/*.tsx"],
    "exclude": ["**/node_modules/**", "**/dist/**"]
  }
}
`
}

// formatJSONArray formats a string slice as a JSON array with proper indentation
func formatJSONArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}

	result := "[\n"
	for i, item := range items {
		result += `      "` + item + `"`
		if i < len(items)-1 {
			result += ","
		}
		result += "\n"
	}
	result += "    ]"
	return result
}
