package handlers

import (
	"net/http"

	"github.com/alligatorO15/fin-tracker/internal/api/middleware"
	"github.com/alligatorO15/fin-tracker/internal/config"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	refreshTokenCookie = "refresh_token"
	refreshTokenPath   = "/api/auth"
)

type AuthHandler struct {
	authService service.AuthService
	config      *config.Config
}

func NewAuthHandler(authService service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		config:      cfg,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input models.UserRegistration
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	response, err := h.authService.Register(c.Request.Context(), &input)
	if err != nil {
		if err == service.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	h.setRefreshTokenCookie(c, response.RefreshToken)
	response.RefreshToken = "" //затираем из json ответа

	c.JSON(http.StatusCreated, response)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input models.UserLogin
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, err := h.authService.Login(c.Request.Context(), &input)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.setRefreshTokenCookie(c, response.RefreshToken)
	response.RefreshToken = ""

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	// берем refersh token из httpOnly cookie
	refreshToken, err := c.Cookie(refreshTokenCookie)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found"})
		return
	}

	response, err := h.authService.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		if err == service.ErrInvalidCredentials || err == service.ErrTokenExpired {
			h.clearRefreshTokenCookie(c)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.setRefreshTokenCookie(c, response.RefreshToken)
	response.RefreshToken = ""

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshTokenCookie)
	if err == nil && refreshToken != "" {
		_ = h.authService.Logout(c.Request.Context(), refreshToken)
	}

	h.clearRefreshTokenCookie(c)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out from all devices"})
}

// устанавливает refresh token в httpOnly cookie
func (h *AuthHandler) setRefreshTokenCookie(c *gin.Context, token string) {
	secure := h.config.Env == "production"
	maxAge := int(h.config.RefreshTokenExpiration.Seconds())

	c.SetCookie(
		refreshTokenCookie, // имя
		token,              // сам refersh token
		maxAge,             // в сек когда истечет
		refreshTokenPath,   // путь для которого браузер будет отправлять куки с токеном
		"",                 // "" - текущий домен
		secure,             // true - куки передается по https, а не по http
		true,               // httpOnly (недоступен из javascript)
	)
}

// удаляет refresh token cookie
func (h *AuthHandler) clearRefreshTokenCookie(c *gin.Context) {
	secure := h.config.Env == "production"

	c.SetCookie(
		refreshTokenCookie,
		"",
		-1,
		refreshTokenPath,
		"",
		secure,
		true,
	)
}
