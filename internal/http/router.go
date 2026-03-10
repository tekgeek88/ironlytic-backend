package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tekgeek88/ironlytic-backend/internal/auth"
	"github.com/tekgeek88/ironlytic-backend/internal/platform/db"
	"github.com/tekgeek88/ironlytic-backend/internal/store"

	"github.com/tekgeek88/ironlytic-backend/internal/config"
	"github.com/tekgeek88/ironlytic-backend/internal/http/handlers"
	"github.com/tekgeek88/ironlytic-backend/internal/http/middleware"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

func NewRouter(cfg config.Config, logger *slog.Logger, pg *db.Postgres) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	httpLogger := logger.With("component", "http")
	router := gin.New()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(httpLogger))
	router.Use(middleware.Recovery(httpLogger))

	healthHandler := handlers.NewHealthHandler(pg)

	// Auth wiring
	userStore := store.NewUserStore(pg.Pool)
	sessionStore := store.NewSessionStore(pg.Pool)

	sameSite := http.SameSiteLaxMode
	if cfg.CookieSameSite == "Strict" {
		sameSite = http.SameSiteStrictMode
	} else if cfg.CookieSameSite == "None" {
		sameSite = http.SameSiteNoneMode
	}

	sessionCfg := auth.SessionConfig{
		CookieName: cfg.SessionCookieName,
		TTL:        time.Duration(cfg.SessionTTLHours) * time.Hour,
		Secure:     cfg.CookieSecure,
		SameSite:   sameSite,
	}

	authSvc := auth.NewService(userStore, sessionStore, sessionCfg)
	authHandler := handlers.NewAuthHandler(authSvc, sessionCfg)

	// Public
	router.GET("/healthz", healthHandler.Healthz)
	router.GET("/readyz", healthHandler.Readyz)

	api := router.Group("/api/v1")
	api.Use(middleware.OptionalAuth(logger, authSvc, sessionCfg.CookieName))
	{
		api.GET("/health", healthHandler.Healthz)

		authGroup := api.Group("/auth")
		authGroup.Use(middleware.OptionalAuth(logger, authSvc, sessionCfg.CookieName))
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/logout", authHandler.Logout)
			authGroup.GET("/me", authHandler.Me)
		}
	}

	app := api.Group("")
	app.Use(middleware.RequireAuth())
	// app.GET("/routines", ...)

	return router
}
