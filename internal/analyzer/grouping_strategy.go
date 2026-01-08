package analyzer

import (
	"container/list"
	"fmt"
	"sort"

	"github.com/ludo-technologies/jscan/domain"
)

// GroupingMode represents the mode of grouping strategy
type GroupingMode string

const (
	GroupingModeConnected       GroupingMode = "connected"
	GroupingModeKCore           GroupingMode = "k_core"
	GroupingModeStarMedoid      GroupingMode = "star_medoid"
	GroupingModeCompleteLinkage GroupingMode = "complete_linkage"
	GroupingModeCentroid        GroupingMode = "centroid"
)

// Algorithm-specific constants
const (
	starMedoidMaxIterations    = 10
	starMedoidConvergenceRatio = 0.01
)

// GroupingConfig holds configuration for clone grouping
type GroupingConfig struct {
	Mode           GroupingMode
	Threshold      float64
	KCoreK         int
	Type1Threshold float64
	Type2Threshold float64
	Type3Threshold float64
	Type4Threshold float64
}

// GroupingStrategy defines a strategy for grouping clone pairs into clone groups.
type GroupingStrategy interface {
	// GroupClones groups the given clone pairs into clone groups.
	GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup
	// GetName returns the strategy name.
	GetName() string
}

// CreateGroupingStrategy creates a grouping strategy based on config
func CreateGroupingStrategy(config GroupingConfig) GroupingStrategy {
	switch config.Mode {
	case GroupingModeKCore:
		return NewKCoreGrouping(config.Threshold, config.KCoreK)
	case GroupingModeStarMedoid:
		return NewStarMedoidGrouping(config.Threshold)
	case GroupingModeCompleteLinkage:
		return NewCompleteLinkageGrouping(config.Threshold)
	case GroupingModeCentroid:
		return NewCentroidGrouping(config.Threshold)
	case GroupingModeConnected:
		fallthrough
	default:
		return NewConnectedGrouping(config.Threshold)
	}
}

// ConnectedGrouping wraps transitive grouping logic using Union-Find
type ConnectedGrouping struct {
	threshold float64
}

func NewConnectedGrouping(threshold float64) *ConnectedGrouping {
	return &ConnectedGrouping{threshold: threshold}
}

func (c *ConnectedGrouping) GetName() string { return "Connected Components" }

func (c *ConnectedGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build set of clones and adjacency filtered by threshold
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)

		// Cache similarity and type for existing pair
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Union-Find across edges with similarity >= threshold
	parent := make(map[int]int, len(clones))
	rank := make(map[int]int, len(clones))

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra := find(a)
		rb := find(b)
		if ra == rb {
			return
		}
		if rank[ra] < rank[rb] {
			parent[ra] = rb
		} else if rank[ra] > rank[rb] {
			parent[rb] = ra
		} else {
			parent[rb] = ra
			rank[ra]++
		}
	}
	for _, clone := range clones {
		parent[clone.ID] = clone.ID
		rank[clone.ID] = 0
	}

	// Union only for edges meeting threshold
	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		if p.Similarity >= c.threshold {
			union(p.Clone1.ID, p.Clone2.ID)
		}
	}

	// Build components
	comp := make(map[int][]*domain.Clone)
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
		r := find(clone.ID)
		comp[r] = append(comp[r], clone)
	}

	// Convert to groups, exclude singletons
	groups := make([]*domain.CloneGroup, 0, len(comp))
	groupID := 0
	for _, members := range comp {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return cloneLess(members[i], members[j]) })
		g := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(members)),
			Size:   len(members),
		}
		groupID++
		for _, clone := range members {
			g.AddClone(clone)
		}
		// Compute average similarity using cached pairs among members
		g.Similarity = averageGroupSimilarityClones(simMap, members)
		// Determine predominant clone type from within-group available pairs
		g.Type = majorityCloneTypeClones(typeMap, members)
		groups = append(groups, g)
	}

	// Sort groups by decreasing similarity then size
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
	})

	return groups
}

