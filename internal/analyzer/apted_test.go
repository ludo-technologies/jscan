package analyzer

import (
	"math"
	"testing"
)

func TestNewTreeNode(t *testing.T) {
	node := NewTreeNode(1, "Test")
	if node.ID != 1 {
		t.Errorf("Expected ID 1, got %d", node.ID)
	}
	if node.Label != "Test" {
		t.Errorf("Expected label 'Test', got %s", node.Label)
	}
	if len(node.Children) != 0 {
		t.Errorf("Expected empty children, got %d", len(node.Children))
	}
}

func TestTreeNodeAddChild(t *testing.T) {
	parent := NewTreeNode(1, "Parent")
	child := NewTreeNode(2, "Child")

	parent.AddChild(child)

	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}
	if child.Parent != parent {
		t.Error("Child's parent should be set")
	}
}

func TestTreeNodeIsLeaf(t *testing.T) {
	leaf := NewTreeNode(1, "Leaf")
	if !leaf.IsLeaf() {
		t.Error("Node without children should be a leaf")
	}

	parent := NewTreeNode(2, "Parent")
	child := NewTreeNode(3, "Child")
	parent.AddChild(child)
	if parent.IsLeaf() {
		t.Error("Node with children should not be a leaf")
	}
}

func TestTreeNodeSize(t *testing.T) {
	// Single node
	single := NewTreeNode(1, "Single")
	if single.Size() != 1 {
		t.Errorf("Single node size should be 1, got %d", single.Size())
	}

	// Tree with children
	root := NewTreeNode(1, "Root")
	child1 := NewTreeNode(2, "Child1")
	child2 := NewTreeNode(3, "Child2")
	grandchild := NewTreeNode(4, "Grandchild")

	root.AddChild(child1)
	root.AddChild(child2)
	child1.AddChild(grandchild)

	if root.Size() != 4 {
		t.Errorf("Tree size should be 4, got %d", root.Size())
	}
}

func TestTreeNodeHeight(t *testing.T) {
	// Single node
	single := NewTreeNode(1, "Single")
	if single.Height() != 0 {
		t.Errorf("Single node height should be 0, got %d", single.Height())
	}

	// Tree with depth
	root := NewTreeNode(1, "Root")
	child := NewTreeNode(2, "Child")
	grandchild := NewTreeNode(3, "Grandchild")

	root.AddChild(child)
	child.AddChild(grandchild)

	if root.Height() != 2 {
		t.Errorf("Tree height should be 2, got %d", root.Height())
	}
}

func TestDefaultCostModel(t *testing.T) {
	costModel := NewDefaultCostModel()
	node := NewTreeNode(1, "Test")

	if costModel.Insert(node) != 1.0 {
		t.Error("Insert cost should be 1.0")
	}
	if costModel.Delete(node) != 1.0 {
		t.Error("Delete cost should be 1.0")
	}

	node2 := NewTreeNode(2, "Test")
	if costModel.Rename(node, node2) != 0.0 {
		t.Error("Rename cost for same labels should be 0.0")
	}

	node3 := NewTreeNode(3, "Different")
	if costModel.Rename(node, node3) != 1.0 {
		t.Error("Rename cost for different labels should be 1.0")
	}
}

func TestJavaScriptCostModel(t *testing.T) {
	costModel := NewJavaScriptCostModel()

	// Test structural node (higher cost)
	funcNode := NewTreeNode(1, "FunctionDeclaration")
	cost := costModel.Insert(funcNode)
	if cost <= 1.0 {
		t.Error("Structural nodes should have higher cost")
	}

	// Test control flow node
	ifNode := NewTreeNode(2, "IfStatement")
	cost = costModel.Insert(ifNode)
	if cost <= 1.0 {
		t.Error("Control flow nodes should have higher cost")
	}

	// Test expression node (lower cost)
	exprNode := NewTreeNode(3, "BinaryExpression")
	cost = costModel.Insert(exprNode)
	if cost >= 1.0 {
		t.Error("Expression nodes should have lower cost")
	}

	// Test rename with same base type
	node1 := NewTreeNode(4, "Identifier(foo)")
	node2 := NewTreeNode(5, "Identifier(bar)")
	renameCost := costModel.Rename(node1, node2)
	if renameCost >= 1.0 {
		t.Error("Rename cost for same base type should be reduced")
	}
}

