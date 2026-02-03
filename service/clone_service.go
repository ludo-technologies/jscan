package service

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/analyzer"
	"github.com/ludo-technologies/jscan/internal/parser"
	"github.com/ludo-technologies/jscan/internal/version"
)

// CloneServiceImpl implements the domain.CloneService interface
type CloneServiceImpl struct {
	config *analyzer.CloneDetectorConfig
}

// NewCloneService creates a new clone detection service
func NewCloneService(config *analyzer.CloneDetectorConfig) *CloneServiceImpl {
	return &CloneServiceImpl{
		config: config,
	}
}

// NewCloneServiceWithDefaults creates a service with default configuration
func NewCloneServiceWithDefaults() *CloneServiceImpl {
	return &CloneServiceImpl{
		config: analyzer.DefaultCloneDetectorConfig(),
	}
}

// DetectClones performs clone detection on the given request
func (s *CloneServiceImpl) DetectClones(ctx context.Context, req *domain.CloneRequest) (*domain.CloneResponse, error) {
	startTime := time.Now()

	// Apply request-specific thresholds to config
	config := *s.config
	if req.MinLines > 0 {
		config.MinLines = req.MinLines
	}
	if req.MinNodes > 0 {
		config.MinNodes = req.MinNodes
	}
	if req.Type1Threshold > 0 {
		config.Type1Threshold = req.Type1Threshold
	}
	if req.Type2Threshold > 0 {
		config.Type2Threshold = req.Type2Threshold
	}
	if req.Type3Threshold > 0 {
		config.Type3Threshold = req.Type3Threshold
	}
	if req.Type4Threshold > 0 {
		config.Type4Threshold = req.Type4Threshold
	}
	if req.MaxEditDistance > 0 {
		config.MaxEditDistance = req.MaxEditDistance
	}
	config.IgnoreLiterals = req.IgnoreLiterals
	config.IgnoreIdentifiers = req.IgnoreIdentifiers

	// Create clone detector with configured settings
	detector := analyzer.NewCloneDetector(&config)

	// Extract fragments from all files
	var allFragments []*analyzer.CodeFragment
	filesAnalyzed := 0
	linesAnalyzed := 0
	var errors []string

	for _, filePath := range req.Paths {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("clone detection cancelled: %w", ctx.Err())
		default:
		}

		// Read file
		content, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("[%s] Failed to read file: %v", filePath, err))
			continue
		}

		// Parse file
		ast, err := parser.ParseForLanguage(filePath, content)
		if err != nil {
			errors = append(errors, fmt.Sprintf("[%s] Failed to parse: %v", filePath, err))
			continue
		}

		// Extract fragments from the AST
		fragments := detector.ExtractFragments(ast.Body, filePath)
		allFragments = append(allFragments, fragments...)

		filesAnalyzed++
		linesAnalyzed += countLines(content)
	}

	if len(allFragments) == 0 {
		// No fragments found, return empty response
		return &domain.CloneResponse{
			Clones:      []*domain.Clone{},
			ClonePairs:  []*domain.ClonePair{},
			CloneGroups: []*domain.CloneGroup{},
			Statistics: &domain.CloneStatistics{
				TotalClones:       0,
				TotalClonePairs:   0,
				TotalCloneGroups:  0,
				ClonesByType:      make(map[string]int),
				AverageSimilarity: 0,
				LinesAnalyzed:     linesAnalyzed,
				FilesAnalyzed:     filesAnalyzed,
			},
			Duration: time.Since(startTime).Milliseconds(),
			Success:  true,
		}, nil
	}

	// Determine whether to use LSH acceleration
	useLSH := domain.ShouldUseLSH(req.LSHEnabled, len(allFragments), req.LSHAutoThreshold)
	if useLSH {
		detector.SetUseLSH(true)
	}

	// Detect clones
	var clonePairs []*domain.ClonePair
	var cloneGroups []*domain.CloneGroup

	if useLSH {
		clonePairs, cloneGroups = detector.DetectClonesWithLSH(ctx, allFragments)
	} else {
		clonePairs, cloneGroups = detector.DetectClonesWithContext(ctx, allFragments)
	}

	// Build statistics
	statistics := s.buildStatistics(clonePairs, cloneGroups, filesAnalyzed, linesAnalyzed)

	// Sort clone pairs by similarity (descending)
	sort.Slice(clonePairs, func(i, j int) bool {
		return clonePairs[i].Similarity > clonePairs[j].Similarity
	})

	// Extract unique clones from pairs
	clones := s.extractUniqueClones(clonePairs)

	return &domain.CloneResponse{
		Clones:      clones,
		ClonePairs:  clonePairs,
		CloneGroups: cloneGroups,
		Statistics:  statistics,
		Duration:    time.Since(startTime).Milliseconds(),
		Success:     true,
	}, nil
}

