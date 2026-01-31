package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileHelperCollectJSFiles(t *testing.T) {
	// Create temp directory with test files
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{"test.js", "test.ts", "test.jsx", "test.tsx", "test.txt"}
	for _, f := range testFiles {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("// test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	helper := NewFileHelper()

	// Test collecting JS files
	files, err := helper.CollectJSFiles([]string{tempDir}, true, nil, nil)
	if err != nil {
		t.Fatalf("CollectJSFiles failed: %v", err)
	}

	// Should find 4 JS/TS files
	if len(files) != 4 {
		t.Errorf("Expected 4 JS/TS files, got %d", len(files))
	}
}

func TestFileHelperIsValidJSFile(t *testing.T) {
	helper := NewFileHelper()

	tests := []struct {
		path     string
		expected bool
	}{
		{"test.js", true},
		{"test.ts", true},
		{"test.jsx", true},
		{"test.tsx", true},
		{"test.mjs", true},
		{"test.cjs", true},
		{"test.mts", true},
		{"test.cts", true},
		{"test.py", false},
		{"test.go", false},
		{"test.txt", false},
	}

	for _, tt := range tests {
		result := helper.IsValidJSFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsValidJSFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestFileHelperFileExists(t *testing.T) {
	helper := NewFileHelper()

	// Create temp file
	tempFile, err := os.CreateTemp("", "test*.js")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Test existing file
	exists, err := helper.FileExists(tempFile.Name())
	if err != nil {
		t.Fatalf("FileExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected file to exist")
	}

	// Test non-existing file
	exists, err = helper.FileExists("/nonexistent/file.js")
	if err != nil {
		t.Fatalf("FileExists failed: %v", err)
	}
	if exists {
		t.Error("Expected file to not exist")
	}
}

func TestFileHelperIsExcluded(t *testing.T) {
	helper := NewFileHelper()

	tests := []struct {
		path            string
		excludePatterns []string
		expected        bool
	}{
		{"test.js", []string{"*.spec.js"}, false},
		{"test.spec.js", []string{"*.spec.js"}, true},
		{"test.test.js", []string{"*.test.js"}, true},
		{"node_modules/test.js", []string{"node_modules"}, true},
		{"src/test.js", []string{"node_modules"}, false},
	}

	for _, tt := range tests {
		result := helper.isExcluded(tt.path, tt.excludePatterns)
		if result != tt.expected {
			t.Errorf("isExcluded(%s, %v) = %v, expected %v", tt.path, tt.excludePatterns, result, tt.expected)
		}
	}
}

func TestResolveFilePaths(t *testing.T) {
	// Create temp directory with test files
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.js")
	if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	helper := NewFileHelper()

	// Test with existing file
	files, err := ResolveFilePaths(helper, []string{testFile}, true, nil, nil)
	if err != nil {
		t.Fatalf("ResolveFilePaths failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	// Test with directory
	files, err = ResolveFilePaths(helper, []string{tempDir}, true, nil, nil)
	if err != nil {
		t.Fatalf("ResolveFilePaths failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

func TestDefaultAnalyzeConfig(t *testing.T) {
	config := DefaultAnalyzeConfig()

	if !config.EnableComplexity {
		t.Error("Expected EnableComplexity to be true")
	}
	if !config.EnableDeadCode {
		t.Error("Expected EnableDeadCode to be true")
	}
	if config.LowThreshold != 9 {
		t.Errorf("Expected LowThreshold to be 9, got %d", config.LowThreshold)
	}
	if config.MediumThreshold != 19 {
		t.Errorf("Expected MediumThreshold to be 19, got %d", config.MediumThreshold)
	}
}

func TestDefaultUseCaseOptions(t *testing.T) {
	opts := DefaultUseCaseOptions()

	if !opts.EnableProgress {
		t.Error("Expected EnableProgress to be true")
	}
	if opts.MaxConcurrency != 4 {
		t.Errorf("Expected MaxConcurrency to be 4, got %d", opts.MaxConcurrency)
	}
}

func TestFileHelperExcludeNodeModules(t *testing.T) {
	// Create temp directory structure with node_modules
	tempDir := t.TempDir()

	// Create a source file
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	srcFile := filepath.Join(srcDir, "index.js")
	if err := os.WriteFile(srcFile, []byte("// source"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create node_modules directory with a JS file
	nodeModulesDir := filepath.Join(tempDir, "node_modules", "some-package")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatalf("Failed to create node_modules dir: %v", err)
	}
	nodeModulesFile := filepath.Join(nodeModulesDir, "index.js")
	if err := os.WriteFile(nodeModulesFile, []byte("// package"), 0644); err != nil {
		t.Fatalf("Failed to create node_modules file: %v", err)
	}

	helper := NewFileHelper()

	// Test with node_modules excluded
	excludePatterns := []string{"node_modules"}
	files, err := helper.CollectJSFiles([]string{tempDir}, true, nil, excludePatterns)
	if err != nil {
		t.Fatalf("CollectJSFiles failed: %v", err)
	}

	// Should only find 1 file (src/index.js), not the one in node_modules
	if len(files) != 1 {
		t.Errorf("Expected 1 file (excluding node_modules), got %d", len(files))
	}

	// Verify the found file is from src, not node_modules
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == "node_modules" || filepath.Base(filepath.Dir(filepath.Dir(f))) == "node_modules" {
			t.Errorf("Found file in node_modules which should be excluded: %s", f)
		}
	}
}

func TestFileHelperExcludeMultiplePatterns(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create various directories
	dirs := []string{"src", "dist", "build", ".next", "coverage"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create %s dir: %v", dir, err)
		}
		file := filepath.Join(dirPath, "index.js")
		if err := os.WriteFile(file, []byte("// "+dir), 0644); err != nil {
			t.Fatalf("Failed to create file in %s: %v", dir, err)
		}
	}

	helper := NewFileHelper()

	// Test with multiple exclusions
	excludePatterns := []string{"dist", "build", ".next", "coverage"}
	files, err := helper.CollectJSFiles([]string{tempDir}, true, nil, excludePatterns)
	if err != nil {
		t.Fatalf("CollectJSFiles failed: %v", err)
	}

	// Should only find 1 file (src/index.js)
	if len(files) != 1 {
		t.Errorf("Expected 1 file (only src), got %d", len(files))
	}
}

func TestFileHelperExcludeMinifiedFiles(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create various files
	testFiles := []string{"app.js", "utils.js", "vendor.min.js", "bundle.bundle.js"}
	for _, f := range testFiles {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("// "+f), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	helper := NewFileHelper()

	// Test with minified file exclusions
	excludePatterns := []string{"*.min.js", "*.bundle.js"}
	files, err := helper.CollectJSFiles([]string{tempDir}, true, nil, excludePatterns)
	if err != nil {
		t.Fatalf("CollectJSFiles failed: %v", err)
	}

	// Should find only app.js and utils.js
	if len(files) != 2 {
		t.Errorf("Expected 2 files (excluding minified/bundled), got %d", len(files))
	}
}

func TestFileHelperExcludeSourceMaps(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create various files including source maps
	testFiles := []string{
		"app.js",
		"app.js.map",      // Source map
		"utils.min.js",    // Minified
		"utils.min.js.map", // Minified source map
		"lib.mjs",
		"lib.min.mjs",     // Minified ESM
	}
	for _, f := range testFiles {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("// "+f), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	helper := NewFileHelper()

	// Test with source map and minified exclusions
	excludePatterns := []string{"*.map", "*.min.js", "*.min.mjs"}
	files, err := helper.CollectJSFiles([]string{tempDir}, true, nil, excludePatterns)
	if err != nil {
		t.Fatalf("CollectJSFiles failed: %v", err)
	}

	// Should find only app.js and lib.mjs
	if len(files) != 2 {
		t.Errorf("Expected 2 files (excluding maps/minified), got %d: %v", len(files), files)
	}
}

func TestFileHelperExcludeCacheDirectories(t *testing.T) {
	// Create temp directory structure with cache directories
	tempDir := t.TempDir()

	// Create various directories including cache dirs
	dirs := []string{"src", ".cache", ".turbo", ".vercel", ".output"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create %s dir: %v", dir, err)
		}
		file := filepath.Join(dirPath, "index.js")
		if err := os.WriteFile(file, []byte("// "+dir), 0644); err != nil {
			t.Fatalf("Failed to create file in %s: %v", dir, err)
		}
	}

	helper := NewFileHelper()

	// Test with cache directory exclusions
	excludePatterns := []string{".cache", ".turbo", ".vercel", ".output"}
	files, err := helper.CollectJSFiles([]string{tempDir}, true, nil, excludePatterns)
	if err != nil {
		t.Fatalf("CollectJSFiles failed: %v", err)
	}

	// Should only find 1 file (src/index.js)
	if len(files) != 1 {
		t.Errorf("Expected 1 file (only src), got %d", len(files))
	}
}
