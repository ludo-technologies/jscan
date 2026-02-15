package analyzer

import (
	"fmt"

	corecfg "github.com/ludo-technologies/codescan-core/cfg"
	"github.com/ludo-technologies/jscan/internal/config"
	"github.com/ludo-technologies/jscan/internal/parser"
)

// ComplexityResult holds cyclomatic complexity metrics for a function or method
type ComplexityResult struct {
	Complexity          int
	Edges               int
	Nodes               int
	ConnectedComponents int
	FunctionName        string
	StartLine           int
	StartCol            int
	EndLine             int
	NestingDepth        int
	IfStatements        int
	LoopStatements      int
	ExceptionHandlers   int
	SwitchCases         int
	LogicalOperators    int
	TernaryOperators    int
	RiskLevel           string
}

func (cr *ComplexityResult) GetComplexity() int      { return cr.Complexity }
func (cr *ComplexityResult) GetFunctionName() string { return cr.FunctionName }
func (cr *ComplexityResult) GetRiskLevel() string    { return cr.RiskLevel }

func (cr *ComplexityResult) GetDetailedMetrics() map[string]int {
	return map[string]int{
		"nodes":              cr.Nodes,
		"edges":              cr.Edges,
		"if_statements":      cr.IfStatements,
		"loop_statements":    cr.LoopStatements,
		"exception_handlers": cr.ExceptionHandlers,
		"switch_cases":       cr.SwitchCases,
		"logical_operators":  cr.LogicalOperators,
		"ternary_operators":  cr.TernaryOperators,
	}
}

func (cr *ComplexityResult) String() string {
	return fmt.Sprintf("Function: %s, Complexity: %d, Risk: %s",
		cr.FunctionName, cr.Complexity, cr.RiskLevel)
}

// CalculateComplexity computes McCabe cyclomatic complexity for a CFG using default thresholds
func CalculateComplexity(cfg *corecfg.CFG) *ComplexityResult {
	defaultConfig := config.DefaultConfig()
	return CalculateComplexityWithConfig(cfg, &defaultConfig.Complexity)
}

// CalculateComplexityWithConfig computes McCabe cyclomatic complexity using provided configuration
func CalculateComplexityWithConfig(cfg *corecfg.CFG, complexityConfig *config.ComplexityConfig) *ComplexityResult {
	if cfg == nil {
		return &ComplexityResult{
			Complexity: 0,
			RiskLevel:  "low",
		}
	}

	coreResult, err := corecfg.ComputeComplexity(cfg, corecfg.ComplexityConfig{
		Contributor: &JSComplexityContributor{},
	})
	if err != nil || coreResult == nil {
		return &ComplexityResult{
			Complexity:   1,
			RiskLevel:    "low",
			FunctionName: cfg.Name,
		}
	}

	// Extract JS-specific contribution counts
	logicalOps := 0
	ternaryOps := 0
	for _, contrib := range coreResult.Contributions {
		switch contrib.Description {
		case "logical_operators":
			logicalOps += contrib.Count
		case "ternary_operators":
			ternaryOps += contrib.Count
		}
	}

	// Count edges and nodes from edge breakdown
	totalEdges := 0
	loopStatements := 0
	exceptionHandlers := 0
	for edgeType, count := range coreResult.EdgeBreakdown {
		totalEdges += count
		switch edgeType {
		case corecfg.EdgeLoop:
			loopStatements += count
		case corecfg.EdgeException:
			exceptionHandlers += count
		}
	}

	// corecfg counts blocks with EdgeLoop back-edges as decision points, but
	// jscan's CFG already creates EdgeCondTrue/EdgeCondFalse at loop headers.
	// Subtract loop back-edges to avoid double-counting each loop.
	complexity := coreResult.McCabe - loopStatements
	if complexity < 1 {
		complexity = 1
	}

	riskLevel := determineRiskLevel(complexity, complexityConfig)

	return &ComplexityResult{
		Complexity:        complexity,
		Edges:             totalEdges,
		Nodes:             cfg.Size(),
		IfStatements:      coreResult.DecisionPoints,
		LoopStatements:    loopStatements,
		ExceptionHandlers: exceptionHandlers,
		LogicalOperators:  logicalOps,
		TernaryOperators:  ternaryOps,
		RiskLevel:         riskLevel,
		FunctionName:      cfg.Name,
	}
}

func determineRiskLevel(complexity int, cfg *config.ComplexityConfig) string {
	if complexity > cfg.MediumThreshold {
		return "high"
	} else if complexity > cfg.LowThreshold {
		return "medium"
	}
	return "low"
}

// CalculateNestingDepth calculates the maximum nesting depth of a function
func CalculateNestingDepth(node *parser.Node) int {
	if node == nil {
		return 0
	}

	maxDepth := 0
	currentDepth := 0

	node.Walk(func(n *parser.Node) bool {
		if isControlStructure(n) {
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		}
		return true
	})

	return maxDepth
}

func isControlStructure(node *parser.Node) bool {
	switch node.Type {
	case parser.NodeIfStatement, parser.NodeSwitchStatement,
		parser.NodeForStatement, parser.NodeForInStatement, parser.NodeForOfStatement,
		parser.NodeWhileStatement, parser.NodeDoWhileStatement,
		parser.NodeTryStatement, parser.NodeCatchClause:
		return true
	}
	return false
}

// ComplexityAnalyzer analyzes complexity for multiple functions
type ComplexityAnalyzer struct {
	cfg *config.ComplexityConfig
}

func NewComplexityAnalyzer(cfg *config.ComplexityConfig) *ComplexityAnalyzer {
	return &ComplexityAnalyzer{cfg: cfg}
}

func (ca *ComplexityAnalyzer) AnalyzeFile(ast *parser.Node) ([]*ComplexityResult, error) {
	if ast == nil {
		return nil, fmt.Errorf("AST is nil")
	}

	builder := NewCFGBuilder()
	cfgs, err := builder.BuildAll(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to build CFGs: %w", err)
	}

	var results []*ComplexityResult
	for _, cfg := range cfgs {
		result := CalculateComplexityWithConfig(cfg, ca.cfg)
		results = append(results, result)
	}

	return results, nil
}