// KCoreGrouping ensures each clone has at least k similar neighbors
type KCoreGrouping struct {
	threshold float64
	k         int
}

func NewKCoreGrouping(threshold float64, k int) *KCoreGrouping {
	if k < 2 {
		k = 2 // Minimum meaningful value
	}
	return &KCoreGrouping{threshold: threshold, k: k}
}

func (kg *KCoreGrouping) GetName() string { return fmt.Sprintf("%d-Core", kg.k) }

func (kg *KCoreGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Collect unique clones and build adjacency with edges meeting threshold
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	adj := make(map[int]map[int]float64)
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
			adj[clone.ID] = make(map[int]float64)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
		if p.Similarity >= kg.threshold {
			adj[p.Clone1.ID][p.Clone2.ID] = p.Similarity
			adj[p.Clone2.ID][p.Clone1.ID] = p.Similarity
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build clone ID to clone map
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
	}

	// Compute initial degrees
	degree := make(map[int]int, len(clones))
	for id, nbrs := range adj {
		degree[id] = len(nbrs)
	}

	// Queue for clones with degree < k
	q := list.New()
	inQueue := make(map[int]bool)
	for id, d := range degree {
		if d < kg.k {
			q.PushBack(id)
			inQueue[id] = true
		}
	}

	// Iteratively remove low-degree clones
	removed := make(map[int]bool)
	for q.Len() > 0 {
		e := q.Front()
		q.Remove(e)
		v := e.Value.(int)
		if removed[v] {
			continue
		}
		removed[v] = true
		// Decrease degree of neighbors
		for u := range adj[v] {
			if removed[u] {
				continue
			}
			degree[u]--
			delete(adj[u], v)
			if degree[u] < kg.k && !inQueue[u] {
				q.PushBack(u)
				inQueue[u] = true
			}
		}
		// Clear v's adjacency
		delete(adj, v)
	}

	// Remaining clones form the k-core subgraph
	// Now find connected components among remaining clones
	groups := make([]*domain.CloneGroup, 0)
	visited := make(map[int]bool)
	groupID := 0

	// Build deterministic order
	sort.Slice(clones, func(i, j int) bool { return cloneLess(clones[i], clones[j]) })

	for _, start := range clones {
		if removed[start.ID] || visited[start.ID] || adj[start.ID] == nil {
			continue
		}
		// BFS/DFS to collect component
		stack := []int{start.ID}
		component := make([]*domain.Clone, 0)
		visited[start.ID] = true
		for len(stack) > 0 {
			v := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			component = append(component, cloneByID[v])
			for u := range adj[v] {
				if !removed[u] && !visited[u] {
					visited[u] = true
					stack = append(stack, u)
				}
			}
		}
		if len(component) < 2 {
			continue
		}
		sort.Slice(component, func(i, j int) bool { return cloneLess(component[i], component[j]) })
		g := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(component)),
			Size:   len(component),
		}
		groupID++
		for _, clone := range component {
			g.AddClone(clone)
		}
		g.Similarity = averageGroupSimilarityClones(simMap, component)
		g.Type = majorityCloneTypeClones(typeMap, component)
		groups = append(groups, g)
	}

	// Sort groups by similarity then size
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
	})

	return groups
}

// StarMedoidGrouping uses iterative medoid optimization for balanced precision/recall
type StarMedoidGrouping struct {
	threshold float64
}

func NewStarMedoidGrouping(threshold float64) *StarMedoidGrouping {
	return &StarMedoidGrouping{threshold: threshold}
}

func (s *StarMedoidGrouping) GetName() string { return "Star/Medoid" }

