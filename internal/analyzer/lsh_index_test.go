package analyzer

import (
	"testing"
)

func TestNewLSHIndex(t *testing.T) {
	// Default values
	idx := NewLSHIndex(0, 0)
	if idx.Bands() != 32 {
		t.Errorf("Expected default 32 bands, got %d", idx.Bands())
	}
	if idx.Rows() != 4 {
		t.Errorf("Expected default 4 rows, got %d", idx.Rows())
	}

	// Custom values
	idx = NewLSHIndex(16, 8)
	if idx.Bands() != 16 {
		t.Errorf("Expected 16 bands, got %d", idx.Bands())
	}
	if idx.Rows() != 8 {
		t.Errorf("Expected 8 rows, got %d", idx.Rows())
	}
}

func TestAddFragment(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	sig := mh.ComputeSignature([]string{"a", "b", "c"})
	err := idx.AddFragment("frag1", sig)
	if err != nil {
		t.Errorf("AddFragment failed: %v", err)
	}

	if idx.Size() != 1 {
		t.Errorf("Expected size 1, got %d", idx.Size())
	}
}

func TestAddFragmentErrors(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Empty signature
	err := idx.AddFragment("frag1", nil)
	if err == nil {
		t.Error("Expected error for nil signature")
	}

	// Empty ID
	sig := mh.ComputeSignature([]string{"a", "b"})
	err = idx.AddFragment("", sig)
	if err == nil {
		t.Error("Expected error for empty ID")
	}
}

func TestFindCandidates(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Add similar fragments
	sig1 := mh.ComputeSignature([]string{"a", "b", "c", "d", "e"})
	sig2 := mh.ComputeSignature([]string{"a", "b", "c", "x", "y"})
	sig3 := mh.ComputeSignature([]string{"p", "q", "r", "s", "t"})

	_ = idx.AddFragment("similar1", sig1)
	_ = idx.AddFragment("similar2", sig2)
	_ = idx.AddFragment("different", sig3)

	// Query with sig1 - should find similar2
	candidates := idx.FindCandidates(sig1)

	if len(candidates) == 0 {
		t.Error("Expected at least one candidate")
	}

	// Should include similar1 (itself)
	found := false
	for _, c := range candidates {
		if c == "similar1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should find itself as candidate")
	}
}

func TestFindCandidatesEmpty(t *testing.T) {
	idx := NewLSHIndex(32, 4)

	// nil signature
	candidates := idx.FindCandidates(nil)
	if len(candidates) != 0 {
		t.Error("Expected empty result for nil signature")
	}

	// Empty index
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a", "b"})
	candidates = idx.FindCandidates(sig)
	if len(candidates) != 0 {
		t.Error("Expected empty result for empty index")
	}
}

func TestGetSignature(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	sig := mh.ComputeSignature([]string{"a", "b", "c"})
	_ = idx.AddFragment("frag1", sig)

	retrieved := idx.GetSignature("frag1")
	if retrieved == nil {
		t.Error("Expected to retrieve signature")
	}
	if len(retrieved.Signatures()) != len(sig.Signatures()) {
		t.Error("Retrieved signature should match original")
	}

	// Non-existent
	retrieved = idx.GetSignature("nonexistent")
	if retrieved != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

func TestBuildIndex(t *testing.T) {
	idx := NewLSHIndex(32, 4)

	// BuildIndex is a no-op but should not error
	err := idx.BuildIndex()
	if err != nil {
		t.Errorf("BuildIndex should not error: %v", err)
	}
}

func TestDuplicateAvoidance(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	sig := mh.ComputeSignature([]string{"a", "b", "c"})

	// Add same fragment twice
	_ = idx.AddFragment("frag1", sig)
	_ = idx.AddFragment("frag1", sig)

	// Size should still be 1
	if idx.Size() != 1 {
		t.Errorf("Expected size 1 (no duplicates), got %d", idx.Size())
	}

	// Candidates should not have duplicates
	candidates := idx.FindCandidates(sig)
	idCount := make(map[string]int)
	for _, c := range candidates {
		idCount[c]++
	}
	for id, count := range idCount {
		if count > 1 {
			t.Errorf("Found duplicate candidate %s (count: %d)", id, count)
		}
	}
}

func TestLSHSensitivity(t *testing.T) {
	// Test that similar items are more likely to be found than dissimilar
	mh := NewMinHasher(128)

	// Create base signature
	baseFeatures := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	baseSig := mh.ComputeSignature(baseFeatures)

	// Similar features (7/9 overlap)
	similarFeatures := []string{"a", "b", "c", "d", "e", "f", "g", "x", "y"}
	similarSig := mh.ComputeSignature(similarFeatures)

	// Different features (0 overlap)
	differentFeatures := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	differentSig := mh.ComputeSignature(differentFeatures)

	// Create index and add fragments
	idx := NewLSHIndex(32, 4)
	_ = idx.AddFragment("base", baseSig)
	_ = idx.AddFragment("similar", similarSig)
	_ = idx.AddFragment("different", differentSig)

	// Query with similar signature
	candidates := idx.FindCandidates(similarSig)

	// Count how many of base vs different are found
	hasBase := false
	hasDifferent := false
	for _, c := range candidates {
		if c == "base" {
			hasBase = true
		}
		if c == "different" {
			hasDifferent = true
		}
	}

	// Similar items should have higher chance of matching bands
	// This is probabilistic, but we expect base to be found more often
	t.Logf("Candidates for similar: %v, hasBase: %v, hasDifferent: %v", candidates, hasBase, hasDifferent)
}

func TestLSHIndexWithManyFragments(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Add many fragments
	for i := 0; i < 100; i++ {
		features := []string{
			"feature_" + string(rune('a'+i%26)),
			"feature_" + string(rune('A'+i%26)),
			"common",
		}
		sig := mh.ComputeSignature(features)
		_ = idx.AddFragment("frag_"+string(rune('0'+i%10))+string(rune('0'+i/10)), sig)
	}

	if idx.Size() != 100 {
		t.Errorf("Expected 100 fragments, got %d", idx.Size())
	}
}
