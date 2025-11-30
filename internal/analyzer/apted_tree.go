package analyzer

import (
	"fmt"

	"github.com/ludo-technologies/jscan/internal/parser"
)

// TreeNode represents a node in the ordered tree for APTED algorithm
type TreeNode struct {
	// Unique identifier for this node
	ID int

	// Label for the node (typically the node type or value)
	Label string

	// Tree structure
	Children []*TreeNode
	Parent   *TreeNode

	// APTED-specific fields for optimization
	PostOrderID  int  // Post-order traversal position
	LeftMostLeaf int  // Left-most leaf descendant
	KeyRoot      bool // Whether this node is a key root

	// Optional metadata from original AST
	OriginalNode *parser.Node
}

// NewTreeNode creates a new tree node with the given ID and label
func NewTreeNode(id int, label string) *TreeNode {
	return &TreeNode{
		ID:       id,
		Label:    label,
		Children: []*TreeNode{},
	}
}

// AddChild adds a child node to this node
func (t *TreeNode) AddChild(child *TreeNode) {
	if child != nil {
		child.Parent = t
		t.Children = append(t.Children, child)
	}
}

// IsLeaf returns true if this node has no children
func (t *TreeNode) IsLeaf() bool {
	return len(t.Children) == 0
}

// Size returns the size of the subtree rooted at this node
func (t *TreeNode) Size() int {
	return t.SizeWithDepthLimit(1000) // Default recursion limit
}

// SizeWithDepthLimit returns the size with maximum recursion depth limit
func (t *TreeNode) SizeWithDepthLimit(maxDepth int) int {
	if maxDepth <= 0 {
		return 1 // Return 1 to avoid infinite loops, treat as leaf
	}

	size := 1
	for _, child := range t.Children {
		size += child.SizeWithDepthLimit(maxDepth - 1)
	}
	return size
}

// Height returns the height of the subtree rooted at this node
func (t *TreeNode) Height() int {
	return t.HeightWithDepthLimit(1000) // Default recursion limit
}

// HeightWithDepthLimit returns the height with maximum recursion depth limit
func (t *TreeNode) HeightWithDepthLimit(maxDepth int) int {
	if maxDepth <= 0 {
		return 0 // Treat as leaf when depth limit reached
	}

	if t.IsLeaf() {
		return 0
	}

	maxHeight := 0
	for _, child := range t.Children {
		if h := child.HeightWithDepthLimit(maxDepth - 1); h > maxHeight {
			maxHeight = h
		}
	}
	return maxHeight + 1
}

// String returns a string representation of the node
func (t *TreeNode) String() string {
	return fmt.Sprintf("Node{ID: %d, Label: %s, Children: %d}", t.ID, t.Label, len(t.Children))
}

// TreeConverter converts parser AST nodes to APTED tree nodes
type TreeConverter struct {
	nextID int
}

// NewTreeConverter creates a new tree converter
func NewTreeConverter() *TreeConverter {
	return &TreeConverter{nextID: 0}
}

// ConvertAST converts a parser AST node to an APTED tree
func (tc *TreeConverter) ConvertAST(astNode *parser.Node) *TreeNode {
	if astNode == nil {
		return nil
	}

	// Create tree node with simplified label
	label := tc.getNodeLabel(astNode)
	treeNode := NewTreeNode(tc.nextID, label)
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

// PostOrderTraversal performs post-order traversal and assigns post-order IDs
func PostOrderTraversal(root *TreeNode) {
	if root == nil {
		return
	}

	postOrderID := 0
	postOrderTraversalRecursive(root, &postOrderID)
}

// postOrderTraversalRecursive recursively performs post-order traversal
func postOrderTraversalRecursive(node *TreeNode, postOrderID *int) {
	if node == nil {
		return
	}

	// Visit children first
	for _, child := range node.Children {
		postOrderTraversalRecursive(child, postOrderID)
	}

	// Then visit this node
	node.PostOrderID = *postOrderID
	*postOrderID++
}

// ComputeLeftMostLeaves computes left-most leaf descendants for all nodes
func ComputeLeftMostLeaves(root *TreeNode) {
	if root == nil {
		return
	}
	computeLeftMostLeavesRecursive(root)
}

// computeLeftMostLeavesRecursive recursively computes left-most leaf descendants
func computeLeftMostLeavesRecursive(node *TreeNode) int {
	if node.IsLeaf() || len(node.Children) == 0 {
		node.LeftMostLeaf = node.PostOrderID
		return node.LeftMostLeaf
	}

	// Get left-most leaf from first child
	leftMostLeaf := computeLeftMostLeavesRecursive(node.Children[0])
	node.LeftMostLeaf = leftMostLeaf

	// Process remaining children
	for i := 1; i < len(node.Children); i++ {
		computeLeftMostLeavesRecursive(node.Children[i])
	}

	return leftMostLeaf
}

// ComputeKeyRoots identifies key roots for path decomposition
func ComputeKeyRoots(root *TreeNode) []int {
	if root == nil {
		return []int{}
	}

	keyRoots := []int{}
	visited := make(map[int]bool)

	computeKeyRootsRecursive(root, &keyRoots, visited)

	return keyRoots
}

// computeKeyRootsRecursive recursively identifies key roots
func computeKeyRootsRecursive(node *TreeNode, keyRoots *[]int, visited map[int]bool) {
	if node == nil {
		return
	}

	// A node is a key root if its left-most leaf hasn't been visited
	if !visited[node.LeftMostLeaf] {
		node.KeyRoot = true
		*keyRoots = append(*keyRoots, node.PostOrderID)
		visited[node.LeftMostLeaf] = true
	}

	// Process children
	for _, child := range node.Children {
		computeKeyRootsRecursive(child, keyRoots, visited)
	}
}

// PrepareTreeForAPTED prepares a tree for APTED algorithm by computing all necessary indices
func PrepareTreeForAPTED(root *TreeNode) []int {
	if root == nil {
		return []int{}
	}

	// Step 1: Assign post-order IDs
	PostOrderTraversal(root)

	// Step 2: Compute left-most leaf descendants
	ComputeLeftMostLeaves(root)

	// Step 3: Identify key roots
	keyRoots := ComputeKeyRoots(root)

	return keyRoots
}

// GetNodeByPostOrderID finds a node by its post-order ID
func GetNodeByPostOrderID(root *TreeNode, postOrderID int) *TreeNode {
	if root == nil {
		return nil
	}

	if root.PostOrderID == postOrderID {
		return root
	}

	for _, child := range root.Children {
		if node := GetNodeByPostOrderID(child, postOrderID); node != nil {
			return node
		}
	}

	return nil
}

// GetSubtreeNodes returns all nodes in the subtree rooted at the given node
func GetSubtreeNodes(root *TreeNode) []*TreeNode {
	return GetSubtreeNodesWithDepthLimit(root, 1000) // Default recursion limit
}

// GetSubtreeNodesWithDepthLimit returns all nodes with maximum recursion depth limit
func GetSubtreeNodesWithDepthLimit(root *TreeNode, maxDepth int) []*TreeNode {
	if root == nil || maxDepth <= 0 {
		return []*TreeNode{}
	}

	nodes := []*TreeNode{root}
	for _, child := range root.Children {
		nodes = append(nodes, GetSubtreeNodesWithDepthLimit(child, maxDepth-1)...)
	}

	return nodes
}
