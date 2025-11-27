package parser

import (
	"os"
	"testing"
)

func TestParseSimpleFunction(t *testing.T) {
	code := `function hello() { return 42; }`

	parser := NewParser()
	defer parser.Close()

	ast, err := parser.ParseString(code)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ast == nil {
		t.Fatal("AST is nil")
	}

	if ast.Type != NodeProgram {
		t.Errorf("Expected NodeProgram, got %s", ast.Type)
	}

	if len(ast.Body) == 0 {
		t.Fatal("Expected at least one statement in body")
	}

	// Check if first statement is a function
	funcNode := ast.Body[0]
	if funcNode.Type != NodeFunction {
		t.Errorf("Expected NodeFunction, got %s", funcNode.Type)
	}

	if funcNode.Name != "hello" {
		t.Errorf("Expected function name 'hello', got '%s'", funcNode.Name)
	}
}

func TestParseIfStatement(t *testing.T) {
	code := `
	function greet(name) {
		if (name) {
			return "Hello, " + name;
		} else {
			return "Hello, stranger";
		}
	}
	`

	parser := NewParser()
	defer parser.Close()

	ast, err := parser.ParseString(code)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ast == nil || len(ast.Body) == 0 {
		t.Fatal("AST is nil or empty")
	}

	funcNode := ast.Body[0]
	if funcNode.Name != "greet" {
		t.Errorf("Expected function name 'greet', got '%s'", funcNode.Name)
	}

	// Check if function has body with if statement
	if len(funcNode.Body) == 0 {
		t.Fatal("Function body is empty")
	}

	// Find if statement in function body
	found := false
	funcNode.Walk(func(n *Node) bool {
		if n.Type == NodeIfStatement {
			found = true
			return false
		}
		return true
	})

	if !found {
		t.Error("Expected to find if statement in function body")
	}
}

func TestParseArrowFunction(t *testing.T) {
	code := `const add = (a, b) => { return a + b; };`

	parser := NewParser()
	defer parser.Close()

	ast, err := parser.ParseString(code)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find arrow function in AST
	found := false
	ast.Walk(func(n *Node) bool {
		if n.Type == NodeArrowFunction {
			found = true
			if len(n.Params) != 2 {
				t.Errorf("Expected 2 parameters, got %d", len(n.Params))
			}
			return false
		}
		return true
	})

	if !found {
		t.Error("Expected to find arrow function")
	}
}

func TestParseFile(t *testing.T) {
	// Read test file
	content, err := os.ReadFile("../../testdata/javascript/simple/function.js")
	if err != nil {
		t.Skipf("Skipping file test: %v", err)
		return
	}

	parser := NewParser()
	defer parser.Close()

	ast, err := parser.ParseFile("function.js", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ast == nil {
		t.Fatal("AST is nil")
	}

	// Count functions in the file
	functionCount := 0
	ast.Walk(func(n *Node) bool {
		if n.IsFunction() {
			functionCount++
		}
		return true
	})

	if functionCount < 3 {
		t.Errorf("Expected at least 3 functions, found %d", functionCount)
	}
}

func TestParseForLoop(t *testing.T) {
	code := `
	for (let i = 0; i < 10; i++) {
		console.log(i);
	}
	`

	parser := NewParser()
	defer parser.Close()

	ast, err := parser.ParseString(code)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	found := false
	ast.Walk(func(n *Node) bool {
		if n.Type == NodeForStatement {
			found = true
			if n.Init == nil {
				t.Error("Expected for loop to have init")
			}
			if n.Test == nil {
				t.Error("Expected for loop to have test")
			}
			if n.Update == nil {
				t.Error("Expected for loop to have update")
			}
			return false
		}
		return true
	})

	if !found {
		t.Error("Expected to find for statement")
	}
}

func TestParseTryCatch(t *testing.T) {
	code := `
	try {
		throw new Error("oops");
	} catch (e) {
		console.error(e);
	} finally {
		cleanup();
	}
	`

	parser := NewParser()
	defer parser.Close()

	ast, err := parser.ParseString(code)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	found := false
	ast.Walk(func(n *Node) bool {
		if n.Type == NodeTryStatement {
			found = true
			if n.Handler == nil {
				t.Error("Expected try statement to have handler (catch)")
			}
			if n.Finalizer == nil {
				t.Error("Expected try statement to have finalizer (finally)")
			}
			return false
		}
		return true
	})

	if !found {
		t.Error("Expected to find try statement")
	}
}
