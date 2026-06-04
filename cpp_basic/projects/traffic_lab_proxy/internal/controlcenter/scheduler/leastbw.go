package scheduler

import (
	"context"
	"fmt"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// LeastBwScheduler selects the node with the lowest bandwidth utilization.
type LeastBwScheduler struct{}

func NewLeastBwScheduler() *LeastBwScheduler {
	return &LeastBwScheduler{}
}

func (s *LeastBwScheduler) Name() string {
	return LeastBw
}

func (s *LeastBwScheduler) Select(ctx context.Context, nodes []*common.Node, statuses map[string]*common.NodeStatus) (*common.Node, error) {
	available := filterAvailableNodes(nodes, statuses)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	var bestNode *common.Node
	minRatio := float64(^uint(0) >> 1) // max float

	for _, n := range available {
		rx := 0.0
		tx := 0.0
		if ns, ok := statuses[n.NodeID]; ok {
			rx = ns.RxMbps
			tx = ns.TxMbps
		}

		totalBw := rx + tx
		ratio := 0.0
		if n.MaxBandwidthMbps > 0 {
			ratio = totalBw / float64(n.MaxBandwidthMbps)
		}

		// Skip nodes at 90% bandwidth
		if ratio >= 0.9 {
			continue
		}

		if ratio < minRatio {
			minRatio = ratio
			bestNode = n
		}
	}

	if bestNode == nil {
		return nil, fmt.Errorf("all nodes at bandwidth capacity")
	}

	return bestNode, nil
}
