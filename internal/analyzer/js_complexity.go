package analyzer

import (
	corecfg "github.com/ludo-technologies/codescan-core/cfg"
	"github.com/ludo-technologies/jscan/internal/parser"
)

// JSComplexityContributor implements corecfg.ComplexityContributor for JavaScript/TypeScript.
// It counts logical operators (&&, ||, ??) and ternary expressions (? :).
type JSComplexityContributor struct{}

var _ corecfg.ComplexityContributor = (*JSComplexityContributor)(nil)

func (c *JSComplexityContributor) ContributeComplexity(block *corecfg.BasicBlock) ([]corecfg.ComplexityContribution, error) {
	var contributions []corecfg.ComplexityContribution
	logicalCount := 0
	ternaryCount := 0

	for _, stmt := range block.Statements {
		node, ok := stmt.(*parser.Node)
		if !ok || node == nil {
			continue
		}
		// Skip function-like statements — they have their own CFGs
		if isNestedFunction(node) {
			continue
		}
		countJSComplexity(node, &logicalCount, &ternaryCount)
	}

	if logicalCount > 0 {
		contributions = append(contributions, corecfg.ComplexityContribution{
			Count:       logicalCount,
			Description: "logical_operators",
		})
	}
	if ternaryCount > 0 {
		contributions = append(contributions, corecfg.ComplexityContribution{
			Count:       ternaryCount,
			Description: "ternary_operators",
		})
	}

	return contributions, nil
}

func countJSComplexity(node *parser.Node, logicalCount, ternaryCount *int) {
	if node == nil {
		return
	}

	if node.Type == parser.NodeLogicalExpression {
		*logicalCount++
	}
	if node.Type == parser.NodeConditionalExpression {
		*ternaryCount++
	}

	node.Walk(func(n *parser.Node) bool {
		if n == node {
			return true
		}
		// Don't descend into nested function bodies — they have their own CFGs
		if isNestedFunction(n) {
			return false
		}
		if n.Type == parser.NodeLogicalExpression {
			*logicalCount++
		}
		if n.Type == parser.NodeConditionalExpression {
			*ternaryCount++
		}
		return true
	})
}

// isNestedFunction returns true for AST nodes that represent function boundaries.
// These nodes have their own CFGs and should not contribute to the parent's complexity.
func isNestedFunction(node *parser.Node) bool {
	switch node.Type {
	case parser.NodeFunction, parser.NodeFunctionExpression, parser.NodeArrowFunction,
		parser.NodeAsyncFunction, parser.NodeGeneratorFunction, parser.NodeMethodDefinition:
		return true
	}
	return false
}
