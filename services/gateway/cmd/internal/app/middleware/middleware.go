package middleware

import (
	"gateway/cmd/internal/app/jwt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		access, err := c.Cookie("access_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no access token"})
			return
		}

		claims, err := jwt.ValidateAccessToken(access)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired access token",
			})
			return
		}

		c.Set("user_id", claims.Subject)
		c.Set("role", claims.Role)

		c.Next()
	}
}
