package analyzer

import (
	"sort"
	"strings"
	"time"

	"github.com/ludo-technologies/jscan/internal/parser"
)

// SeverityLevel represents the severity of a dead code finding
type SeverityLevel string

const (
	// SeverityLevelCritical indicates code that is definitely unreachable
	SeverityLevelCritical SeverityLevel = "critical"

	// SeverityLevelWarning indicates code that is likely unreachable
	SeverityLevelWarning SeverityLevel = "warning"

	// SeverityLevelInfo indicates potential optimization opportunities
	SeverityLevelInfo SeverityLevel = "info"
)

// DeadCodeReason represents the reason why code is considered dead
type DeadCodeReason string

const (
	// ReasonUnreachableAfterReturn indicates code after a return statement
	ReasonUnreachableAfterReturn DeadCodeReason = "unreachable_after_return"

	// ReasonUnreachableAfterBreak indicates code after a break statement
	ReasonUnreachableAfterBreak DeadCodeReason = "unreachable_after_break"

	// ReasonUnreachableAfterContinue indicates code after a continue statement
	ReasonUnreachableAfterContinue DeadCodeReason = "unreachable_after_continue"

	// ReasonUnreachableAfterThrow indicates code after a throw statement
	ReasonUnreachableAfterThrow DeadCodeReason = "unreachable_after_throw"

	// ReasonUnreachableBranch indicates an unreachable branch condition
	ReasonUnreachableBranch DeadCodeReason = "unreachable_branch"

	// ReasonUnreachableAfterInfiniteLoop indicates code after an infinite loop
	ReasonUnreachableAfterInfiniteLoop DeadCodeReason = "unreachable_after_infinite_loop"

	// ReasonUnusedImport indicates an imported name that is never referenced
	ReasonUnusedImport DeadCodeReason = "unused_import"

	// ReasonUnusedExport indicates an exported name that is never imported by other files
	ReasonUnusedExport DeadCodeReason = "unused_export"
)

