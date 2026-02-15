package analyzer

import (
	"time"

	corecfg "github.com/ludo-technologies/codescan-core/cfg"
	"github.com/ludo-technologies/jscan/internal/parser"
)

// ReachabilityResult contains the results of reachability analysis
type ReachabilityResult struct {
	ReachableBlocks   map[string]*corecfg.BasicBlock
	UnreachableBlocks map[string]*corecfg.BasicBlock
	TotalBlocks       int
	ReachableCount    int
	UnreachableCount  int
	AnalysisTime      time.Duration
}

// ReachabilityAnalyzer performs reachability analysis on CFGs
type ReachabilityAnalyzer struct {
	cfg *corecfg.CFG
}

func NewReachabilityAnalyzer(cfg *corecfg.CFG) *ReachabilityAnalyzer {
	return &ReachabilityAnalyzer{cfg: cfg}
}

func (ra *ReachabilityAnalyzer) AnalyzeReachability() *ReachabilityResult {
	startTime := time.Now()

	result := &ReachabilityResult{
		ReachableBlocks:   make(map[string]*corecfg.BasicBlock),
		UnreachableBlocks: make(map[string]*corecfg.BasicBlock),
	}

	if ra.cfg == nil || ra.cfg.Entry == nil || ra.cfg.Blocks == nil {
		result.AnalysisTime = time.Since(startTime)
		return result
	}

	result.TotalBlocks = len(ra.cfg.Blocks)

	if len(ra.cfg.Blocks) == 0 {
		result.AnalysisTime = time.Since(startTime)
		return result
	}

	ra.performEnhancedReachabilityAnalysis(result)

	result.ReachableCount = len(result.ReachableBlocks)
	result.UnreachableCount = len(result.UnreachableBlocks)
	result.AnalysisTime = time.Since(startTime)

	return result
}

func (ra *ReachabilityAnalyzer) AnalyzeReachabilityFrom(startBlock *corecfg.BasicBlock) *ReachabilityResult {
	startTime := time.Now()

	result := &ReachabilityResult{
		ReachableBlocks:   make(map[string]*corecfg.BasicBlock),
		UnreachableBlocks: make(map[string]*corecfg.BasicBlock),
	}

	if ra.cfg == nil || ra.cfg.Blocks == nil || startBlock == nil {
		result.AnalysisTime = time.Since(startTime)
		return result
	}

	result.TotalBlocks = len(ra.cfg.Blocks)

	if len(ra.cfg.Blocks) == 0 {
		result.AnalysisTime = time.Since(startTime)
		return result
	}

	visited := make(map[string]bool)
	ra.traverseFrom(startBlock, visited, result.ReachableBlocks)

	for id, block := range ra.cfg.Blocks {
		if _, isReachable := result.ReachableBlocks[id]; !isReachable {
			result.UnreachableBlocks[id] = block
		}
	}

	result.ReachableCount = len(result.ReachableBlocks)
	result.UnreachableCount = len(result.UnreachableBlocks)
	result.AnalysisTime = time.Since(startTime)

	return result
}

func (ra *ReachabilityAnalyzer) traverseFrom(block *corecfg.BasicBlock, visited map[string]bool, reachable map[string]*corecfg.BasicBlock) {
	if block == nil || visited[block.ID] {
		return
	}

	visited[block.ID] = true
	reachable[block.ID] = block

	for _, edge := range block.Successors {
		ra.traverseFrom(edge.To, visited, reachable)
	}
}

func (result *ReachabilityResult) GetUnreachableBlocksWithStatements() map[string]*corecfg.BasicBlock {
	blocksWithStatements := make(map[string]*corecfg.BasicBlock)
	for id, block := range result.UnreachableBlocks {
		if !block.IsEmpty() {
			blocksWithStatements[id] = block
		}
	}
	return blocksWithStatements
}

