package analyzer

import (
	corecfg "github.com/ludo-technologies/codescan-core/cfg"
	"github.com/ludo-technologies/jscan/internal/parser"
)

// JSStatementClassifier implements corecfg.StatementClassifier for JavaScript/TypeScript.
type JSStatementClassifier struct{}

var _ corecfg.StatementClassifier = (*JSStatementClassifier)(nil)

func (c *JSStatementClassifier) IsReturn(stmt any) bool {
	node, ok := stmt.(*parser.Node)
	return ok && node != nil && node.Type == parser.NodeReturnStatement
}

func (c *JSStatementClassifier) IsBreak(stmt any) bool {
	node, ok := stmt.(*parser.Node)
	return ok && node != nil && node.Type == parser.NodeBreakStatement
}

func (c *JSStatementClassifier) IsContinue(stmt any) bool {
	node, ok := stmt.(*parser.Node)
	return ok && node != nil && node.Type == parser.NodeContinueStatement
}

func (c *JSStatementClassifier) IsThrow(stmt any) bool {
	node, ok := stmt.(*parser.Node)
	return ok && node != nil && node.Type == parser.NodeThrowStatement
}
