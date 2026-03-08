package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestID, _ := c.Get(RequestIDKey)

		logger.Error("panic recovered",
			"request_id", requestID,
			"error", recovered,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "An unexpected error occurred",
			},
		})
	})
}
