package engine

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EngineClient struct {
	addr    string
	conn    net.Conn
	mu      sync.Mutex
	pending map[string]chan EngineResponse
	readerMu sync.Mutex
	enabled bool
	timeout time.Duration
}

func NewEngineClient(addr string, enabled bool, timeout time.Duration) *EngineClient {
	return &EngineClient{
		addr:    addr,
		enabled: enabled,
		timeout: timeout,
		pending: make(map[string]chan EngineResponse),
	}
}

func (c *EngineClient) Connect() error {
	if !c.enabled {
		return fmt.Errorf("engine disabled")
	}

	conn, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to engine: %w", err)
	}
	c.conn = conn
	go c.readLoop()
	return nil
}

func (c *EngineClient) IsEnabled() bool {
	return c.enabled
}

func (c *EngineClient) IsConnected() bool {
	return c.conn != nil
}

func (c *EngineClient) Send(req EngineRequest) (*EngineResponse, error) {
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respCh := make(chan EngineResponse, 1)
	c.mu.Lock()
	c.pending[req.ID] = respCh
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, req.ID)
		c.mu.Unlock()
	}()

	// Write 4-byte length prefix + JSON
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

	c.mu.Lock()
	if _, err := c.conn.Write(lenBuf); err != nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("write length: %w", err)
	}
	if _, err := c.conn.Write(data); err != nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("write payload: %w", err)
	}
	c.mu.Unlock()

	// Wait for response
	select {
	case resp := <-respCh:
		return &resp, nil
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("engine request timed out")
	}
}

func (c *EngineClient) readLoop() {
	for {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
			return
		}
		length := binary.BigEndian.Uint32(lenBuf)
		payload := make([]byte, length)
		if _, err := io.ReadFull(c.conn, payload); err != nil {
			return
		}

		var resp EngineResponse
		if err := json.Unmarshal(payload, &resp); err != nil {
			continue
		}

		c.mu.Lock()
		ch, ok := c.pending[resp.ID]
		c.mu.Unlock()

		if ok {
			select {
			case ch <- resp:
			default:
			}
		}
	}
}

func (c *EngineClient) HealthCheck() bool {
	if !c.enabled || c.conn == nil {
		return false
	}

	req := EngineRequest{ID: uuid.New().String(), Type: "health_check"}
	resp, err := c.Send(req)
	if err != nil {
		return false
	}
	return resp.Status == "ok"
}

func (c *EngineClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
