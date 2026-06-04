package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/internal/repository"
	"github.com/embedded-ops-platform/backend/internal/ws"
	"github.com/embedded-ops-platform/backend/pkg/crypto"
	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/google/uuid"
)

// CommandAction defines a whitelisted command action.
type CommandAction struct {
	Name        string              `json:"name"`
	RiskLevel   string              `json:"risk_level"` // low, medium, high
	Description string              `json:"description"`
	Params      map[string]ParamRule `json:"params"`
}

// ParamRule defines validation rules for command parameters.
type ParamRule struct {
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Min      float64  `json:"min"`
	Max      float64  `json:"max"`
	Allowed  []string `json:"allowed"`
}

// CommandService handles remote command execution logic.
type CommandService struct {
	db       *repository.PostgresRepo
	hub      *ws.Hub
	actions  map[string]CommandAction
}

// NewCommandService creates a new command service with a built-in whitelist.
func NewCommandService(db *repository.PostgresRepo, hub *ws.Hub) *CommandService {
	cs := &CommandService{
		db:  db,
		hub: hub,
		actions: map[string]CommandAction{
			"get_basic_info": {
				Name:        "get_basic_info",
				RiskLevel:   "low",
				Description: "Read basic system information",
			},
			"get_dmesg": {
				Name:        "get_dmesg",
				RiskLevel:   "low",
				Description: "Get kernel log (dmesg)",
			},
			"get_usb_devices": {
				Name:        "get_usb_devices",
				RiskLevel:   "low",
				Description: "List USB devices",
			},
			"get_audio_status": {
				Name:        "get_audio_status",
				RiskLevel:   "low",
				Description: "Get audio/sound card status",
			},
			"set_audio_volume": {
				Name:        "set_audio_volume",
				RiskLevel:   "medium",
				Description: "Set audio volume",
				Params: map[string]ParamRule{
					"card": {
						Type:     "number",
						Required: true,
						Min:      0,
						Max:      8,
					},
					"control": {
						Type:     "string",
						Required: true,
						Allowed:  []string{"HP", "Speaker", "Headphone", "Master", "PCM"},
					},
					"value": {
						Type:     "number",
						Required: true,
						Min:      0,
						Max:      100,
					},
				},
			},
			"get_gpu_status": {
				Name:        "get_gpu_status",
				RiskLevel:   "low",
				Description: "Get GPU/DRI/Mali status",
			},
			"restart_app": {
				Name:        "restart_app",
				RiskLevel:   "medium",
				Description: "Restart specified application service",
				Params: map[string]ParamRule{
					"service_name": {
						Type:     "string",
						Required: true,
						Allowed:  []string{"agent", "networking", "audio", "display"},
					},
				},
			},
			"reboot_device": {
				Name:        "reboot_device",
				RiskLevel:   "high",
				Description: "Reboot the device (disabled by default)",
			},
			"upload_log_bundle": {
				Name:        "upload_log_bundle",
				RiskLevel:   "medium",
				Description: "Package and upload device logs",
			},
			"get_wifi_status": {
				Name:        "get_wifi_status",
				RiskLevel:   "low",
				Description: "Get WiFi module status",
			},
		},
	}
	return cs
}

// ListAllowedActions returns the list of whitelisted command actions.
func (s *CommandService) ListAllowedActions() []CommandAction {
	actions := make([]CommandAction, 0, len(s.actions))
	for _, a := range s.actions {
		actions = append(actions, a)
	}
	return actions
}

// CreateCommand creates a new command task for a device.
func (s *CommandService) CreateCommand(ctx context.Context, deviceID string, req *model.CommandRequest, userID string) (*model.CommandTask, error) {
	// Check if action is in whitelist
	action, ok := s.actions[req.Action]
	if !ok {
		return nil, fmt.Errorf("unknown action '%s'", req.Action)
	}

	// Validate params
	if req.Params != nil {
		for paramName, paramVal := range req.Params {
			rules, hasRule := action.Params[paramName]
			if !hasRule {
				return nil, fmt.Errorf("unknown parameter '%s' for action '%s'", paramName, req.Action)
			}
			if rules.Type == "number" {
				val, ok := paramVal.(float64)
				if !ok {
					return nil, fmt.Errorf("parameter '%s' must be a number", paramName)
				}
				if val < rules.Min || val > rules.Max {
					return nil, fmt.Errorf("parameter '%s' out of range [%f, %f]", paramName, rules.Min, rules.Max)
				}
			}
			if rules.Type == "string" {
				val, ok := paramVal.(string)
				if !ok {
					return nil, fmt.Errorf("parameter '%s' must be a string", paramName)
				}
				if len(rules.Allowed) > 0 {
					allowed := false
					for _, a := range rules.Allowed {
						if val == a {
							allowed = true
							break
						}
					}
					if !allowed {
						return nil, fmt.Errorf("parameter '%s' value '%s' not in allowed list %v", paramName, val, rules.Allowed)
					}
				}
			}
		}
	}

	// Create task
	taskID, err := crypto.GenerateToken(16)
	if err != nil {
		taskID = uuid.New().String()
	}

	paramsJSON, _ := json.Marshal(req.Params)
	task := &model.CommandTask{
		TaskID:    taskID,
		DeviceID:  deviceID,
		Action:    req.Action,
		Params:    string(paramsJSON),
		Status:    "pending",
		CreatedBy: userID,
	}

	if err := s.db.CreateCommandTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create command task: %w", err)
	}

	logger.Info("Command created",
		logger.String("task_id", taskID),
		logger.String("device_id", deviceID),
		logger.String("action", req.Action),
		logger.String("created_by", userID))

	// Notify frontend
	s.hub.Broadcast(ws.MsgTypeCommand, map[string]interface{}{
		"task_id":   taskID,
		"device_id": deviceID,
		"action":    req.Action,
		"status":    "pending",
	})

	return task, nil
}

// GetPendingTask returns the next pending task for an agent.
func (s *CommandService) GetPendingTask(ctx context.Context, deviceID string) (*model.CommandTask, error) {
	task, err := s.db.GetPendingTask(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// Mark as running
	if err := s.db.SetTaskRunning(ctx, task.TaskID); err != nil {
		return nil, fmt.Errorf("failed to mark task as running: %w", err)
	}

	return task, nil
}

// HandleTaskResult processes the result of a completed command task.
func (s *CommandService) HandleTaskResult(ctx context.Context, req *model.CommandResultRequest) error {
	status := req.Status
	if status == "" {
		if req.ExitCode == 0 {
			status = "completed"
		} else {
			status = "failed"
		}
	}

	if err := s.db.UpdateCommandTask(ctx, req.TaskID, status, req.Stdout, req.Stderr, req.ExitCode); err != nil {
		return fmt.Errorf("failed to update command task: %w", err)
	}

	// Broadcast result to frontend
	s.hub.Broadcast(ws.MsgTypeCommandResult, map[string]interface{}{
		"task_id":   req.TaskID,
		"stdout":    req.Stdout,
		"stderr":    req.Stderr,
		"exit_code": req.ExitCode,
		"status":    status,
	})

	logger.Info("Command result received",
		logger.String("task_id", req.TaskID),
		logger.String("status", status),
		logger.Int("exit_code", req.ExitCode))

	return nil
}

// GetDeviceCommands returns command history for a device.
func (s *CommandService) GetDeviceCommands(ctx context.Context, deviceID string, limit int) ([]*model.CommandTask, error) {
	return s.db.ListDeviceCommands(ctx, deviceID, limit)
}