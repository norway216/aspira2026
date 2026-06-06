package channel

import (
	"fmt"
	"sync"
)

// ChannelRegistry manages payment channel implementations and routes
// payment instructions to the appropriate channel based on currency
// pair, amount, and availability.
type ChannelRegistry struct {
	channels map[string]PaymentChannel
	mu       sync.RWMutex
}

// NewChannelRegistry creates an empty channel registry.
func NewChannelRegistry() *ChannelRegistry {
	return &ChannelRegistry{
		channels: make(map[string]PaymentChannel),
	}
}

// Register adds a payment channel to the registry.
func (r *ChannelRegistry) Register(ch PaymentChannel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[ch.GetChannelName()] = ch
}

// Get retrieves a channel by name.
func (r *ChannelRegistry) Get(name string) (PaymentChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch, ok := r.channels[name]
	if !ok {
		return nil, fmt.Errorf("channel not found: %s", name)
	}
	return ch, nil
}

// GetHealthy returns the first healthy channel matching a preferred type.
// Falls back to any healthy channel if the preferred one is unavailable.
func (r *ChannelRegistry) GetHealthy(preferred string) (PaymentChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try preferred channel first
	if preferred != "" {
		if ch, ok := r.channels[preferred]; ok && ch.IsHealthy() {
			return ch, nil
		}
	}

	// Fall back to any healthy channel
	for _, ch := range r.channels {
		if ch.IsHealthy() {
			return ch, nil
		}
	}

	return nil, fmt.Errorf("no healthy payment channel available")
}

// List returns all registered channel names.
func (r *ChannelRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.channels))
	for name := range r.channels {
		names = append(names, name)
	}
	return names
}

// HealthStatus returns a map of channel name to health status.
func (r *ChannelRegistry) HealthStatus() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status := make(map[string]bool, len(r.channels))
	for name, ch := range r.channels {
		status[name] = ch.IsHealthy()
	}
	return status
}
