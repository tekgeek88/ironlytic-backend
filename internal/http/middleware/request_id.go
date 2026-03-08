package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, err := uuid.NewV4()
		if err != nil {
			requestID = uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000000"))
		}

		id := requestID.String()
		c.Set(RequestIDKey, id)
		c.Writer.Header().Set("X-Request-ID", id)

		c.Next()
	}
}
