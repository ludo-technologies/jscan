package analyzer

import (
	"sort"
	"strings"
	"time"

	corecfg "github.com/ludo-technologies/codescan-core/cfg"
	"github.com/ludo-technologies/jscan/internal/parser"
)

type SeverityLevel string

const (
	SeverityLevelCritical SeverityLevel = "critical"
	SeverityLevelWarning  SeverityLevel = "warning"
	SeverityLevelInfo     SeverityLevel = "info"
)

type DeadCodeReason string

const (
	ReasonUnreachableAfterReturn       DeadCodeReason = "unreachable_after_return"
	ReasonUnreachableAfterBreak        DeadCodeReason = "unreachable_after_break"
	ReasonUnreachableAfterContinue     DeadCodeReason = "unreachable_after_continue"
	ReasonUnreachableAfterThrow        DeadCodeReason = "unreachable_after_throw"
	ReasonUnreachableBranch            DeadCodeReason = "unreachable_branch"
	ReasonUnreachableAfterInfiniteLoop DeadCodeReason = "unreachable_after_infinite_loop"
	ReasonUnusedImport                 DeadCodeReason = "unused_import"
	ReasonUnusedExport                 DeadCodeReason = "unused_export"
	ReasonOrphanFile                   DeadCodeReason = "orphan_file"
	ReasonUnusedExportedFunction       DeadCodeReason = "unused_exported_function"
)

type DeadCodeFinding struct {
	FunctionName string         `json:"function_name"`
	FilePath     string         `json:"file_path"`
	StartLine    int            `json:"start_line"`
	EndLine      int            `json:"end_line"`
	BlockID      string         `json:"block_id"`
	Code         string         `json:"code"`
	Reason       DeadCodeReason `json:"reason"`
	Severity     SeverityLevel  `json:"severity"`
	Description  string         `json:"description"`
	Context      []string       `json:"context,omitempty"`
}

type DeadCodeResult struct {
	FunctionName   string             `json:"function_name"`
	FilePath       string             `json:"file_path"`
	Findings       []*DeadCodeFinding `json:"findings"`
	TotalBlocks    int                `json:"total_blocks"`
	DeadBlocks     int                `json:"dead_blocks"`
	ReachableRatio float64            `json:"reachable_ratio"`
	AnalysisTime   time.Duration      `json:"analysis_time"`
}

type DeadCodeDetector struct {
	cfg      *corecfg.CFG
	filePath string
}

func NewDeadCodeDetector(cfg *corecfg.CFG) *DeadCodeDetector {
	return &DeadCodeDetector{cfg: cfg, filePath: ""}
}

func NewDeadCodeDetectorWithFilePath(cfg *corecfg.CFG, filePath string) *DeadCodeDetector {
	return &DeadCodeDetector{cfg: cfg, filePath: filePath}
}

func (dcd *DeadCodeDetector) Detect() *DeadCodeResult {
	startTime := time.Now()

	result := &DeadCodeResult{
		FunctionName: dcd.getFunctionName(),
		FilePath:     dcd.getFilePath(),
		Findings:     make([]*DeadCodeFinding, 0),
		AnalysisTime: time.Since(startTime),
	}

	if dcd.cfg == nil || dcd.cfg.Blocks == nil {
		return result
	}

	result.TotalBlocks = len(dcd.cfg.Blocks)

	analyzer := NewReachabilityAnalyzer(dcd.cfg)
	reachResult := analyzer.AnalyzeReachability()

	result.ReachableRatio = reachResult.GetReachabilityRatio()

	unreachableWithStatements := reachResult.GetUnreachableBlocksWithStatements()
	result.DeadBlocks = len(unreachableWithStatements)

	for _, block := range unreachableWithStatements {
		findings := dcd.analyzeDeadBlock(block)
		result.Findings = append(result.Findings, findings...)
	}

	sort.Slice(result.Findings, func(i, j int) bool {
		return result.Findings[i].StartLine < result.Findings[j].StartLine
	})

	result.AnalysisTime = time.Since(startTime)
	return result
}

