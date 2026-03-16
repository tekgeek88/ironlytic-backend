package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tekgeek88/ironlytic-backend/internal/api/middleware"
	"github.com/tekgeek88/ironlytic-backend/internal/domain"
	"github.com/tekgeek88/ironlytic-backend/internal/service/auth"
)

type AuthHandler struct {
	svc *auth.Service
	sc  auth.SessionConfig
}

func NewAuthHandler(svc *auth.Service, sc auth.SessionConfig) *AuthHandler {
	return &AuthHandler{svc: svc, sc: sc}
}

type registerReq struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName *string `json:"displayName"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, "Invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		validationError(c, "Email and password are required")
		return
	}

	ua := c.Request.UserAgent()
	ip := c.ClientIP()

	res, err := h.svc.Register(c.Request.Context(), auth.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		UserAgent:   ptr(ua),
		IP:          ptr(ip),
	})
	if err != nil {
		switch err {
		case auth.ErrEmailAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "email_already_exists", "message": "Email already exists"}})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": "Unexpected error"}})
		}
		return
	}

	auth.SetSessionCookie(c.Writer, h.sc, res.RawToken, res.ExpiresAt)

	c.JSON(http.StatusCreated, gin.H{
		"user": publicUser(res.User),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, "Invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		validationError(c, "Email and password are required")
		return
	}

	ua := c.Request.UserAgent()
	ip := c.ClientIP()

	res, err := h.svc.Login(c.Request.Context(), auth.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: ptr(ua),
		IP:        ptr(ip),
	})
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "invalid_credentials", "message": "Invalid email or password"}})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": "Unexpected error"}})
		}
		return
	}

	auth.SetSessionCookie(c.Writer, h.sc, res.RawToken, res.ExpiresAt)

	c.JSON(http.StatusOK, gin.H{
		"user": publicUser(res.User),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	raw, ok := auth.ReadSessionCookie(c.Request, h.sc.CookieName)
	if ok {
		_ = h.svc.Logout(c.Request.Context(), raw)
	}
	auth.ClearSessionCookie(c.Writer, h.sc)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	// We rely on OptionalAuth middleware
	uAny, ok := c.Get(middleware.CurrentUserKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "Authentication required"}})
		return
	}

	u, _ := uAny.(*domain.User)

	c.JSON(http.StatusOK, gin.H{
		"user": publicUser(u),
		"ts":   time.Now().UTC(),
	})
}

func publicUser(u *domain.User) gin.H {
	if u == nil {
		return gin.H{}
	}
	return gin.H{
		"id":          u.ID.String(),
		"email":       u.Email,
		"displayName": u.DisplayName,
		"createdAt":   u.CreatedAt,
	}
}

func validationError(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{
			"code":    "validation_error",
			"message": msg,
		},
	})
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
