package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/internal/service"
	"github.com/embedded-ops-platform/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// CommandAPI handles user-facing command endpoints.
type CommandAPI struct {
	commandService *service.CommandService
}

// NewCommandAPI creates a new command API handler.
func NewCommandAPI(commandService *service.CommandService) *CommandAPI {
	return &CommandAPI{commandService: commandService}
}

// CreateCommand creates a new remote command for a device.
func (a *CommandAPI) CreateCommand(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req model.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	user := middleware.GetUserFromContext(c)
	userID := user.Username

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	task, err := a.commandService.CreateCommand(ctx, deviceID, &req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// ListCommands returns command history for a device.
func (a *CommandAPI) ListCommands(c *gin.Context) {
	deviceID := c.Param("device_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tasks, err := a.commandService.GetDeviceCommands(ctx, deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"commands": tasks,
		"limit":    limit,
	})
}

// ListActions returns the whitelist of allowed command actions.
func (a *CommandAPI) ListActions(c *gin.Context) {
	actions := a.commandService.ListAllowedActions()
	c.JSON(http.StatusOK, gin.H{"actions": actions})
}

// RegisterRoutes registers all command API routes.
func (a *CommandAPI) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/commands/actions", a.ListActions)
	rg.GET("/devices/:device_id/commands", a.ListCommands)
	rg.POST("/devices/:device_id/commands", a.CreateCommand)
}