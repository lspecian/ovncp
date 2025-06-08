package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/metrics"
	"go.uber.org/zap"
)

// Recovery middleware with panic metrics
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Record panic metric
				metrics.PanicsTotal.Inc()

				// Get stack trace
				stack := debug.Stack()

				// Log the panic
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("client_ip", c.ClientIP()),
					zap.ByteString("stack", stack),
				)

				// Abort with error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"message": fmt.Sprintf("%v", err),
				})
			}
		}()

		c.Next()
	}
}