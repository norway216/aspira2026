package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/internal/service"
	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/gin-gonic/gin"
)

// AgentAPI handles agent communication endpoints.
type AgentAPI struct {
	registerService *service.RegisterService
	deviceService   *service.DeviceService
	commandService  *service.CommandService
}

// NewAgentAPI creates a new agent API handler.
func NewAgentAPI(
	registerService *service.RegisterService,
	deviceService *service.DeviceService,
	commandService *service.CommandService,
) *AgentAPI {
	return &AgentAPI{
		registerService: registerService,
		deviceService:   deviceService,
		commandService:  commandService,
	}
}

// Register handles agent registration.
func (a *AgentAPI) Register(c *gin.Context) {
	var req model.AgentRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := a.registerService.RegisterAgent(ctx, &req)
	if err != nil {
		logger.Warn("Agent registration failed",
			logger.String("hostname", req.Hostname),
			logger.ErrField(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Heartbeat handles agent heartbeat.
func (a *AgentAPI) Heartbeat(c *gin.Context) {
	deviceID, _ := c.Get("device_id")

	var req model.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Accept heartbeat with just device ID from header
		req.DeviceID = deviceID.(string)
	}
	req.DeviceID = deviceID.(string)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := a.deviceService.HandleHeartbeat(ctx, &req); err != nil {
		logger.Warn("Heartbeat error",
			logger.String("device_id", req.DeviceID),
			logger.ErrField(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"server_time": time.Now().Unix(),
	})
}

// SubmitMetrics handles agent metric submission.
func (a *AgentAPI) SubmitMetrics(c *gin.Context) {
	deviceID, _ := c.Get("device_id")

	var req model.MetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.DeviceID = deviceID.(string)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := a.deviceService.HandleMetrics(ctx, &req); err != nil {
		logger.Warn("Metrics error",
			logger.String("device_id", req.DeviceID),
			logger.ErrField(err))
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// PullTasks returns pending command tasks for an agent.
func (a *AgentAPI) PullTasks(c *gin.Context) {
	deviceID, _ := c.Get("device_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	task, err := a.commandService.GetPendingTask(ctx, deviceID.(string))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
		return
	}

	// Parse params JSON
	var params map[string]interface{}
	if task.Params != "" {
		json.Unmarshal([]byte(task.Params), &params)
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": []gin.H{
			{
				"task_id": task.TaskID,
				"action":  task.Action,
				"params":  params,
			},
		},
	})
}

// SubmitTaskResult handles the result of a completed command task.
func (a *AgentAPI) SubmitTaskResult(c *gin.Context) {
	taskID := c.Param("task_id")

	var req model.CommandResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.TaskID = taskID

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := a.commandService.HandleTaskResult(ctx, &req); err != nil {
		logger.Warn("Task result error",
			logger.String("task_id", taskID),
			logger.ErrField(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RegisterRoutes registers all agent API routes (unauthenticated and authenticated).
func (a *AgentAPI) RegisterRoutes(rg, authRG *gin.RouterGroup) {
	// Public routes (no agent token required)
	rg.POST("/agents/register", a.Register)

	// Authenticated routes (agent token required)
	authRG.POST("/agents/heartbeat", a.Heartbeat)
	authRG.POST("/agents/metrics", a.SubmitMetrics)
	authRG.GET("/agents/tasks/pull", a.PullTasks)
	authRG.POST("/agents/tasks/:task_id/result", a.SubmitTaskResult)
}