func (s *StarMedoidGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build set of clones and similarity map
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build clone ID to clone map
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
	}

	// Phase 1: Initial clustering using Union-Find (same as ConnectedGrouping)
	parent := make(map[int]int, len(clones))
	rank := make(map[int]int, len(clones))

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra := find(a)
		rb := find(b)
		if ra == rb {
			return
		}
		if rank[ra] < rank[rb] {
			parent[ra] = rb
		} else if rank[ra] > rank[rb] {
			parent[rb] = ra
		} else {
			parent[rb] = ra
			rank[ra]++
		}
	}
	for _, clone := range clones {
		parent[clone.ID] = clone.ID
		rank[clone.ID] = 0
	}

	// Union only for edges meeting threshold
	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		if p.Similarity >= s.threshold {
			union(p.Clone1.ID, p.Clone2.ID)
		}
	}

	// Build initial components
	comp := make(map[int][]*domain.Clone)
	for _, clone := range clones {
		r := find(clone.ID)
		comp[r] = append(comp[r], clone)
	}

	// Convert to groups (including singletons for now, we'll filter later)
	type groupData struct {
		members []*domain.Clone
		medoid  *domain.Clone
	}
	groups := make([]*groupData, 0)
	for _, members := range comp {
		if len(members) < 2 {
			continue
		}
		g := &groupData{members: members}
		groups = append(groups, g)
	}

	if len(groups) == 0 {
		return []*domain.CloneGroup{}
	}

	// Phase 2: Iterative medoid refinement
	for iter := 0; iter < starMedoidMaxIterations; iter++ {
		// Find medoid for each group
		for _, g := range groups {
			g.medoid = s.findMedoid(g.members, simMap)
		}

		// Reassign clones to closest medoid
		newAssignment := make(map[int]int) // clone ID -> group index
		changed := 0

		for _, clone := range clones {
			bestGroup := -1
			bestSim := -1.0

			for gi, g := range groups {
				if g.medoid == nil {
					continue
				}
				sim := cloneSimilarity(simMap, clone, g.medoid)
				if sim >= s.threshold && sim > bestSim {
					bestSim = sim
					bestGroup = gi
				}
			}

			// Find current group
			currentGroup := -1
			for gi, g := range groups {
				for _, m := range g.members {
					if m.ID == clone.ID {
						currentGroup = gi
						break
					}
				}
				if currentGroup >= 0 {
					break
				}
			}

			if bestGroup >= 0 {
				newAssignment[clone.ID] = bestGroup
				if bestGroup != currentGroup {
					changed++
				}
			}
		}

		// Rebuild groups from new assignments
		newGroups := make([]*groupData, len(groups))
		for i := range newGroups {
			newGroups[i] = &groupData{members: make([]*domain.Clone, 0)}
		}
		for cloneID, gi := range newAssignment {
			newGroups[gi].members = append(newGroups[gi].members, cloneByID[cloneID])
		}

		// Filter empty groups
		filteredGroups := make([]*groupData, 0)
		for _, g := range newGroups {
			if len(g.members) >= 2 {
				filteredGroups = append(filteredGroups, g)
			}
		}
		groups = filteredGroups

		if len(groups) == 0 {
			return []*domain.CloneGroup{}
		}

		// Check convergence
		if float64(changed)/float64(len(clones)) < starMedoidConvergenceRatio {
			break
		}
	}

	// Phase 3: Finalize groups
	result := make([]*domain.CloneGroup, 0, len(groups))
	groupID := 0
	for _, g := range groups {
		if len(g.members) < 2 {
			continue
		}
		sort.Slice(g.members, func(i, j int) bool { return cloneLess(g.members[i], g.members[j]) })
		cg := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(g.members)),
			Size:   len(g.members),
		}
		groupID++
		for _, clone := range g.members {
			cg.AddClone(clone)
		}
		cg.Similarity = averageGroupSimilarityClones(simMap, g.members)
		cg.Type = majorityCloneTypeClones(typeMap, g.members)
		result = append(result, cg)
	}

	// Sort groups by similarity then size
	sort.Slice(result, func(i, j int) bool {
		if !almostEqual(result[i].Similarity, result[j].Similarity) {
			return result[i].Similarity > result[j].Similarity
		}
		if result[i].Size != result[j].Size {
			return result[i].Size > result[j].Size
		}
		if len(result[i].Clones) == 0 || len(result[j].Clones) == 0 {
			return false
		}
		return cloneLess(result[i].Clones[0], result[j].Clones[0])
	})

	return result
}

