package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
)

// SagaStep represents a single step in a saga workflow.
// Each step has an Execute function for forward progress and a
// Compensate function for rolling back if a later step fails.
type SagaStep struct {
	Name       string
	Execute    func(ctx context.Context, orderID string, state map[string]interface{}) error
	Compensate func(ctx context.Context, orderID string, state map[string]interface{}) error
}

// Saga defines a complete saga workflow per architecture §8.2.
type Saga struct {
	Name  string
	Steps []SagaStep
	db    database.DB
}

// SagaState tracks the execution state of a saga instance.
type SagaState struct {
	OrderID     string                 `json:"order_id"`
	SagaName    string                 `json:"saga_name"`
	CurrentStep int                    `json:"current_step"`
	StepStatus  string                 `json:"step_status"` // running/compensating/completed/failed
	StepResults map[string]interface{} `json:"step_results"`
}

// Orchestrator manages saga lifecycle and execution.
type Orchestrator struct {
	db    database.DB
	sagas map[string]*Saga
}

// NewOrchestrator creates a new saga orchestrator.
func NewOrchestrator(db database.DB) *Orchestrator {
	return &Orchestrator{
		db:    db,
		sagas: make(map[string]*Saga),
	}
}

// Register adds a saga definition to the orchestrator.
func (o *Orchestrator) Register(saga *Saga) {
	saga.db = o.db
	o.sagas[saga.Name] = saga
}

// Get retrieves a saga by name.
func (o *Orchestrator) Get(name string) (*Saga, error) {
	saga, ok := o.sagas[name]
	if !ok {
		return nil, fmt.Errorf("saga not found: %s", name)
	}
	return saga, nil
}

// Execute runs a saga from start to finish. If any step fails, the
// compensations for all previously completed steps are executed in
// reverse order per architecture §8.3.
func (s *Saga) Execute(ctx context.Context, orderID string) error {
	log.Printf("[Saga:%s] Starting execution for order %s", s.Name, orderID)
	startTime := time.Now()

	state := make(map[string]interface{})
	state["order_id"] = orderID
	state["started_at"] = startTime.Format(time.RFC3339)

	var completedSteps []int

	for i, step := range s.Steps {
		log.Printf("[Saga:%s] Step %d/%d: %s", s.Name, i+1, len(s.Steps), step.Name)

		// Save saga state before each step
		s.saveState(orderID, SagaState{
			OrderID:     orderID,
			SagaName:    s.Name,
			CurrentStep: i,
			StepStatus:  "running",
			StepResults: state,
		})

		// Execute the step
		err := step.Execute(ctx, orderID, state)
		if err != nil {
			log.Printf("[Saga:%s] Step %s FAILED: %v — starting compensation", s.Name, step.Name, err)
			state["error"] = err.Error()
			state["failed_step"] = i
			state["failed_step_name"] = step.Name

			// Compensate in reverse order
			s.compensate(ctx, orderID, completedSteps, state)

			s.saveState(orderID, SagaState{
				OrderID:     orderID,
				SagaName:    s.Name,
				CurrentStep: i,
				StepStatus:  "failed",
				StepResults: state,
			})

			return fmt.Errorf("saga %s failed at step %s: %w", s.Name, step.Name, err)
		}

		completedSteps = append(completedSteps, i)
		state[fmt.Sprintf("step_%d_completed", i)] = true
		state[fmt.Sprintf("step_%d_name", i)] = step.Name
	}

	// All steps completed
	elapsed := time.Since(startTime)
	state["completed_at"] = time.Now().Format(time.RFC3339)
	state["elapsed"] = elapsed.String()

	s.saveState(orderID, SagaState{
		OrderID:     orderID,
		SagaName:    s.Name,
		CurrentStep: len(s.Steps),
		StepStatus:  "completed",
		StepResults: state,
	})

	log.Printf("[Saga:%s] Completed for order %s in %v", s.Name, orderID, elapsed)
	return nil
}

// compensate runs compensation actions in reverse order.
func (s *Saga) compensate(ctx context.Context, orderID string, completedSteps []int, state map[string]interface{}) {
	log.Printf("[Saga:%s] Compensating %d steps for order %s", s.Name, len(completedSteps), orderID)

	s.saveState(orderID, SagaState{
		OrderID:     orderID,
		SagaName:    s.Name,
		CurrentStep: len(completedSteps),
		StepStatus:  "compensating",
		StepResults: state,
	})

	// Compensate in reverse order
	for idx := len(completedSteps) - 1; idx >= 0; idx-- {
		stepIdx := completedSteps[idx]
		step := s.Steps[stepIdx]

		if step.Compensate == nil {
			log.Printf("[Saga:%s] Step %s has no compensation, skipping", s.Name, step.Name)
			continue
		}

		log.Printf("[Saga:%s] Compensating step: %s", s.Name, step.Name)

		err := step.Compensate(ctx, orderID, state)
		if err != nil {
			log.Printf("[Saga:%s] Compensation for step %s FAILED: %v", s.Name, step.Name, err)
			state[fmt.Sprintf("compensate_%d_error", stepIdx)] = err.Error()
			// Continue compensating other steps even if one fails
		} else {
			state[fmt.Sprintf("compensate_%d_completed", stepIdx)] = true
		}
	}

	log.Printf("[Saga:%s] Compensation complete for order %s", s.Name, orderID)
}

// saveState persists the saga state to the database.
func (s *Saga) saveState(orderID string, state SagaState) {
	resultsJSON, _ := json.Marshal(state.StepResults)

	// Upsert saga state
	// The saga_states table is created via migration
	_ = resultsJSON // TODO: persist to saga_states table via DB interface
}

// ListSagas returns all registered saga names.
func (o *Orchestrator) ListSagas() []string {
	names := make([]string, 0, len(o.sagas))
	for name := range o.sagas {
		names = append(names, name)
	}
	return names
}
