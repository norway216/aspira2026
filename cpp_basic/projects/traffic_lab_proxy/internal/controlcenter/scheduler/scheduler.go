package scheduler

import (
	"context"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// Scheduler is the interface for node selection strategies.
type Scheduler interface {
	// Name returns the strategy name.
	Name() string

	// Select picks the best node from the given list based on current status.
	// Returns nil and an error if no suitable node is available.
	Select(ctx context.Context, nodes []*common.Node, statuses map[string]*common.NodeStatus) (*common.Node, error)
}

// Available strategy names.
const (
	RoundRobin     = "round_robin"
	LeastConn      = "least_connections"
	LeastBw        = "least_bandwidth"
	WeightedRR     = "weighted_round_robin"
	CompositeScore = "composite"
)
