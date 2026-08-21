package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if id == "" {
			buf := make([]byte, 12)
			_, _ = rand.Read(buf)
			id = hex.EncodeToString(buf)
		}
		c.Set(RequestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
func GetRequestID(c *gin.Context) string {
	value, _ := c.Get(RequestIDKey)
	id, _ := value.(string)
	return id
}
