package analyzer

import (
	"fmt"
	"sort"
	"strings"

	coregraph "github.com/ludo-technologies/codescan-core/graph"
	"github.com/ludo-technologies/jscan/domain"
)

// CircularDependencyDetector detects circular dependencies using Tarjan's algorithm
type CircularDependencyDetector struct{}

// NewCircularDependencyDetector creates a new CircularDependencyDetector
func NewCircularDependencyDetector() *CircularDependencyDetector {
	return &CircularDependencyDetector{}
}

// DetectCycles finds all cycles in the dependency graph using Tarjan's SCC algorithm
func (d *CircularDependencyDetector) DetectCycles(graph *domain.DependencyGraph) *domain.CircularDependencyAnalysis {
	if graph == nil || graph.NodeCount() == 0 {
		return &domain.CircularDependencyAnalysis{
			HasCircularDependencies: false,
			TotalCycles:             0,
			TotalModulesInCycles:    0,
			CircularDependencies:    []domain.CircularDependency{},
		}
	}

	// Use codescan-core's Tarjan SCC implementation
	adapter := &domain.DirectedGraphAdapter{Graph: graph}
	cycleResult := coregraph.NewCycleDetector().DetectCycles(adapter)

	var cycles []domain.CircularDependency
	modulesInCycles := make(map[string]bool)
	coreInfrastructure := make(map[string]int)

	for _, scc := range cycleResult.Cycles {
		// Sort for deterministic output
		sort.Strings(scc)

		cycle := d.buildCycleInfo(scc, graph)
		cycles = append(cycles, cycle)

		for _, module := range scc {
			modulesInCycles[module] = true
			coreInfrastructure[module]++
		}
	}

	// Find core infrastructure (modules in multiple cycles)
	var coreModules []string
	for module, count := range coreInfrastructure {
		if count > 1 {
			coreModules = append(coreModules, module)
		}
	}
	sort.Strings(coreModules)

	// Generate cycle breaking suggestions
	suggestions := d.suggestCycleBreaking(cycles, graph)

	return &domain.CircularDependencyAnalysis{
		HasCircularDependencies:  len(cycles) > 0,
		TotalCycles:              len(cycles),
		TotalModulesInCycles:     len(modulesInCycles),
		CircularDependencies:     cycles,
		CycleBreakingSuggestions: suggestions,
		CoreInfrastructure:       coreModules,
	}
}

// buildCycleInfo creates a CircularDependency from an SCC
func (d *CircularDependencyDetector) buildCycleInfo(scc []string, graph *domain.DependencyGraph) domain.CircularDependency {
	// Find the cycle path (edges forming the cycle)
	var paths []domain.DependencyPath
	sccSet := make(map[string]bool)
	for _, module := range scc {
		sccSet[module] = true
	}

	// Find edges within the SCC
	for _, from := range scc {
		edges := graph.GetOutgoingEdges(from)
		for _, edge := range edges {
			if sccSet[edge.To] {
				paths = append(paths, domain.DependencyPath{
					From:   from,
					To:     edge.To,
					Path:   []string{from, edge.To},
					Length: 1,
				})
			}
		}
	}

	// Calculate severity based on cycle size
	severity := d.calculateCycleSeverity(scc)

	// Generate description
	description := d.generateCycleDescription(scc)

	return domain.CircularDependency{
		Modules:      scc,
		Dependencies: paths,
		Severity:     severity,
		Size:         len(scc),
		Description:  description,
	}
}

// calculateCycleSeverity determines the severity of a cycle based on its size
func (d *CircularDependencyDetector) calculateCycleSeverity(scc []string) domain.CycleSeverity {
	size := len(scc)
	switch {
	case size <= 2:
		return domain.CycleSeverityLow
	case size <= 4:
		return domain.CycleSeverityMedium
	case size <= 6:
		return domain.CycleSeverityHigh
	default:
		return domain.CycleSeverityCritical
	}
}

// generateCycleDescription creates a human-readable description of the cycle
func (d *CircularDependencyDetector) generateCycleDescription(scc []string) string {
	if len(scc) == 0 {
		return ""
	}

	// Sort for consistent output
	sorted := make([]string, len(scc))
	copy(sorted, scc)
	sort.Strings(sorted)

	// Create a simple cycle representation
	var parts []string
	for _, module := range sorted {
		// Extract just the filename for readability
		parts = append(parts, getModuleBaseName(module))
	}

	return fmt.Sprintf("Circular dependency involving %d modules: %s", len(scc), strings.Join(parts, " <-> "))
}

// suggestCycleBreaking generates suggestions for breaking cycles
func (d *CircularDependencyDetector) suggestCycleBreaking(cycles []domain.CircularDependency, graph *domain.DependencyGraph) []string {
	var suggestions []string

	for _, cycle := range cycles {
		if len(cycle.Modules) == 0 {
			continue
		}

		// Find the best edge to break
		bestEdge := d.findBestEdgeToBreak(cycle, graph)
		if bestEdge != nil {
			suggestion := fmt.Sprintf(
				"Consider removing or inverting the dependency from '%s' to '%s' to break the cycle",
				getModuleBaseName(bestEdge.From),
				getModuleBaseName(bestEdge.To),
			)
			suggestions = append(suggestions, suggestion)
		}

		// Suggest interface extraction for larger cycles
		if len(cycle.Modules) >= 3 {
			suggestion := fmt.Sprintf(
				"Consider extracting interfaces for modules: %s",
				strings.Join(getModuleBaseNames(cycle.Modules), ", "),
			)
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// findBestEdgeToBreak finds the edge with the lowest weight (least used) to break a cycle
func (d *CircularDependencyDetector) findBestEdgeToBreak(cycle domain.CircularDependency, graph *domain.DependencyGraph) *domain.DependencyEdge {
	var bestEdge *domain.DependencyEdge
	minWeight := int(^uint(0) >> 1) // Max int

	sccSet := make(map[string]bool)
	for _, module := range cycle.Modules {
		sccSet[module] = true
	}

	for _, module := range cycle.Modules {
		edges := graph.GetOutgoingEdges(module)
		for _, edge := range edges {
			if sccSet[edge.To] && edge.Weight < minWeight {
				minWeight = edge.Weight
				bestEdge = edge
			}
		}
	}

	return bestEdge
}

// getModuleBaseName extracts a readable module name from a path
func getModuleBaseName(path string) string {
	// Get the last component of the path
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// getModuleBaseNames extracts readable names from multiple paths
func getModuleBaseNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = getModuleBaseName(path)
	}
	return names
}

// FindCyclePath finds a path from one module to another within a cycle
func (d *CircularDependencyDetector) FindCyclePath(from, to string, graph *domain.DependencyGraph) []string {
	if graph == nil {
		return nil
	}

	visited := make(map[string]bool)
	path := []string{}

	var dfs func(current string) bool
	dfs = func(current string) bool {
		if current == to {
			path = append(path, current)
			return true
		}

		if visited[current] {
			return false
		}
		visited[current] = true
		path = append(path, current)

		edges := graph.GetOutgoingEdges(current)
		for _, edge := range edges {
			if dfs(edge.To) {
				return true
			}
		}

		// Backtrack
		path = path[:len(path)-1]
		return false
	}

	if dfs(from) {
		return path
	}
	return nil
}
