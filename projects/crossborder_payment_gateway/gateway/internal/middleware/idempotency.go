package middleware

import (
	"bytes"
	"fmt"

	"github.com/aspira/crossborder-payment-gateway/internal/database"
	"github.com/gin-gonic/gin"
)

// responseBodyWriter wraps gin.ResponseWriter to capture the response body.
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Idempotency creates middleware that prevents duplicate request processing.
// It checks the X-Request-Id and X-Merchant-Id headers against the
// idempotency_keys table. If a cached response exists for the key, it is
// returned immediately without re-executing the handler.
//
// Per architecture §5.1: every request must include request_id and merchant_id
// for idempotency. The key is "merchant_id:request_id".
func Idempotency(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		merchantID := c.GetHeader("X-Merchant-Id")

		// Skip idempotency check if headers are not present (e.g. dashboard requests)
		if requestID == "" || merchantID == "" {
			c.Next()
			return
		}

		key := fmt.Sprintf("%s:%s", merchantID, requestID)

		// Check if we've already processed this request
		cachedBody, cachedStatus, err := db.GetIdempotencyKey(key)
		if err == nil {
			// Key exists — return cached response
			c.Header("X-Idempotency-Replayed", "true")
			c.Data(cachedStatus, "application/json", []byte(cachedBody))
			c.Abort()
			return
		}

		// Wrap response writer to capture the response
		writer := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		// After handler completes successfully, store the response for future idempotency
		if c.Writer.Status() < 500 {
			// Only cache successful responses (2xx, 3xx, 4xx)
			_ = db.CreateIdempotencyKey(key, writer.body.String(), c.Writer.Status())
		}
	}
}

// IdempotencyRequired is an alias for Idempotency with a clearer name for route groups.
var IdempotencyRequired = Idempotency
