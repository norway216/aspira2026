package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
)

var (
	SchedulerEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "control_scheduler_events_total",
		Help: "Total number of scheduler events by strategy",
	}, []string{"strategy"})
)

// Engine manages scheduling strategies and node selection.
type Engine struct {
	db         db.DB
	schedulers map[string]Scheduler
	current    Scheduler
	mu         sync.RWMutex
}

// NewEngine creates a scheduler engine with all strategies registered.
func NewEngine(database db.DB, defaultStrategy string) *Engine {
	strategies := map[string]Scheduler{
		RoundRobin:     NewRoundRobinScheduler(),
		LeastConn:      NewLeastConnScheduler(),
		LeastBw:        NewLeastBwScheduler(),
		WeightedRR:     NewWeightedRRScheduler(),
		CompositeScore: NewCompositeScheduler(),
	}

	current, ok := strategies[defaultStrategy]
	if !ok {
		current = strategies[CompositeScore]
	}

	return &Engine{
		db:         database,
		schedulers: strategies,
		current:    current,
	}
}

// SelectNode fetches available nodes and their statuses, then delegates to the current strategy.
func (e *Engine) SelectNode(ctx context.Context, userID string) (*common.Node, error) {
	e.mu.RLock()
	strategy := e.current
	e.mu.RUnlock()

	// Fetch online nodes
	nodes, err := e.db.ListOnlineNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no online nodes available")
	}

	// Fetch latest statuses
	statuses, err := e.db.GetLatestNodeStatuses(ctx)
	if err != nil {
		// Continue without statuses; strategies handle nil gracefully
		statuses = make(map[string]*common.NodeStatus)
	}

	// Select node
	node, err := strategy.Select(ctx, nodes, statuses)
	if err != nil {
		// Log failed scheduler event
		_ = e.db.InsertSchedulerEvent(ctx, &common.SchedulerEvent{
			UserID:   userID,
			Strategy: strategy.Name(),
			Reason:   fmt.Sprintf("selection failed: %v", err),
		})
		return nil, err
	}

	// Record scheduler event
	reason := fmt.Sprintf("strategy=%s selected node=%s", strategy.Name(), node.NodeID)
	if ns, ok := statuses[node.NodeID]; ok {
		reason += fmt.Sprintf(" cpu=%.1f mem=%.1f conns=%d rx=%.1f tx=%.1f",
			ns.CPUUsage, ns.MemoryUsage, ns.CurrentConnections, ns.RxMbps, ns.TxMbps)
	}
	_ = e.db.InsertSchedulerEvent(ctx, &common.SchedulerEvent{
		UserID:         userID,
		SelectedNodeID: node.NodeID,
		Strategy:       strategy.Name(),
		Reason:         reason,
	})

	// Update Prometheus metric
	SchedulerEventsTotal.WithLabelValues(strategy.Name()).Inc()

	return node, nil
}

// CurrentStrategy returns the name of the currently active strategy.
func (e *Engine) CurrentStrategy() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current.Name()
}

// SetStrategy changes the active scheduling strategy.
func (e *Engine) SetStrategy(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	s, ok := e.schedulers[name]
	if !ok {
		return fmt.Errorf("unknown strategy: %s (available: %v)", name, e.availableNames())
	}
	e.current = s
	return nil
}

// ListEvents returns recent scheduler events from the database.
func (e *Engine) ListEvents(ctx context.Context, limit int) ([]*common.SchedulerEvent, error) {
	return e.db.ListSchedulerEvents(ctx, limit)
}

func (e *Engine) availableNames() []string {
	names := make([]string, 0, len(e.schedulers))
	for n := range e.schedulers {
		names = append(names, n)
	}
	return names
}