// DeadCodeFinding represents a single dead code detection result
type DeadCodeFinding struct {
	// Function information
	FunctionName string `json:"function_name"`
	FilePath     string `json:"file_path"`

	// Location information
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`

	// Dead code details
	BlockID     string         `json:"block_id"`
	Code        string         `json:"code"`
	Reason      DeadCodeReason `json:"reason"`
	Severity    SeverityLevel  `json:"severity"`
	Description string         `json:"description"`

	// Context information
	Context []string `json:"context,omitempty"`
}

// DeadCodeResult contains the results of dead code analysis for a single CFG
type DeadCodeResult struct {
	// Function information
	FunctionName string `json:"function_name"`
	FilePath     string `json:"file_path"`

	// Analysis results
	Findings       []*DeadCodeFinding `json:"findings"`
	TotalBlocks    int                `json:"total_blocks"`
	DeadBlocks     int                `json:"dead_blocks"`
	ReachableRatio float64            `json:"reachable_ratio"`

	// Performance metrics
	AnalysisTime time.Duration `json:"analysis_time"`
}

// DeadCodeDetector provides high-level dead code detection functionality
type DeadCodeDetector struct {
	cfg      *CFG
	filePath string // File path for context in findings
}

// NewDeadCodeDetector creates a new dead code detector for the given CFG
func NewDeadCodeDetector(cfg *CFG) *DeadCodeDetector {
	return &DeadCodeDetector{
		cfg:      cfg,
		filePath: "",
	}
}

// NewDeadCodeDetectorWithFilePath creates a new dead code detector with file path context
func NewDeadCodeDetectorWithFilePath(cfg *CFG, filePath string) *DeadCodeDetector {
	return &DeadCodeDetector{
		cfg:      cfg,
		filePath: filePath,
	}
}

// Detect performs dead code detection and returns structured findings
func (dcd *DeadCodeDetector) Detect() *DeadCodeResult {
	startTime := time.Now()

	result := &DeadCodeResult{
		FunctionName: dcd.getFunctionName(),
		FilePath:     dcd.getFilePath(),
		Findings:     make([]*DeadCodeFinding, 0),
		TotalBlocks:  0,
		DeadBlocks:   0,
		AnalysisTime: time.Since(startTime),
	}

	// Handle nil or empty CFG
	if dcd.cfg == nil || dcd.cfg.Blocks == nil {
		return result
	}

	result.TotalBlocks = len(dcd.cfg.Blocks)

	// Use reachability analyzer to find unreachable blocks
	analyzer := NewReachabilityAnalyzer(dcd.cfg)
	reachResult := analyzer.AnalyzeReachability()

	result.ReachableRatio = reachResult.GetReachabilityRatio()

	// Convert unreachable blocks to dead code findings
	unreachableWithStatements := reachResult.GetUnreachableBlocksWithStatements()
	result.DeadBlocks = len(unreachableWithStatements)

	for _, block := range unreachableWithStatements {
		findings := dcd.analyzeDeadBlock(block)
		result.Findings = append(result.Findings, findings...)
	}

	// Sort findings by line number for consistent output
	sort.Slice(result.Findings, func(i, j int) bool {
		return result.Findings[i].StartLine < result.Findings[j].StartLine
	})

	result.AnalysisTime = time.Since(startTime)
	return result
}

// analyzeDeadBlock analyzes a dead block to determine the reason and create findings
func (dcd *DeadCodeDetector) analyzeDeadBlock(block *BasicBlock) []*DeadCodeFinding {
	var findings []*DeadCodeFinding

	// Determine the reason for unreachability
	reason, severity := dcd.determineDeadCodeReason(block)

	// Create a finding for the block
	finding := &DeadCodeFinding{
		FunctionName: dcd.getFunctionName(),
		FilePath:     dcd.getFilePath(),
		BlockID:      block.ID,
		Reason:       reason,
		Severity:     severity,
		Description:  dcd.generateDescription(reason),
	}

	// Extract location from first statement in block
	if len(block.Statements) > 0 {
		firstStmt := block.Statements[0]
		finding.StartLine = firstStmt.Location.StartLine
		finding.EndLine = block.Statements[len(block.Statements)-1].Location.EndLine

		// Generate code snippet
		finding.Code = dcd.getCodeSnippet(block.Statements)
	}

	findings = append(findings, finding)
	return findings
}

// determineDeadCodeReason determines why a block is unreachable
func (dcd *DeadCodeDetector) determineDeadCodeReason(block *BasicBlock) (DeadCodeReason, SeverityLevel) {
	// Check predecessors for terminating statements
	for _, pred := range block.Predecessors {
		if pred.From == nil {
			continue
		}

		// Check last statement in predecessor block
		if len(pred.From.Statements) > 0 {
			lastStmt := pred.From.Statements[len(pred.From.Statements)-1]

			switch lastStmt.Type {
			case parser.NodeReturnStatement:
				return ReasonUnreachableAfterReturn, SeverityLevelCritical
			case parser.NodeBreakStatement:
				return ReasonUnreachableAfterBreak, SeverityLevelCritical
			case parser.NodeContinueStatement:
				return ReasonUnreachableAfterContinue, SeverityLevelCritical
			case parser.NodeThrowStatement:
				return ReasonUnreachableAfterThrow, SeverityLevelCritical
			}
		}
	}

	// Check if block is after an infinite loop
	if strings.Contains(block.ID, "unreachable") {
		return ReasonUnreachableAfterInfiniteLoop, SeverityLevelWarning
	}

	// Default to unreachable branch
	return ReasonUnreachableBranch, SeverityLevelWarning
}

// generateDescription generates a human-readable description for a dead code reason
func (dcd *DeadCodeDetector) generateDescription(reason DeadCodeReason) string {
	descriptions := map[DeadCodeReason]string{
		ReasonUnreachableAfterReturn:       "Code after return statement is unreachable",
		ReasonUnreachableAfterBreak:        "Code after break statement is unreachable",
		ReasonUnreachableAfterContinue:     "Code after continue statement is unreachable",
		ReasonUnreachableAfterThrow:        "Code after throw statement is unreachable",
		ReasonUnreachableBranch:            "This branch is unreachable",
		ReasonUnreachableAfterInfiniteLoop: "Code after infinite loop is unreachable",
		ReasonUnusedImport:                 "Imported name is never used in this file",
		ReasonUnusedExport:                 "Exported name is not imported by any other analyzed file",
	}

	if desc, exists := descriptions[reason]; exists {
		return desc
	}
	return "Code is unreachable"
}

// getCodeSnippet generates a code snippet from statements
func (dcd *DeadCodeDetector) getCodeSnippet(statements []*parser.Node) string {
	if len(statements) == 0 {
		return ""
	}

	var snippets []string
	for _, stmt := range statements {
		// Use a simplified representation for now
		snippets = append(snippets, string(stmt.Type))
	}

	snippet := strings.Join(snippets, "; ")
	if len(snippet) > 100 {
		snippet = snippet[:100] + "..."
	}

	return snippet
}

// getFunctionName returns the function name from the CFG
func (dcd *DeadCodeDetector) getFunctionName() string {
	if dcd.cfg != nil {
		return dcd.cfg.Name
	}
	return ""
}

// getFilePath returns the file path for context
func (dcd *DeadCodeDetector) getFilePath() string {
	return dcd.filePath
}

// DetectAll analyzes dead code for all functions in a file
func DetectAll(cfgs map[string]*CFG, filePath string) map[string]*DeadCodeResult {
	results := make(map[string]*DeadCodeResult)

	for name, cfg := range cfgs {
		detector := NewDeadCodeDetectorWithFilePath(cfg, filePath)
		result := detector.Detect()
		results[name] = result
	}

	return results
}

// HasFindings returns true if there are any dead code findings
func (dcr *DeadCodeResult) HasFindings() bool {
	return len(dcr.Findings) > 0
}

// GetCriticalFindings returns only critical severity findings
func (dcr *DeadCodeResult) GetCriticalFindings() []*DeadCodeFinding {
	var critical []*DeadCodeFinding
	for _, finding := range dcr.Findings {
		if finding.Severity == SeverityLevelCritical {
			critical = append(critical, finding)
		}
	}
	return critical
}

// GetWarningFindings returns only warning severity findings
func (dcr *DeadCodeResult) GetWarningFindings() []*DeadCodeFinding {
	var warnings []*DeadCodeFinding
	for _, finding := range dcr.Findings {
		if finding.Severity == SeverityLevelWarning {
			warnings = append(warnings, finding)
		}
	}
	return warnings
}
