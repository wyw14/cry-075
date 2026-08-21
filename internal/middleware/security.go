package middleware

import "github.com/gin-gonic/gin"

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'")
		c.Next()
	}
}

func CORS(allowed []string) gin.HandlerFunc {
	set := map[string]bool{}
	for _, origin := range allowed {
		set[origin] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if set[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key,X-Request-ID,X-Actor-ID,X-Actor-Role")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
