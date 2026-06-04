package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// RoundRobinScheduler cycles through available nodes in order.
type RoundRobinScheduler struct {
	counter atomic.Uint64
}

func NewRoundRobinScheduler() *RoundRobinScheduler {
	return &RoundRobinScheduler{}
}

func (s *RoundRobinScheduler) Name() string {
	return RoundRobin
}

func (s *RoundRobinScheduler) Select(ctx context.Context, nodes []*common.Node, statuses map[string]*common.NodeStatus) (*common.Node, error) {
	available := filterAvailableNodes(nodes, statuses)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	idx := s.counter.Add(1) % uint64(len(available))
	return available[idx], nil
}

// filterAvailableNodes returns nodes that are online or degraded.
func filterAvailableNodes(nodes []*common.Node, statuses map[string]*common.NodeStatus) []*common.Node {
	var available []*common.Node
	for _, n := range nodes {
		if n.Status == common.NodeStatusOnline || n.Status == common.NodeStatusDegraded {
			available = append(available, n)
		}
	}
	return available
}

// filterOnlineNodes returns only nodes with status "online" (not degraded).
func filterOnlineNodes(nodes []*common.Node) []*common.Node {
	var available []*common.Node
	for _, n := range nodes {
		if n.Status == common.NodeStatusOnline {
			available = append(available, n)
		}
	}
	return available
}