func (result *ReachabilityResult) GetReachabilityRatio() float64 {
	if result.TotalBlocks == 0 {
		return 1.0
	}
	return float64(result.ReachableCount) / float64(result.TotalBlocks)
}

func (result *ReachabilityResult) HasUnreachableCode() bool {
	for _, block := range result.UnreachableBlocks {
		if !block.IsEmpty() {
			return true
		}
	}
	return false
}

// performEnhancedReachabilityAnalysis performs reachability analysis with all-paths-return detection
func (ra *ReachabilityAnalyzer) performEnhancedReachabilityAnalysis(result *ReachabilityResult) {
	// Use corecfg for basic structural reachability via Walk
	ra.cfg.Walk(&reachabilityVisitor{reachableBlocks: result.ReachableBlocks})

	// Then, apply jscan-specific all-paths-return analysis
	ra.detectAllPathsReturnUnreachability(result)

	for id, block := range ra.cfg.Blocks {
		if _, isReachable := result.ReachableBlocks[id]; !isReachable {
			result.UnreachableBlocks[id] = block
		}
	}
}

// reachabilityVisitor implements corecfg.Visitor
type reachabilityVisitor struct {
	reachableBlocks map[string]*corecfg.BasicBlock
}

func (rv *reachabilityVisitor) VisitBlock(block *corecfg.BasicBlock) bool {
	if block != nil {
		rv.reachableBlocks[block.ID] = block
	}
	return true
}

func (rv *reachabilityVisitor) VisitEdge(edge *corecfg.Edge) bool {
	return true
}

func (ra *ReachabilityAnalyzer) detectAllPathsReturnUnreachability(result *ReachabilityResult) {
	allPathsReturnBlocks := make(map[string]bool)

	for _, block := range ra.cfg.Blocks {
		if ra.allSuccessorsReturn(block, make(map[string]bool)) {
			allPathsReturnBlocks[block.ID] = true
		}
	}

	for blockID := range allPathsReturnBlocks {
		block := ra.cfg.Blocks[blockID]
		ra.markSuccessorsUnreachableAfterReturn(block, result, make(map[string]bool))
	}
}

func (ra *ReachabilityAnalyzer) allSuccessorsReturn(block *corecfg.BasicBlock, visited map[string]bool) bool {
	if block == nil {
		return false
	}

	if visited[block.ID] {
		return false
	}
	visited[block.ID] = true

	if ra.blockContainsReturn(block) {
		return true
	}

	if block == ra.cfg.Exit {
		return false
	}

	if len(block.Successors) == 0 {
		return false
	}

	for _, edge := range block.Successors {
		if edge.Type == corecfg.EdgeReturn && edge.To == ra.cfg.Exit {
			continue
		}

		if !ra.allSuccessorsReturn(edge.To, copyVisited(visited)) {
			return false
		}
	}

	return true
}

func (ra *ReachabilityAnalyzer) markSuccessorsUnreachableAfterReturn(block *corecfg.BasicBlock, result *ReachabilityResult, visited map[string]bool) {
	if block == nil || visited[block.ID] {
		return
	}
	visited[block.ID] = true

	if ra.blockContainsReturn(block) {
		for _, edge := range block.Successors {
			if edge.Type == corecfg.EdgeNormal {
				delete(result.ReachableBlocks, edge.To.ID)
				ra.markSuccessorsUnreachableAfterReturn(edge.To, result, copyVisited(visited))
			}
		}
	}
}

func (ra *ReachabilityAnalyzer) blockContainsReturn(block *corecfg.BasicBlock) bool {
	if block == nil {
		return false
	}

	for _, rawStmt := range block.Statements {
		stmt, ok := rawStmt.(*parser.Node)
		if ok && stmt != nil && stmt.Type == parser.NodeReturnStatement {
			return true
		}
	}

	return false
}

func copyVisited(visited map[string]bool) map[string]bool {
	cp := make(map[string]bool)
	for k, v := range visited {
		cp[k] = v
	}
	return cp
}