// DetectClonesInFiles performs clone detection on specific files
func (s *CloneServiceImpl) DetectClonesInFiles(ctx context.Context, filePaths []string, req *domain.CloneRequest) (*domain.CloneResponse, error) {
	singleReq := *req
	singleReq.Paths = filePaths
	return s.DetectClones(ctx, &singleReq)
}

// ComputeSimilarity computes similarity between two code fragments
func (s *CloneServiceImpl) ComputeSimilarity(ctx context.Context, fragment1, fragment2 string) (float64, error) {
	// This would require parsing both fragments and computing APTED distance
	// For now, return a placeholder
	return 0.0, fmt.Errorf("ComputeSimilarity not yet implemented")
}

// buildStatistics builds clone detection statistics
func (s *CloneServiceImpl) buildStatistics(pairs []*domain.ClonePair, groups []*domain.CloneGroup, filesAnalyzed, linesAnalyzed int) *domain.CloneStatistics {
	stats := &domain.CloneStatistics{
		TotalClonePairs:  len(pairs),
		TotalCloneGroups: len(groups),
		ClonesByType:     make(map[string]int),
		FilesAnalyzed:    filesAnalyzed,
		LinesAnalyzed:    linesAnalyzed,
	}

	if len(pairs) == 0 {
		return stats
	}

	// Count clones by type and calculate average similarity
	totalSimilarity := 0.0
	uniqueClones := make(map[string]bool)

	for _, pair := range pairs {
		stats.ClonesByType[pair.Type.String()]++
		totalSimilarity += pair.Similarity

		// Track unique clone locations
		if pair.Clone1 != nil && pair.Clone1.Location != nil {
			key := pair.Clone1.Location.String()
			uniqueClones[key] = true
		}
		if pair.Clone2 != nil && pair.Clone2.Location != nil {
			key := pair.Clone2.Location.String()
			uniqueClones[key] = true
		}
	}

	stats.TotalClones = len(uniqueClones)
	stats.AverageSimilarity = totalSimilarity / float64(len(pairs))

	return stats
}

// extractUniqueClones extracts unique clones from clone pairs
func (s *CloneServiceImpl) extractUniqueClones(pairs []*domain.ClonePair) []*domain.Clone {
	seen := make(map[string]*domain.Clone)
	var clones []*domain.Clone

	for _, pair := range pairs {
		if pair.Clone1 != nil && pair.Clone1.Location != nil {
			key := pair.Clone1.Location.String()
			if _, exists := seen[key]; !exists {
				seen[key] = pair.Clone1
				clones = append(clones, pair.Clone1)
			}
		}
		if pair.Clone2 != nil && pair.Clone2.Location != nil {
			key := pair.Clone2.Location.String()
			if _, exists := seen[key]; !exists {
				seen[key] = pair.Clone2
				clones = append(clones, pair.Clone2)
			}
		}
	}

	return clones
}

// countLines counts the number of lines in content
func countLines(content []byte) int {
	count := 1
	for _, b := range content {
		if b == '\n' {
			count++
		}
	}
	return count
}

// GetVersion returns the current version for response metadata
func (s *CloneServiceImpl) GetVersion() string {
	return version.Version
}
