package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tekgeek88/ironlytic-backend/internal/service/auth"
)

const CurrentUserKey = "current_user"

type Authenticator interface {
	Authenticate(ctx any, rawToken string) (any, any, error)
}

type AuthService interface {
	Authenticate(ctx interface{}, rawToken string) (interface{}, interface{}, error)
}

func OptionalAuth(logger *slog.Logger, svc *auth.Service, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := auth.ReadSessionCookie(c.Request, cookieName)
		if !ok {
			c.Next()
			return
		}

		u, _, err := svc.Authenticate(c.Request.Context(), raw)
		if err != nil {
			// ignore and treat as guest; do NOT error the request
			c.Next()
			return
		}

		c.Set(CurrentUserKey, u)
		c.Next()
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get(CurrentUserKey); !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "unauthorized",
					"message": "Authentication required",
				},
			})
			return
		}
		c.Next()
	}
}