// findMedoid returns the clone with highest average similarity to all other members
func (s *StarMedoidGrouping) findMedoid(members []*domain.Clone, simMap map[string]float64) *domain.Clone {
	if len(members) == 0 {
		return nil
	}
	if len(members) == 1 {
		return members[0]
	}

	var bestMedoid *domain.Clone
	bestAvgSim := -1.0

	for _, candidate := range members {
		sumSim := 0.0
		for _, other := range members {
			if candidate.ID != other.ID {
				sumSim += cloneSimilarity(simMap, candidate, other)
			}
		}
		avgSim := sumSim / float64(len(members)-1)
		if avgSim > bestAvgSim {
			bestAvgSim = avgSim
			bestMedoid = candidate
		}
	}

	return bestMedoid
}

// CompleteLinkageGrouping ensures all pairs within a group have similarity above threshold
type CompleteLinkageGrouping struct {
	threshold float64
}

func NewCompleteLinkageGrouping(threshold float64) *CompleteLinkageGrouping {
	return &CompleteLinkageGrouping{threshold: threshold}
}

func (c *CompleteLinkageGrouping) GetName() string { return "Complete Linkage" }

func (c *CompleteLinkageGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build adjacency set and clone map
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	adj := make(map[int]map[int]bool)
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
			adj[clone.ID] = make(map[int]bool)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
		if p.Similarity >= c.threshold {
			adj[p.Clone1.ID][p.Clone2.ID] = true
			adj[p.Clone2.ID][p.Clone1.ID] = true
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build clone ID to clone map and sorted ID list
	cloneByID := make(map[int]*domain.Clone)
	cloneIDs := make([]int, 0, len(clones))
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
		cloneIDs = append(cloneIDs, clone.ID)
	}
	sort.Ints(cloneIDs)

	// Find all maximal cliques using Bron-Kerbosch with pivot
	cliques := make([][]int, 0)
	c.bronKerbosch(
		[]int{},                 // R: current clique
		cloneIDs,                // P: potential candidates
		[]int{},                 // X: excluded
		adj,
		&cliques,
	)

	// Filter cliques by minimum size and deduplicate
	filteredCliques := make([][]int, 0)
	for _, clique := range cliques {
		if len(clique) >= 2 {
			filteredCliques = append(filteredCliques, clique)
		}
	}

	// Sort cliques by size (descending) for deterministic output
	sort.Slice(filteredCliques, func(i, j int) bool {
		return len(filteredCliques[i]) > len(filteredCliques[j])
	})

	// Convert to CloneGroups
	groups := make([]*domain.CloneGroup, 0, len(filteredCliques))
	groupID := 0
	for _, clique := range filteredCliques {
		members := make([]*domain.Clone, 0, len(clique))
		for _, id := range clique {
			members = append(members, cloneByID[id])
		}
		sort.Slice(members, func(i, j int) bool { return cloneLess(members[i], members[j]) })

		g := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(members)),
			Size:   len(members),
		}
		groupID++
		for _, clone := range members {
			g.AddClone(clone)
		}
		g.Similarity = averageGroupSimilarityClones(simMap, members)
		g.Type = majorityCloneTypeClones(typeMap, members)
		groups = append(groups, g)
	}

	// Sort groups by similarity then size
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
	})

	return groups
}

// bronKerbosch implements Bron-Kerbosch algorithm with pivot for finding maximal cliques
func (c *CompleteLinkageGrouping) bronKerbosch(R, P, X []int, adj map[int]map[int]bool, result *[][]int) {
	if len(P) == 0 && len(X) == 0 {
		if len(R) >= 2 {
			clique := make([]int, len(R))
			copy(clique, R)
			*result = append(*result, clique)
		}
		return
	}

	// Choose pivot from P ∪ X that maximizes |P ∩ N(u)|
	pivot := c.choosePivot(P, X, adj)
	pivotNeighbors := adj[pivot]

	// Iterate over P \ N(pivot)
	pCopy := make([]int, len(P))
	copy(pCopy, P)

	for _, v := range pCopy {
		if pivotNeighbors[v] {
			continue // Skip neighbors of pivot
		}

		// New R = R ∪ {v}
		newR := append([]int{}, R...)
		newR = append(newR, v)

		// New P = P ∩ N(v)
		newP := c.intersect(P, adj[v])

		// New X = X ∩ N(v)
		newX := c.intersect(X, adj[v])

		c.bronKerbosch(newR, newP, newX, adj, result)

		// P = P \ {v}
		P = c.remove(P, v)
		// X = X ∪ {v}
		X = append(X, v)
	}
}

