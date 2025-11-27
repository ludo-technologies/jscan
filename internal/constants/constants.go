package constants

// Tool name and related constants
const (
	// ToolName is the name of this tool
	ToolName = "jscan"

	// ConfigFileName is the default config file name
	ConfigFileName = ".jscan.toml"

	// EnvVarPrefix is the prefix for environment variables
	EnvVarPrefix = "JSCAN"
)

// Analysis type constants
const (
	AnalysisComplexity = "complexity"
	AnalysisDeadCode   = "deadcode"
	AnalysisClones     = "clones"
	AnalysisCBO        = "cbo"
	AnalysisSystem     = "system"
)

// Output format constants
const (
	OutputFormatText = "text"
	OutputFormatJSON = "json"
	OutputFormatHTML = "html"
	OutputFormatCSV  = "csv"
)

// Clone detection threshold constants
const (
	DefaultType1CloneThreshold = 0.98
	DefaultType2CloneThreshold = 0.95
	DefaultType3CloneThreshold = 0.85
	DefaultType4CloneThreshold = 0.70
)