func TestJavaScriptCostModelIgnore(t *testing.T) {
	// Test with ignore literals
	costModel := NewJavaScriptCostModelWithConfig(true, false)

	lit1 := NewTreeNode(1, "Literal(42)")
	lit2 := NewTreeNode(2, "Literal(100)")
	cost := costModel.Rename(lit1, lit2)
	if cost != 0.0 {
		t.Error("Literal differences should be ignored when configured")
	}

	// Test with ignore identifiers
	costModel2 := NewJavaScriptCostModelWithConfig(false, true)

	id1 := NewTreeNode(3, "Identifier(foo)")
	id2 := NewTreeNode(4, "Identifier(bar)")
	cost = costModel2.Rename(id1, id2)
	if cost != 0.0 {
		t.Error("Identifier differences should be ignored when configured")
	}
}

func TestPrepareTreeForAPTED(t *testing.T) {
	// Build a simple tree
	root := NewTreeNode(1, "Root")
	child1 := NewTreeNode(2, "Child1")
	child2 := NewTreeNode(3, "Child2")
	root.AddChild(child1)
	root.AddChild(child2)

	keyRoots := PrepareTreeForAPTED(root)

	// Verify post-order IDs were assigned
	if child1.PostOrderID >= child2.PostOrderID {
		t.Error("Post-order IDs should be in correct order")
	}

	// Verify key roots were identified
	if len(keyRoots) == 0 {
		t.Error("Key roots should be identified")
	}
}

func TestAPTEDAnalyzerIdenticalTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create identical trees
	tree1 := NewTreeNode(1, "Root")
	tree1.AddChild(NewTreeNode(2, "Child"))

	tree2 := NewTreeNode(1, "Root")
	tree2.AddChild(NewTreeNode(2, "Child"))

	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance != 0.0 {
		t.Errorf("Distance between identical trees should be 0, got %f", distance)
	}

	similarity := analyzer.ComputeSimilarity(tree1, tree2)
	if similarity != 1.0 {
		t.Errorf("Similarity between identical trees should be 1.0, got %f", similarity)
	}
}

func TestAPTEDAnalyzerDifferentTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create different trees
	tree1 := NewTreeNode(1, "A")
	tree1.AddChild(NewTreeNode(2, "B"))

	tree2 := NewTreeNode(1, "X")
	tree2.AddChild(NewTreeNode(2, "Y"))
	tree2.AddChild(NewTreeNode(3, "Z"))

	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance <= 0.0 {
		t.Error("Distance between different trees should be positive")
	}

	similarity := analyzer.ComputeSimilarity(tree1, tree2)
	if similarity >= 1.0 || similarity < 0.0 {
		t.Errorf("Similarity should be between 0 and 1, got %f", similarity)
	}
}

func TestAPTEDAnalyzerNilTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Both nil
	distance := analyzer.ComputeDistance(nil, nil)
	if distance != 0.0 {
		t.Error("Distance between two nil trees should be 0")
	}

	// One nil
	tree := NewTreeNode(1, "Root")
	tree.AddChild(NewTreeNode(2, "Child"))

	distance = analyzer.ComputeDistance(tree, nil)
	if distance != 2.0 {
		t.Errorf("Distance should be tree size (2), got %f", distance)
	}

	distance = analyzer.ComputeDistance(nil, tree)
	if distance != 2.0 {
		t.Errorf("Distance should be tree size (2), got %f", distance)
	}
}

func TestAPTEDAnalyzerComputeDetailedDistance(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	tree1 := NewTreeNode(1, "A")
	tree1.AddChild(NewTreeNode(2, "B"))

	tree2 := NewTreeNode(1, "A")
	tree2.AddChild(NewTreeNode(2, "C"))

	result := analyzer.ComputeDetailedDistance(tree1, tree2)

	if result.Tree1Size != 2 {
		t.Errorf("Expected tree1 size 2, got %d", result.Tree1Size)
	}
	if result.Tree2Size != 2 {
		t.Errorf("Expected tree2 size 2, got %d", result.Tree2Size)
	}
	if result.Distance < 0 {
		t.Error("Distance should be non-negative")
	}
	if result.Similarity < 0 || result.Similarity > 1 {
		t.Error("Similarity should be between 0 and 1")
	}
}

func TestOptimizedAPTEDAnalyzer(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewOptimizedAPTEDAnalyzer(costModel, 5.0)

	tree1 := NewTreeNode(1, "Root")
	tree2 := NewTreeNode(1, "Root")

	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance != 0.0 {
		t.Errorf("Distance between identical trees should be 0, got %f", distance)
	}
}