// choosePivot selects a pivot vertex that maximizes connections to P
func (c *CompleteLinkageGrouping) choosePivot(P, X []int, adj map[int]map[int]bool) int {
	maxConnections := -1
	pivot := -1

	candidates := append([]int{}, P...)
	candidates = append(candidates, X...)

	for _, u := range candidates {
		connections := 0
		for _, p := range P {
			if adj[u][p] {
				connections++
			}
		}
		if connections > maxConnections {
			maxConnections = connections
			pivot = u
		}
	}

	if pivot == -1 && len(P) > 0 {
		pivot = P[0]
	}

	return pivot
}

// intersect returns the intersection of slice and set (represented as map keys)
func (c *CompleteLinkageGrouping) intersect(slice []int, set map[int]bool) []int {
	result := make([]int, 0)
	for _, v := range slice {
		if set[v] {
			result = append(result, v)
		}
	}
	return result
}

// remove removes an element from a slice
func (c *CompleteLinkageGrouping) remove(slice []int, elem int) []int {
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if v != elem {
			result = append(result, v)
		}
	}
	return result
}

// CentroidGrouping uses BFS expansion with strict similarity to all existing members
type CentroidGrouping struct {
	threshold float64
}

func NewCentroidGrouping(threshold float64) *CentroidGrouping {
	return &CentroidGrouping{threshold: threshold}
}

func (cg *CentroidGrouping) GetName() string { return "Centroid" }

func (cg *CentroidGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build similarity map and adjacency
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)
	neighbors := make(map[int][]int) // clone ID -> neighbor IDs above threshold

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
			neighbors[clone.ID] = make([]int, 0)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
		if p.Similarity >= cg.threshold {
			neighbors[p.Clone1.ID] = append(neighbors[p.Clone1.ID], p.Clone2.ID)
			neighbors[p.Clone2.ID] = append(neighbors[p.Clone2.ID], p.Clone1.ID)
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Sort clones for deterministic processing
	sort.Slice(clones, func(i, j int) bool { return cloneLess(clones[i], clones[j]) })

	// Build clone ID to clone map
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
	}

	// BFS expansion from each unassigned clone
	assigned := make(map[int]bool)
	groups := make([]*domain.CloneGroup, 0)
	groupID := 0

	for _, seed := range clones {
		if assigned[seed.ID] {
			continue
		}

		// Start new group with seed
		members := []*domain.Clone{seed}
		assigned[seed.ID] = true

		// BFS queue: neighbor IDs to consider
		queue := list.New()
		visited := make(map[int]bool)
		visited[seed.ID] = true

		// Add seed's neighbors to queue
		for _, nid := range neighbors[seed.ID] {
			if !visited[nid] && !assigned[nid] {
				queue.PushBack(nid)
				visited[nid] = true
			}
		}

		// BFS expansion
		for queue.Len() > 0 {
			e := queue.Front()
			queue.Remove(e)
			candidateID := e.Value.(int)

			if assigned[candidateID] {
				continue
			}

			candidate := cloneByID[candidateID]
			if candidate == nil {
				continue
			}

			// Check if candidate is similar to ALL current members
			if cg.isSimilarToAll(candidate, members, simMap) {
				members = append(members, candidate)
				assigned[candidateID] = true

				// Add candidate's neighbors to queue
				for _, nid := range neighbors[candidateID] {
					if !visited[nid] && !assigned[nid] {
						queue.PushBack(nid)
						visited[nid] = true
					}
				}
			}
		}

		// Only keep groups with at least 2 members
		if len(members) >= 2 {
			sort.Slice(members, func(i, j int) bool { return cloneLess(members[i], members[j]) })
			g := &domain.CloneGroup{
				ID:     groupID,
				Clones: make([]*domain.Clone, 0, len(members)),
				Size:   len(members),
			}
			groupID++
			for _, clone := range members {
				g.AddClone(clone)
			}
			g.Similarity = averageGroupSimilarityClones(simMap, members)
			g.Type = majorityCloneTypeClones(typeMap, members)
			groups = append(groups, g)
		}
	}

	// Sort groups by similarity then size
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
	})

	return groups
}

