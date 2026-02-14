package analyzer

import (
	"testing"

	"github.com/ludo-technologies/codescan-core/apted"
)

func TestJavaScriptCostModel(t *testing.T) {
	costModel := NewJavaScriptCostModel()

	// Test structural node (higher cost)
	funcNode := apted.NewTreeNode(1, "FunctionDeclaration")
	cost := costModel.Insert(funcNode)
	if cost <= 1.0 {
		t.Error("Structural nodes should have higher cost")
	}

	// Test control flow node
	ifNode := apted.NewTreeNode(2, "IfStatement")
	cost = costModel.Insert(ifNode)
	if cost <= 1.0 {
		t.Error("Control flow nodes should have higher cost")
	}

	// Test expression node (lower cost)
	exprNode := apted.NewTreeNode(3, "BinaryExpression")
	cost = costModel.Insert(exprNode)
	if cost >= 1.0 {
		t.Error("Expression nodes should have lower cost")
	}

	// Test rename with same base type
	node1 := apted.NewTreeNode(4, "Identifier(foo)")
	node2 := apted.NewTreeNode(5, "Identifier(bar)")
	renameCost := costModel.Rename(node1, node2)
	if renameCost >= 1.0 {
		t.Error("Rename cost for same base type should be reduced")
	}
}

func TestJavaScriptCostModelIgnore(t *testing.T) {
	// Test with ignore literals
	costModel := NewJavaScriptCostModelWithConfig(true, false)

	lit1 := apted.NewTreeNode(1, "Literal(42)")
	lit2 := apted.NewTreeNode(2, "Literal(100)")
	cost := costModel.Rename(lit1, lit2)
	if cost != 0.0 {
		t.Error("Literal differences should be ignored when configured")
	}

	// Test with ignore identifiers
	costModel2 := NewJavaScriptCostModelWithConfig(false, true)

	id1 := apted.NewTreeNode(3, "Identifier(foo)")
	id2 := apted.NewTreeNode(4, "Identifier(bar)")
	cost = costModel2.Rename(id1, id2)
	if cost != 0.0 {
		t.Error("Identifier differences should be ignored when configured")
	}
}

func TestJavaScriptCostModelImplementsCostModel(t *testing.T) {
	// Verify JavaScriptCostModel implements apted.CostModel
	var _ apted.CostModel = NewJavaScriptCostModel()
	var _ apted.CostModel = NewJavaScriptCostModelWithConfig(true, true)
}

func TestJavaScriptCostModelNil(t *testing.T) {
	costModel := NewJavaScriptCostModel()

	if costModel.Insert(nil) != 1.0 {
		t.Error("Insert cost for nil should be base cost 1.0")
	}
	if costModel.Delete(nil) != 1.0 {
		t.Error("Delete cost for nil should be base cost 1.0")
	}
	if costModel.Rename(nil, nil) != 1.0 {
		t.Error("Rename cost for nil should be base cost 1.0")
	}
}

func TestJavaScriptCostModelSameLabel(t *testing.T) {
	costModel := NewJavaScriptCostModel()

	node1 := apted.NewTreeNode(1, "IfStatement")
	node2 := apted.NewTreeNode(2, "IfStatement")
	if costModel.Rename(node1, node2) != 0.0 {
		t.Error("Rename cost for same labels should be 0.0")
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

func TestJavaScriptCostModelWithAPTEDAnalyzer(t *testing.T) {
	// Integration test: JavaScriptCostModel works with apted.APTEDAnalyzer
	costModel := NewJavaScriptCostModel()
	analyzer := apted.NewAPTEDAnalyzerWithNormalization(costModel, apted.NormalizeBySum)

	tree1 := apted.NewTreeNode(1, "FunctionDeclaration")
	tree1.AddChild(apted.NewTreeNode(2, "Identifier(foo)"))

	tree2 := apted.NewTreeNode(1, "FunctionDeclaration")
	tree2.AddChild(apted.NewTreeNode(2, "Identifier(bar)"))

	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance < 0 {
		t.Error("Distance should be non-negative")
	}

	similarity := analyzer.ComputeSimilarity(tree1, tree2)
	if similarity < 0 || similarity > 1 {
		t.Errorf("Similarity should be between 0 and 1, got %f", similarity)
	}

	// These trees differ only in identifier names, so they should be fairly similar
	if similarity < 0.3 {
		t.Errorf("Trees with same structure should have reasonable similarity, got %f", similarity)
	}
}

func TestJavaScriptCostModelRelatedTypes(t *testing.T) {
	costModel := NewJavaScriptCostModel()

	// Related types should have lower rename cost
	funcDecl := apted.NewTreeNode(1, "FunctionDeclaration")
	funcExpr := apted.NewTreeNode(2, "FunctionExpression")
	cost := costModel.Rename(funcDecl, funcExpr)
	if cost >= 1.0 {
		t.Errorf("Related types should have reduced rename cost, got %f", cost)
	}

	// Unrelated types should have full rename cost
	ifStmt := apted.NewTreeNode(3, "IfStatement")
	callExpr := apted.NewTreeNode(4, "CallExpression")
	cost = costModel.Rename(ifStmt, callExpr)
	if cost < 0.5 {
		t.Errorf("Unrelated types should have higher rename cost, got %f", cost)
	}
}
