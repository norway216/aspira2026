package scheduler

import (
	"context"
	"fmt"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// CompositeScheduler uses a weighted scoring formula to select the best node.
// score = cpu_usage*0.25 + memory_usage*0.15 + bandwidth_ratio*0.40 + connection_ratio*0.20
// Lower score is better.
type CompositeScheduler struct{}

func NewCompositeScheduler() *CompositeScheduler {
	return &CompositeScheduler{}
}

func (s *CompositeScheduler) Name() string {
	return CompositeScore
}

func (s *CompositeScheduler) Select(ctx context.Context, nodes []*common.Node, statuses map[string]*common.NodeStatus) (*common.Node, error) {
	available := filterAvailableNodes(nodes, statuses)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	var bestNode *common.Node
	bestScore := float64(^uint(0) >> 1) // max float

	for _, n := range available {
		cpu := 0.0
		mem := 0.0
		rx := 0.0
		tx := 0.0
		conns := 0

		if ns, ok := statuses[n.NodeID]; ok {
			cpu = ns.CPUUsage
			mem = ns.MemoryUsage
			rx = ns.RxMbps
			tx = ns.TxMbps
			conns = ns.CurrentConnections
		}

		// Normalize bandwidth ratio (0-100)
		bwRatio := 0.0
		if n.MaxBandwidthMbps > 0 {
			bwRatio = ((rx + tx) / float64(n.MaxBandwidthMbps)) * 100
		}
		if bwRatio > 100 {
			bwRatio = 100
		}

		// Normalize connection ratio (0-100)
		connRatio := 0.0
		if n.MaxConnections > 0 {
			connRatio = (float64(conns) / float64(n.MaxConnections)) * 100
		}
		if connRatio > 100 {
			connRatio = 100
		}

		// Clamp CPU and memory to 0-100
		if cpu > 100 {
			cpu = 100
		}
		if mem > 100 {
			mem = 100
		}

		// Composite score
		score := cpu*0.25 + mem*0.15 + bwRatio*0.40 + connRatio*0.20

		// Penalty for degraded nodes
		if n.Status == common.NodeStatusDegraded {
			score += 20
		}

		if score < bestScore {
			bestScore = score
			bestNode = n
		}
	}

	if bestNode == nil {
		return nil, fmt.Errorf("no suitable node found")
	}

	return bestNode, nil
}
