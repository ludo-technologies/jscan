package analyzer

import (
	"fmt"

	coreclone "github.com/ludo-technologies/codescan-core/clone"
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

// ---------------------------------------------------------------------------
// coreGroupingAdapter wraps a codescan-core GroupingStrategy
// ---------------------------------------------------------------------------

type coreGroupingAdapter struct {
	core coreclone.GroupingStrategy[*domain.Clone]
	name string
}

func (a *coreGroupingAdapter) GetName() string { return a.name }

func (a *coreGroupingAdapter) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	corePairs := make([]*coreclone.ItemPair[*domain.Clone], 0, len(pairs))
	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		corePairs = append(corePairs, &coreclone.ItemPair[*domain.Clone]{
			Item1:      p.Clone1,
			Item2:      p.Clone2,
			Similarity: p.Similarity,
			PairType:   int(p.Type),
		})
	}

	coreGroups := a.core.GroupItems(corePairs)

	groups := make([]*domain.CloneGroup, 0, len(coreGroups))
	for _, cg := range coreGroups {
		if len(cg.Items) < 2 {
			continue // Skip singletons
		}
		g := &domain.CloneGroup{
			ID:         cg.ID,
			Clones:     cg.Items,
			Type:       domain.CloneType(cg.GroupType),
			Similarity: cg.Similarity,
			Size:       len(cg.Items),
		}
		groups = append(groups, g)
	}

	return groups
}

// ---------------------------------------------------------------------------
// ConnectedGrouping wraps codescan-core connected components
// ---------------------------------------------------------------------------

type ConnectedGrouping struct {
	threshold float64
	adapter   *coreGroupingAdapter
}

func NewConnectedGrouping(threshold float64) *ConnectedGrouping {
	core := coreclone.NewGroupingStrategy[*domain.Clone](coreclone.GroupingConfig{
		Mode:      coreclone.ModeConnected,
		Threshold: threshold,
	})
	return &ConnectedGrouping{
		threshold: threshold,
		adapter:   &coreGroupingAdapter{core: core, name: "Connected Components"},
	}
}

func (c *ConnectedGrouping) GetName() string { return c.adapter.GetName() }
func (c *ConnectedGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	return c.adapter.GroupClones(pairs)
}

// ---------------------------------------------------------------------------
// KCoreGrouping wraps codescan-core k-core decomposition
// ---------------------------------------------------------------------------

type KCoreGrouping struct {
	threshold float64
	k         int
	adapter   *coreGroupingAdapter
}

func NewKCoreGrouping(threshold float64, k int) *KCoreGrouping {
	if k < 2 {
		k = 2 // Minimum meaningful value
	}
	core := coreclone.NewGroupingStrategy[*domain.Clone](coreclone.GroupingConfig{
		Mode:      coreclone.ModeKCore,
		Threshold: threshold,
		KCoreK:    k,
	})
	return &KCoreGrouping{
		threshold: threshold,
		k:         k,
		adapter:   &coreGroupingAdapter{core: core, name: fmt.Sprintf("%d-Core", k)},
	}
}

func (kg *KCoreGrouping) GetName() string { return kg.adapter.GetName() }
func (kg *KCoreGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	return kg.adapter.GroupClones(pairs)
}

// ---------------------------------------------------------------------------
// StarMedoidGrouping wraps codescan-core star/medoid grouping
// ---------------------------------------------------------------------------

type StarMedoidGrouping struct {
	threshold float64
	adapter   *coreGroupingAdapter
}

func NewStarMedoidGrouping(threshold float64) *StarMedoidGrouping {
	core := coreclone.NewGroupingStrategy[*domain.Clone](coreclone.GroupingConfig{
		Mode:      coreclone.ModeStarMedoid,
		Threshold: threshold,
	})
	return &StarMedoidGrouping{
		threshold: threshold,
		adapter:   &coreGroupingAdapter{core: core, name: "Star/Medoid"},
	}
}

func (s *StarMedoidGrouping) GetName() string { return s.adapter.GetName() }
func (s *StarMedoidGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	return s.adapter.GroupClones(pairs)
}

// ---------------------------------------------------------------------------
// CompleteLinkageGrouping wraps codescan-core complete linkage (Bron-Kerbosch)
// ---------------------------------------------------------------------------

type CompleteLinkageGrouping struct {
	threshold float64
	adapter   *coreGroupingAdapter
}

func NewCompleteLinkageGrouping(threshold float64) *CompleteLinkageGrouping {
	core := coreclone.NewGroupingStrategy[*domain.Clone](coreclone.GroupingConfig{
		Mode:      coreclone.ModeCompleteLinkage,
		Threshold: threshold,
	})
	return &CompleteLinkageGrouping{
		threshold: threshold,
		adapter:   &coreGroupingAdapter{core: core, name: "Complete Linkage"},
	}
}

func (c *CompleteLinkageGrouping) GetName() string { return c.adapter.GetName() }
func (c *CompleteLinkageGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	return c.adapter.GroupClones(pairs)
}

// ---------------------------------------------------------------------------
// CentroidGrouping wraps codescan-core centroid grouping
// ---------------------------------------------------------------------------

type CentroidGrouping struct {
	threshold float64
	adapter   *coreGroupingAdapter
}

func NewCentroidGrouping(threshold float64) *CentroidGrouping {
	core := coreclone.NewGroupingStrategy[*domain.Clone](coreclone.GroupingConfig{
		Mode:      coreclone.ModeCentroid,
		Threshold: threshold,
	})
	return &CentroidGrouping{
		threshold: threshold,
		adapter:   &coreGroupingAdapter{core: core, name: "Centroid"},
	}
}

func (cg *CentroidGrouping) GetName() string { return cg.adapter.GetName() }
func (cg *CentroidGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	return cg.adapter.GroupClones(pairs)
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

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
