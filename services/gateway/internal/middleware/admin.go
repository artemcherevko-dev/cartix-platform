package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRoles allows the request only when the JWT role is one of the given
// roles. Role comes from the Auth middleware (c.GetString("role")).
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}

	return func(c *gin.Context) {
		if !allowed[c.GetString("role")] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// Admin allows only the admin role.
func Admin() gin.HandlerFunc {
	return RequireRoles("admin")
}
