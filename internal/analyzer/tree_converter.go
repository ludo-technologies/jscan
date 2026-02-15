package analyzer

import (
	"fmt"

	"github.com/ludo-technologies/codescan-core/apted"
	"github.com/ludo-technologies/jscan/internal/parser"
)

// TreeConverter converts parser AST nodes to APTED tree nodes
type TreeConverter struct {
	nextID int
}

// NewTreeConverter creates a new tree converter
func NewTreeConverter() *TreeConverter {
	return &TreeConverter{nextID: 0}
}

// ConvertAST converts a parser AST node to an APTED tree
func (tc *TreeConverter) ConvertAST(astNode *parser.Node) *apted.TreeNode {
	if astNode == nil {
		return nil
	}

	// Create tree node with simplified label
	label := tc.getNodeLabel(astNode)
	treeNode := apted.NewTreeNode(tc.nextID, label)
	tc.nextID++

	// Store reference to original AST node
	treeNode.OriginalNode = astNode

	// Convert children recursively
	for _, child := range astNode.Children {
		if childNode := tc.ConvertAST(child); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert body nodes
	for _, bodyNode := range astNode.Body {
		if childNode := tc.ConvertAST(bodyNode); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert params
	for _, param := range astNode.Params {
		if childNode := tc.ConvertAST(param); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert cases
	for _, caseNode := range astNode.Cases {
		if childNode := tc.ConvertAST(caseNode); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert handlers
	for _, handler := range astNode.Handlers {
		if childNode := tc.ConvertAST(handler); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert arguments
	for _, arg := range astNode.Arguments {
		if childNode := tc.ConvertAST(arg); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert declarations
	for _, decl := range astNode.Declarations {
		if childNode := tc.ConvertAST(decl); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert specifiers
	for _, spec := range astNode.Specifiers {
		if childNode := tc.ConvertAST(spec); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	// Convert individual nodes
	if astNode.Test != nil {
		if childNode := tc.ConvertAST(astNode.Test); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Consequent != nil {
		if childNode := tc.ConvertAST(astNode.Consequent); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Alternate != nil {
		if childNode := tc.ConvertAST(astNode.Alternate); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Init != nil {
		if childNode := tc.ConvertAST(astNode.Init); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Update != nil {
		if childNode := tc.ConvertAST(astNode.Update); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Handler != nil {
		if childNode := tc.ConvertAST(astNode.Handler); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Finalizer != nil {
		if childNode := tc.ConvertAST(astNode.Finalizer); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Left != nil {
		if childNode := tc.ConvertAST(astNode.Left); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Right != nil {
		if childNode := tc.ConvertAST(astNode.Right); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Argument != nil {
		if childNode := tc.ConvertAST(astNode.Argument); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Callee != nil {
		if childNode := tc.ConvertAST(astNode.Callee); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Object != nil {
		if childNode := tc.ConvertAST(astNode.Object); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}
	if astNode.Property != nil {
		if childNode := tc.ConvertAST(astNode.Property); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	return treeNode
}

// getNodeLabel extracts a meaningful label from the AST node
func (tc *TreeConverter) getNodeLabel(astNode *parser.Node) string {
	// Use the node type as the primary label
	label := string(astNode.Type)

	// For some node types, include additional information
	switch astNode.Type {
	case parser.NodeIdentifier:
		if astNode.Name != "" {
			label = fmt.Sprintf("Identifier(%s)", astNode.Name)
		}
	case parser.NodeLiteral, parser.NodeStringLiteral, parser.NodeNumberLiteral:
		if astNode.Value != nil {
			label = fmt.Sprintf("Literal(%v)", astNode.Value)
		}
	case parser.NodeFunction, parser.NodeAsyncFunction, parser.NodeArrowFunction:
		if astNode.Name != "" {
			label = fmt.Sprintf("Function(%s)", astNode.Name)
		}
	case parser.NodeClass, parser.NodeClassExpression:
		if astNode.Name != "" {
			label = fmt.Sprintf("Class(%s)", astNode.Name)
		}
	case parser.NodeBinaryExpression, parser.NodeUnaryExpression, parser.NodeLogicalExpression:
		if astNode.Operator != "" {
			label = fmt.Sprintf("%s(%s)", astNode.Type, astNode.Operator)
		}
	case parser.NodeVariableDeclaration:
		if astNode.Kind != "" {
			label = fmt.Sprintf("VariableDeclaration(%s)", astNode.Kind)
		}
	}

	return label
}
