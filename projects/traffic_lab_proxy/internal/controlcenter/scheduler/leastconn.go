package scheduler

import (
	"context"
	"fmt"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// LeastConnScheduler selects the node with the fewest current connections.
type LeastConnScheduler struct{}

func NewLeastConnScheduler() *LeastConnScheduler {
	return &LeastConnScheduler{}
}

func (s *LeastConnScheduler) Name() string {
	return LeastConn
}

func (s *LeastConnScheduler) Select(ctx context.Context, nodes []*common.Node, statuses map[string]*common.NodeStatus) (*common.Node, error) {
	available := filterAvailableNodes(nodes, statuses)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	var bestNode *common.Node
	minConns := int(^uint(0) >> 1) // max int

	for _, n := range available {
		conns := 0
		if ns, ok := statuses[n.NodeID]; ok {
			conns = ns.CurrentConnections
		}

		// Skip nodes at 90% capacity
		if n.MaxConnections > 0 && conns >= int(float64(n.MaxConnections)*0.9) {
			continue
		}

		if conns < minConns {
			minConns = conns
			bestNode = n
		}
	}

	if bestNode == nil {
		return nil, fmt.Errorf("all nodes at connection capacity")
	}

	return bestNode, nil
}
