package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/internal/repository"
	"github.com/embedded-ops-platform/backend/pkg/config"
	"github.com/embedded-ops-platform/backend/pkg/crypto"
	"github.com/embedded-ops-platform/backend/pkg/logger"
)

// RegisterService handles agent registration logic.
type RegisterService struct {
	db     *repository.PostgresRepo
	cfg    config.AgentConfig
	secret string // HMAC secret for token hashing
}

// NewRegisterService creates a new register service.
func NewRegisterService(db *repository.PostgresRepo, jwtSecret string) *RegisterService {
	return &RegisterService{
		db:     db,
		secret: jwtSecret,
	}
}

// RegisterAgent handles the agent registration flow.
func (s *RegisterService) RegisterAgent(ctx context.Context, req *model.AgentRegisterRequest) (*model.AgentRegisterResponse, error) {
	// Validate register_code (in production, check against DB of valid codes)
	if len(req.RegisterCode) < 6 {
		return nil, fmt.Errorf("invalid register code")
	}

	// Generate device ID
	boardType := req.BoardType
	if boardType == "" {
		boardType = "generic"
	}
	deviceID, err := crypto.GenerateDeviceID(boardType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate device ID: %w", err)
	}

	// Generate agent token
	agentToken, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent token: %w", err)
	}

	// Hash the token for storage
	tokenHash := crypto.HashToken(agentToken, s.secret)

	// Create device record
	macStr := strings.Join(req.MACAddresses, ",")
	device := &model.Device{
		DeviceID:      deviceID,
		DeviceName:    req.Hostname,
		Hostname:      req.Hostname,
		MACAddress:    macStr,
		BoardType:     req.BoardType,
		Arch:          req.Arch,
		OSName:        req.OSName,
		KernelVersion: req.KernelVersion,
		AgentVersion:  req.AgentVersion,
		Status:        "online",
		LastSeen:      time.Now(),
	}

	if err := s.db.CreateDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// Store credential
	cred := &model.AgentCredential{
		DeviceID:  deviceID,
		TokenHash: tokenHash,
		Enabled:   true,
	}
	if err := s.db.CreateAgentCredential(ctx, cred); err != nil {
		logger.Warn("Failed to store agent credential",
			logger.String("device_id", deviceID),
			logger.ErrField(err))
	}

	logger.Info("Agent registered",
		logger.String("device_id", deviceID),
		logger.String("hostname", req.Hostname),
		logger.String("board_type", req.BoardType))

	return &model.AgentRegisterResponse{
		DeviceID:      deviceID,
		AgentToken:    agentToken,
		ServerTime:    time.Now().Unix(),
		ConfigVersion: 1,
	}, nil
}

// ValidateAgentToken checks if an agent's token is valid.
func (s *RegisterService) ValidateAgentToken(deviceID, tokenStr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cred, err := s.db.GetAgentCredential(ctx, deviceID)
	if err != nil {
		return false
	}

	if !cred.Enabled {
		return false
	}

	return crypto.VerifyToken(tokenStr, cred.TokenHash, s.secret)
}