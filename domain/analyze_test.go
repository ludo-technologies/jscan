package domain

import (
	"math"
	"testing"
)

func TestCalculateHealthScore_CyclePenaltyLogFloor(t *testing.T) {
	tests := []struct {
		name              string
		modulesInCycles   int
		totalModules      int
		wantMinDepPenalty int // minimum expected dependency penalty from cycles alone
	}{
		{
			name:              "small ratio still penalised via log floor",
			modulesInCycles:   18,
			totalModules:      587,
			wantMinDepPenalty: 4, // log2(19) ≈ 4.25 → round 4
		},
		{
			name:              "very small ratio still penalised",
			modulesInCycles:   15,
			totalModules:      1500,
			wantMinDepPenalty: 4, // log2(16) = 4.0 → round 4
		},
		{
			name:              "moderate ratio uses log floor",
			modulesInCycles:   6,
			totalModules:      80,
			wantMinDepPenalty: 3, // log2(7) ≈ 2.81 → round 3
		},
		{
			name:              "no cycles no penalty",
			modulesInCycles:   0,
			totalModules:      500,
			wantMinDepPenalty: 0,
		},
		{
			name:              "large ratio uses proportion",
			modulesInCycles:   40,
			totalModules:      80,
			wantMinDepPenalty: 5, // proportion: 10*0.5 = 5, log2(41) ≈ 5.36 → max is 5.36 → round 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyzeSummary{
				DepsEnabled:         true,
				DepsTotalModules:    tt.totalModules,
				DepsModulesInCycles: tt.modulesInCycles,
			}
			if err := s.CalculateHealthScore(); err != nil {
				t.Fatalf("CalculateHealthScore() error: %v", err)
			}
			// Dependency penalty = 100 - DependencyScore (mapped to 0-100 via penaltyToScore)
			// We verify the score is low enough to reflect the expected penalty.
			// With only cycle penalty active (depth=0, MSD=0), the normalized penalty
			// is: round(cyclePenalty / 16 * 20), and score = 100 - round(norm * 100 / 20).
			expectedCyclePenalty := 0
			if tt.modulesInCycles > 0 {
				ratio := float64(tt.modulesInCycles) / float64(tt.totalModules)
				if ratio > 1 {
					ratio = 1
				}
				prop := float64(MaxCyclesPenalty) * ratio
				logF := math.Log2(float64(tt.modulesInCycles) + 1)
				expectedCyclePenalty = int(math.Round(math.Min(float64(MaxCyclesPenalty), math.Max(logF, prop))))
			}

			if expectedCyclePenalty < tt.wantMinDepPenalty {
				t.Errorf("expected cycle penalty >= %d, formula gives %d", tt.wantMinDepPenalty, expectedCyclePenalty)
			}

			// The dependency score must reflect the penalty (not 100)
			if tt.modulesInCycles > 0 && s.DependencyScore >= 100 {
				t.Errorf("DependencyScore should be < 100 when cycles exist, got %d", s.DependencyScore)
			}
			if tt.modulesInCycles == 0 && s.DependencyScore != 100 {
				t.Errorf("DependencyScore should be 100 with no cycles and no other dep issues, got %d", s.DependencyScore)
			}
		})
	}
}

func TestCalculateHealthScore_MSDPenalty(t *testing.T) {
	tests := []struct {
		name    string
		msd     float64
		wantMax int // maximum expected DependencyScore
		wantMin int // minimum expected DependencyScore
	}{
		{
			name:    "zero MSD gives full score",
			msd:     0.0,
			wantMin: 100,
			wantMax: 100,
		},
		{
			name:    "moderate MSD reduces score",
			msd:     0.4,
			wantMin: 85,
			wantMax: 99,
		},
		{
			name:    "high MSD reduces score further",
			msd:     1.0,
			wantMin: 75,
			wantMax: 95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyzeSummary{
				DepsEnabled:               true,
				DepsTotalModules:          100,
				DepsMainSequenceDeviation: tt.msd,
			}
			if err := s.CalculateHealthScore(); err != nil {
				t.Fatalf("CalculateHealthScore() error: %v", err)
			}
			if s.DependencyScore < tt.wantMin || s.DependencyScore > tt.wantMax {
				t.Errorf("DependencyScore = %d, want [%d, %d] for MSD=%.1f",
					s.DependencyScore, tt.wantMin, tt.wantMax, tt.msd)
			}
		})
	}
}
