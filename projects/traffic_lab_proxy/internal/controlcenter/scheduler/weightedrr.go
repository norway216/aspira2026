package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// WeightedRRScheduler does weighted round-robin selection.
// Nodes with higher weights get more selections.
type WeightedRRScheduler struct {
	counter atomic.Uint64
}

func NewWeightedRRScheduler() *WeightedRRScheduler {
	return &WeightedRRScheduler{}
}

func (s *WeightedRRScheduler) Name() string {
	return WeightedRR
}

func (s *WeightedRRScheduler) Select(ctx context.Context, nodes []*common.Node, statuses map[string]*common.NodeStatus) (*common.Node, error) {
	available := filterAvailableNodes(nodes, statuses)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	// Calculate total weight
	totalWeight := 0
	for _, n := range available {
		w := n.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	// Pick based on counter and weight
	c := s.counter.Add(1) % uint64(totalWeight)
	cumulative := uint64(0)
	for _, n := range available {
		w := n.Weight
		if w <= 0 {
			w = 1
		}
		cumulative += uint64(w)
		if c < cumulative {
			return n, nil
		}
	}

	// Fallback
	return available[0], nil
}
