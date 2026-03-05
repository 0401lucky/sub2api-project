package middleware

import (
	"net/http"
	"strings"

	"welfare-backend/internal/service"

	"github.com/gin-gonic/gin"
)

const authClaimsKey = "auth_claims"

func Auth(jwtService *service.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "missing authorization header"})
			c.Abort()
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "invalid authorization header"})
			c.Abort()
			return
		}
		claims, err := jwtService.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "invalid token"})
			c.Abort()
			return
		}
		c.Set(authClaimsKey, claims)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "unauthorized"})
			c.Abort()
			return
		}
		if !claims.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "admin only"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetClaims(c *gin.Context) (*service.AuthClaims, bool) {
	v, ok := c.Get(authClaimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*service.AuthClaims)
	return claims, ok
}
