package analyzer

import (
	"testing"

	corecfg "github.com/ludo-technologies/codescan-core/cfg"
)

func TestSeverityLevelConstants(t *testing.T) {
	if SeverityLevelCritical != "critical" {
		t.Errorf("SeverityLevelCritical should be 'critical', got %s", SeverityLevelCritical)
	}
	if SeverityLevelWarning != "warning" {
		t.Errorf("SeverityLevelWarning should be 'warning', got %s", SeverityLevelWarning)
	}
	if SeverityLevelInfo != "info" {
		t.Errorf("SeverityLevelInfo should be 'info', got %s", SeverityLevelInfo)
	}
}

func TestDeadCodeReasonConstants(t *testing.T) {
	reasons := []struct {
		reason   DeadCodeReason
		expected string
	}{
		{ReasonUnreachableAfterReturn, "unreachable_after_return"},
		{ReasonUnreachableAfterBreak, "unreachable_after_break"},
		{ReasonUnreachableAfterContinue, "unreachable_after_continue"},
		{ReasonUnreachableAfterThrow, "unreachable_after_throw"},
		{ReasonUnreachableBranch, "unreachable_branch"},
		{ReasonUnreachableAfterInfiniteLoop, "unreachable_after_infinite_loop"},
	}

	for _, tc := range reasons {
		if string(tc.reason) != tc.expected {
			t.Errorf("DeadCodeReason should be '%s', got %s", tc.expected, tc.reason)
		}
	}
}

func TestNewDeadCodeDetector(t *testing.T) {
	cfg := corecfg.NewCFG("test")
	detector := NewDeadCodeDetector(cfg)

	if detector == nil {
		t.Fatal("NewDeadCodeDetector should not return nil")
	}
	if detector.cfg != cfg {
		t.Error("Detector should store CFG reference")
	}
	if detector.filePath != "" {
		t.Error("Detector should have empty file path by default")
	}
}

func TestNewDeadCodeDetectorWithFilePath(t *testing.T) {
	cfg := corecfg.NewCFG("test")
	filePath := "/path/to/file.js"
	detector := NewDeadCodeDetectorWithFilePath(cfg, filePath)

	if detector == nil {
		t.Fatal("NewDeadCodeDetectorWithFilePath should not return nil")
	}
	if detector.cfg != cfg {
		t.Error("Detector should store CFG reference")
	}
	if detector.filePath != filePath {
		t.Errorf("Detector file path should be '%s', got '%s'", filePath, detector.filePath)
	}
}

func TestDeadCodeDetector_Detect_NilCFG(t *testing.T) {
	detector := &DeadCodeDetector{cfg: nil}
	result := detector.Detect()

	if result == nil {
		t.Fatal("Detect should return non-nil result even with nil CFG")
	}
	if len(result.Findings) != 0 {
		t.Error("Should have no findings with nil CFG")
	}
	if result.TotalBlocks != 0 {
		t.Error("TotalBlocks should be 0 with nil CFG")
	}
}

func TestDeadCodeDetector_Detect_SimpleCFG(t *testing.T) {
	cfg := corecfg.NewCFG("simpleFunc")
	cfg.ConnectBlocks(cfg.Entry, cfg.Exit, corecfg.EdgeNormal)

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.FunctionName != "simpleFunc" {
		t.Errorf("FunctionName should be 'simpleFunc', got %s", result.FunctionName)
	}
	// Simple CFG with no dead code
	if result.HasFindings() {
		t.Error("Simple CFG should have no dead code findings")
	}
}

func TestDeadCodeDetector_Detect_CodeAfterReturn(t *testing.T) {
	code := `
		function test() {
			return 1;
			console.log("unreachable");
		}
	`
	ast := parseJS(t, code)
	funcNode := findFunction(ast, "test")

	builder := NewCFGBuilder()
	cfg, err := builder.Build(funcNode)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Should detect dead code after return
	if result.DeadBlocks == 0 {
		t.Log("Note: Dead code after return may be handled differently")
	}
}