func TestBatchComputeDistances(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	tree1 := NewTreeNode(1, "A")
	tree2 := NewTreeNode(2, "A")
	tree3 := NewTreeNode(3, "B")

	pairs := [][2]*TreeNode{
		{tree1, tree2},
		{tree1, tree3},
		{tree2, tree3},
	}

	distances := analyzer.BatchComputeDistances(pairs)

	if len(distances) != 3 {
		t.Errorf("Expected 3 distances, got %d", len(distances))
	}

	// tree1 and tree2 have same label
	if distances[0] != 0.0 {
		t.Errorf("Distance between identical label trees should be 0, got %f", distances[0])
	}

	// tree1 and tree3 have different labels
	if distances[1] == 0.0 {
		t.Error("Distance between different label trees should not be 0")
	}
}

func TestClusterSimilarTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create similar trees
	tree1 := NewTreeNode(1, "A")
	tree1.AddChild(NewTreeNode(2, "B"))

	tree2 := NewTreeNode(1, "A")
	tree2.AddChild(NewTreeNode(2, "B"))

	// Create different tree
	tree3 := NewTreeNode(1, "X")
	tree3.AddChild(NewTreeNode(2, "Y"))
	tree3.AddChild(NewTreeNode(3, "Z"))

	trees := []*TreeNode{tree1, tree2, tree3}
	result := analyzer.ClusterSimilarTrees(trees, 0.8)

	if len(result.Groups) == 0 {
		t.Error("Should have at least one group")
	}
}

func TestClusterEmptyTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Empty slice
	result := analyzer.ClusterSimilarTrees([]*TreeNode{}, 0.8)
	if len(result.Groups) != 0 {
		t.Error("Empty input should produce empty groups")
	}

	// Single tree
	tree := NewTreeNode(1, "A")
	result = analyzer.ClusterSimilarTrees([]*TreeNode{tree}, 0.8)
	if len(result.Groups) != 1 {
		t.Error("Single tree should produce one group")
	}
}

func TestWeightedCostModel(t *testing.T) {
	baseCost := NewDefaultCostModel()
	weighted := NewWeightedCostModel(2.0, 0.5, 1.5, baseCost)

	node := NewTreeNode(1, "Test")

	if weighted.Insert(node) != 2.0 {
		t.Errorf("Weighted insert cost should be 2.0, got %f", weighted.Insert(node))
	}
	if weighted.Delete(node) != 0.5 {
		t.Errorf("Weighted delete cost should be 0.5, got %f", weighted.Delete(node))
	}

	node2 := NewTreeNode(2, "Different")
	if weighted.Rename(node, node2) != 1.5 {
		t.Errorf("Weighted rename cost should be 1.5, got %f", weighted.Rename(node, node2))
	}
}

func TestTreeConverterConvertAST(t *testing.T) {
	// This test verifies the converter works with nil input
	converter := NewTreeConverter()

	result := converter.ConvertAST(nil)
	if result != nil {
		t.Error("Converting nil AST should return nil")
	}
}

func TestSimilarityBounds(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create various tree pairs
	testCases := []struct {
		name  string
		tree1 *TreeNode
		tree2 *TreeNode
	}{
		{"identical", createTestTree(3), createTestTree(3)},
		{"different_size", createTestTree(2), createTestTree(5)},
		{"completely_different", createDifferentTree(3), createDifferentTree(3)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sim := analyzer.ComputeSimilarity(tc.tree1, tc.tree2)
			if sim < 0.0 || sim > 1.0 {
				t.Errorf("Similarity must be in [0, 1], got %f", sim)
			}
			if math.IsNaN(sim) || math.IsInf(sim, 0) {
				t.Errorf("Similarity must be a valid number, got %f", sim)
			}
		})
	}
}

// Helper functions for creating test trees
func createTestTree(size int) *TreeNode {
	root := NewTreeNode(0, "Root")
	for i := 1; i < size; i++ {
		root.AddChild(NewTreeNode(i, "Child"))
	}
	return root
}

func createDifferentTree(size int) *TreeNode {
	root := NewTreeNode(0, "Different")
	for i := 1; i < size; i++ {
		root.AddChild(NewTreeNode(i, "Other"))
	}
	return root
}
