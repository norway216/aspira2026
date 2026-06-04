package register

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/embedded-ops-platform/agent/internal/config"
)

// RegisterResult holds the result of the registration process.
type RegisterResult struct {
	Success bool
	Identity *config.AgentIdentity
}

// Register sends a registration request to the server.
func Register(serverURL, registerCode string, info map[string]interface{}) (*config.AgentIdentity, error) {
	payload := map[string]interface{}{
		"register_code":  registerCode,
		"hostname":       info["hostname"],
		"machine_id":     info["machine_id"],
		"mac_addresses":  info["mac_addresses"],
		"arch":           info["arch"],
		"os_name":        info["os_name"],
		"kernel_version": info["kernel_version"],
		"board_type":     info["board_type"],
		"agent_version":  info["agent_version"],
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal register request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/agents/register", serverURL)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to send register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	identity := &config.AgentIdentity{}
	if err := json.NewDecoder(resp.Body).Decode(identity); err != nil {
		return nil, fmt.Errorf("failed to decode register response: %w", err)
	}

	return identity, nil
}

// TryRegister attempts registration with retry logic.
func TryRegister(cfg *config.Config, info map[string]interface{}) (*RegisterResult, error) {
	identity, err := Register(cfg.Server.URL, cfg.Agent.RegisterCode, info)
	if err != nil {
		return &RegisterResult{Success: false}, err
	}

	// Save identity
	if err := config.SaveIdentity(cfg.Security.TokenFile, identity); err != nil {
		return &RegisterResult{Success: false, Identity: identity},
			fmt.Errorf("failed to save identity: %w", err)
	}

	return &RegisterResult{Success: true, Identity: identity}, nil
}