func (dcd *DeadCodeDetector) analyzeDeadBlock(block *corecfg.BasicBlock) []*DeadCodeFinding {
	var findings []*DeadCodeFinding

	reason, severity := dcd.determineDeadCodeReason(block)

	finding := &DeadCodeFinding{
		FunctionName: dcd.getFunctionName(),
		FilePath:     dcd.getFilePath(),
		BlockID:      block.ID,
		Reason:       reason,
		Severity:     severity,
		Description:  dcd.generateDescription(reason),
	}

	if len(block.Statements) > 0 {
		if firstStmt, ok := block.Statements[0].(*parser.Node); ok && firstStmt != nil {
			finding.StartLine = firstStmt.Location.StartLine
			if lastStmt, ok2 := block.Statements[len(block.Statements)-1].(*parser.Node); ok2 && lastStmt != nil {
				finding.EndLine = lastStmt.Location.EndLine
			}
		}
		finding.Code = dcd.getCodeSnippet(block.Statements)
	}

	findings = append(findings, finding)
	return findings
}

func (dcd *DeadCodeDetector) determineDeadCodeReason(block *corecfg.BasicBlock) (DeadCodeReason, SeverityLevel) {
	for _, pred := range block.Predecessors {
		if pred.From == nil {
			continue
		}

		if len(pred.From.Statements) > 0 {
			if lastStmt, ok := pred.From.Statements[len(pred.From.Statements)-1].(*parser.Node); ok && lastStmt != nil {
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
	}

	if strings.Contains(block.ID, "unreachable") {
		return ReasonUnreachableAfterInfiniteLoop, SeverityLevelWarning
	}

	return ReasonUnreachableBranch, SeverityLevelWarning
}

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
		ReasonOrphanFile:                   "File is not imported by any other analyzed file",
		ReasonUnusedExportedFunction:       "Exported function is not imported by any other analyzed file",
	}

	if desc, exists := descriptions[reason]; exists {
		return desc
	}
	return "Code is unreachable"
}

func (dcd *DeadCodeDetector) getCodeSnippet(statements []any) string {
	if len(statements) == 0 {
		return ""
	}

	var snippets []string
	for _, rawStmt := range statements {
		if stmt, ok := rawStmt.(*parser.Node); ok && stmt != nil {
			snippets = append(snippets, string(stmt.Type))
		}
	}

	snippet := strings.Join(snippets, "; ")
	if len(snippet) > 100 {
		snippet = snippet[:100] + "..."
	}

	return snippet
}

func (dcd *DeadCodeDetector) getFunctionName() string {
	if dcd.cfg != nil {
		return dcd.cfg.Name
	}
	return ""
}

func (dcd *DeadCodeDetector) getFilePath() string {
	return dcd.filePath
}

func DetectAll(cfgs map[string]*corecfg.CFG, filePath string) map[string]*DeadCodeResult {
	results := make(map[string]*DeadCodeResult)

	for name, cfg := range cfgs {
		detector := NewDeadCodeDetectorWithFilePath(cfg, filePath)
		result := detector.Detect()
		results[name] = result
	}

	return results
}

func (dcr *DeadCodeResult) HasFindings() bool {
	return len(dcr.Findings) > 0
}

func (dcr *DeadCodeResult) GetCriticalFindings() []*DeadCodeFinding {
	var critical []*DeadCodeFinding
	for _, finding := range dcr.Findings {
		if finding.Severity == SeverityLevelCritical {
			critical = append(critical, finding)
		}
	}
	return critical
}

func (dcr *DeadCodeResult) GetWarningFindings() []*DeadCodeFinding {
	var warnings []*DeadCodeFinding
	for _, finding := range dcr.Findings {
		if finding.Severity == SeverityLevelWarning {
			warnings = append(warnings, finding)
		}
	}
	return warnings
}