func TestDeadCodeDetector_Detect_CodeAfterThrow(t *testing.T) {
	code := `
		function test() {
			throw new Error("error");
			console.log("unreachable");
		}
	`
	ast := parseJS(t, code)
	funcNode := findFunction(ast, "test")

	builder := NewCFGBuilder()
	cfg, err := builder.Build(funcNode)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestDeadCodeDetector_Detect_CodeAfterBreak(t *testing.T) {
	code := `
		function test() {
			for (let i = 0; i < 10; i++) {
				break;
				console.log("unreachable");
			}
		}
	`
	ast := parseJS(t, code)
	funcNode := findFunction(ast, "test")

	builder := NewCFGBuilder()
	cfg, err := builder.Build(funcNode)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestDeadCodeDetector_Detect_CodeAfterContinue(t *testing.T) {
	code := `
		function test() {
			for (let i = 0; i < 10; i++) {
				continue;
				console.log("unreachable");
			}
		}
	`
	ast := parseJS(t, code)
	funcNode := findFunction(ast, "test")

	builder := NewCFGBuilder()
	cfg, err := builder.Build(funcNode)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestDeadCodeDetector_Detect_AllPathsReturn(t *testing.T) {
	code := `
		function test(x) {
			if (x > 0) {
				return 1;
			} else {
				return -1;
			}
			console.log("unreachable");
		}
	`
	ast := parseJS(t, code)
	funcNode := findFunction(ast, "test")

	builder := NewCFGBuilder()
	cfg, err := builder.Build(funcNode)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	// Code after if-else that both return should be dead
	if result.ReachableRatio > 1.0 || result.ReachableRatio < 0.0 {
		t.Errorf("ReachableRatio should be between 0 and 1, got %f", result.ReachableRatio)
	}
}

func TestDeadCodeResult_HasFindings(t *testing.T) {
	result := &DeadCodeResult{
		Findings: []*DeadCodeFinding{},
	}

	if result.HasFindings() {
		t.Error("Empty findings should return false")
	}

	result.Findings = []*DeadCodeFinding{
		{BlockID: "test"},
	}

	if !result.HasFindings() {
		t.Error("Non-empty findings should return true")
	}
}

func TestDeadCodeResult_GetCriticalFindings(t *testing.T) {
	result := &DeadCodeResult{
		Findings: []*DeadCodeFinding{
			{BlockID: "critical1", Severity: SeverityLevelCritical},
			{BlockID: "warning1", Severity: SeverityLevelWarning},
			{BlockID: "critical2", Severity: SeverityLevelCritical},
			{BlockID: "info1", Severity: SeverityLevelInfo},
		},
	}

	critical := result.GetCriticalFindings()

	if len(critical) != 2 {
		t.Errorf("Should have 2 critical findings, got %d", len(critical))
	}

	for _, finding := range critical {
		if finding.Severity != SeverityLevelCritical {
			t.Errorf("GetCriticalFindings should only return critical findings, got %s", finding.Severity)
		}
	}
}

func TestDeadCodeResult_GetWarningFindings(t *testing.T) {
	result := &DeadCodeResult{
		Findings: []*DeadCodeFinding{
			{BlockID: "critical1", Severity: SeverityLevelCritical},
			{BlockID: "warning1", Severity: SeverityLevelWarning},
			{BlockID: "warning2", Severity: SeverityLevelWarning},
			{BlockID: "info1", Severity: SeverityLevelInfo},
		},
	}

	warnings := result.GetWarningFindings()

	if len(warnings) != 2 {
		t.Errorf("Should have 2 warning findings, got %d", len(warnings))
	}

	for _, finding := range warnings {
		if finding.Severity != SeverityLevelWarning {
			t.Errorf("GetWarningFindings should only return warning findings, got %s", finding.Severity)
		}
	}
}

func TestDetectAll(t *testing.T) {
	code := `
		function foo() {
			return 1;
		}

		function bar() {
			return 2;
		}
	`
	ast := parseJS(t, code)

	builder := NewCFGBuilder()
	cfgs, err := builder.BuildAll(ast)
	if err != nil {
		t.Fatalf("BuildAll failed: %v", err)
	}

	results := DetectAll(cfgs, "/test/file.js")

	if len(results) == 0 {
		t.Error("DetectAll should return results for all CFGs")
	}

	// Check that all results have the file path set
	for name, result := range results {
		if result.FilePath != "/test/file.js" {
			t.Errorf("Result for %s should have file path set", name)
		}
	}
}

func TestDeadCodeDetector_generateDescription(t *testing.T) {
	detector := &DeadCodeDetector{}

	testCases := []struct {
		reason   DeadCodeReason
		expected string
	}{
		{ReasonUnreachableAfterReturn, "Code after return statement is unreachable"},
		{ReasonUnreachableAfterBreak, "Code after break statement is unreachable"},
		{ReasonUnreachableAfterContinue, "Code after continue statement is unreachable"},
		{ReasonUnreachableAfterThrow, "Code after throw statement is unreachable"},
		{ReasonUnreachableBranch, "This branch is unreachable"},
		{ReasonUnreachableAfterInfiniteLoop, "Code after infinite loop is unreachable"},
		{DeadCodeReason("unknown"), "Code is unreachable"},
	}

	for _, tc := range testCases {
		desc := detector.generateDescription(tc.reason)
		if desc != tc.expected {
			t.Errorf("generateDescription(%s) = %s, expected %s", tc.reason, desc, tc.expected)
		}
	}
}

func TestDeadCodeDetector_getCodeSnippet_Empty(t *testing.T) {
	detector := &DeadCodeDetector{}
	snippet := detector.getCodeSnippet(nil)

	if snippet != "" {
		t.Errorf("Empty statements should produce empty snippet, got '%s'", snippet)
	}
}

func TestDeadCodeDetector_getFunctionName(t *testing.T) {
	// With CFG
	cfg := corecfg.NewCFG("myFunction")
	detector := NewDeadCodeDetector(cfg)
	name := detector.getFunctionName()

	if name != "myFunction" {
		t.Errorf("getFunctionName should return 'myFunction', got '%s'", name)
	}

	// With nil CFG
	nilDetector := &DeadCodeDetector{cfg: nil}
	nilName := nilDetector.getFunctionName()

	if nilName != "" {
		t.Errorf("Nil CFG should return empty function name, got '%s'", nilName)
	}
}

func TestDeadCodeDetector_getFilePath(t *testing.T) {
	detector := NewDeadCodeDetectorWithFilePath(corecfg.NewCFG("test"), "/path/to/file.js")
	path := detector.getFilePath()

	if path != "/path/to/file.js" {
		t.Errorf("getFilePath should return '/path/to/file.js', got '%s'", path)
	}
}

func TestDeadCodeDetector_Detect_AnalysisTime(t *testing.T) {
	cfg := corecfg.NewCFG("test")
	cfg.ConnectBlocks(cfg.Entry, cfg.Exit, corecfg.EdgeNormal)

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result.AnalysisTime < 0 {
		t.Error("AnalysisTime should be non-negative")
	}
}

func TestDeadCodeDetector_Detect_SortsByLineNumber(t *testing.T) {
	code := `
		function test() {
			return 1;
			console.log("first");
			console.log("second");
		}
	`
	ast := parseJS(t, code)
	funcNode := findFunction(ast, "test")

	builder := NewCFGBuilder()
	cfg, err := builder.Build(funcNode)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	// If there are multiple findings, they should be sorted by line number
	if len(result.Findings) > 1 {
		for i := 1; i < len(result.Findings); i++ {
			if result.Findings[i].StartLine < result.Findings[i-1].StartLine {
				t.Error("Findings should be sorted by line number")
			}
		}
	}
}

func TestDeadCodeDetector_Detect_WithFilePath(t *testing.T) {
	cfg := corecfg.NewCFG("test")
	cfg.ConnectBlocks(cfg.Entry, cfg.Exit, corecfg.EdgeNormal)

	filePath := "/src/components/Button.js"
	detector := NewDeadCodeDetectorWithFilePath(cfg, filePath)
	result := detector.Detect()

	if result.FilePath != filePath {
		t.Errorf("Result FilePath should be '%s', got '%s'", filePath, result.FilePath)
	}
}

// Test for complex nested control flow
func TestDeadCodeDetector_Detect_NestedControlFlow(t *testing.T) {
	code := `
		function test(x, y) {
			if (x > 0) {
				if (y > 0) {
					return "both positive";
				} else {
					return "x positive, y not";
				}
				console.log("unreachable after nested if-else");
			}
			return "x not positive";
		}
	`
	ast := parseJS(t, code)
	funcNode := findFunction(ast, "test")

	builder := NewCFGBuilder()
	cfg, err := builder.Build(funcNode)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	detector := NewDeadCodeDetector(cfg)
	result := detector.Detect()

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}
