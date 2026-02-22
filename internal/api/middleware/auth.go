package middleware

import (
	"net/http"
	"strings"

	"github.com/alligatorO15/fin-tracker/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	UserIDKey = "user_id"
	EmailKey  = "email"
)

func Auth(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)
		c.Next()

	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return uuid.Nil
	}

	return userID.(uuid.UUID)
}