// isSimilarToAll checks if candidate is similar to all members above threshold
func (cg *CentroidGrouping) isSimilarToAll(candidate *domain.Clone, members []*domain.Clone, simMap map[string]float64) bool {
	for _, member := range members {
		if cloneSimilarity(simMap, candidate, member) < cg.threshold {
			return false
		}
	}
	return true
}

// Helper functions

// clonePairKey creates a canonical key for a pair of clones
func clonePairKey(a, b *domain.Clone) string {
	ka := cloneID(a)
	kb := cloneID(b)
	if ka <= kb {
		return ka + "||" + kb
	}
	return kb + "||" + ka
}

// cloneID returns a stable identifier for a clone based on its location
func cloneID(c *domain.Clone) string {
	if c == nil || c.Location == nil {
		return fmt.Sprintf("%p", c)
	}
	loc := c.Location
	return fmt.Sprintf("%s|%d|%d|%d|%d", loc.FilePath, loc.StartLine, loc.EndLine, loc.StartCol, loc.EndCol)
}

// cloneLess provides deterministic ordering between two clones by location
func cloneLess(a, b *domain.Clone) bool {
	if a == b {
		return false
	}
	if a == nil {
		return true
	}
	if b == nil {
		return false
	}
	al, bl := a.Location, b.Location
	if al == nil && bl == nil {
		return a.ID < b.ID
	}
	if al == nil {
		return true
	}
	if bl == nil {
		return false
	}
	if al.FilePath != bl.FilePath {
		return al.FilePath < bl.FilePath
	}
	if al.StartLine != bl.StartLine {
		return al.StartLine < bl.StartLine
	}
	if al.StartCol != bl.StartCol {
		return al.StartCol < bl.StartCol
	}
	if al.EndLine != bl.EndLine {
		return al.EndLine < bl.EndLine
	}
	return al.EndCol < bl.EndCol
}

// similarity returns cached similarity, or 0 if not present
func cloneSimilarity(sims map[string]float64, a, b *domain.Clone) float64 {
	if a == nil || b == nil {
		return 0.0
	}
	if a == b || a.ID == b.ID {
		return 1.0
	}
	key := clonePairKey(a, b)
	if s, ok := sims[key]; ok {
		return s
	}
	return 0.0
}

// averageGroupSimilarityClones computes average pairwise similarity among clones using cache
func averageGroupSimilarityClones(sims map[string]float64, members []*domain.Clone) float64 {
	if len(members) < 2 {
		return 1.0
	}
	sum := 0.0
	cnt := 0
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			sum += cloneSimilarity(sims, members[i], members[j])
			cnt++
		}
	}
	if cnt == 0 {
		return 0.0
	}
	return sum / float64(cnt)
}

// majorityCloneTypeClones chooses the most frequent CloneType among all pair edges in members
func majorityCloneTypeClones(typeMap map[string]domain.CloneType, members []*domain.Clone) domain.CloneType {
	counts := make(map[domain.CloneType]int)
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			key := clonePairKey(members[i], members[j])
			if t, ok := typeMap[key]; ok {
				counts[t]++
			}
		}
	}
	var best domain.CloneType
	maxC := -1
	for t, c := range counts {
		if c > maxC {
			maxC = c
			best = t
		}
	}
	if maxC < 0 {
		return domain.Type3Clone // fallback reasonable default
	}
	return best
}

func almostEqual(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}
