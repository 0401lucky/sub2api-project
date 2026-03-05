package middleware

import (
	"net/http"
	"strings"

	"welfare-backend/internal/service"

	"github.com/gin-gonic/gin"
)

const authClaimsKey = "auth_claims"

func Auth(jwtService *service.JWTService, revocation *service.TokenRevocationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenRaw := strings.TrimSpace(extractBearer(c))
		if tokenRaw == "" {
			tokenRaw = readTokenFromCookie(c)
		}
		if tokenRaw == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "unauthorized"})
			c.Abort()
			return
		}
		claims, err := jwtService.Parse(tokenRaw)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "invalid token"})
			c.Abort()
			return
		}
		if revocation != nil && revocation.IsRevoked(claims.ID) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "token revoked"})
			c.Abort()
			return
		}
		c.Set(authClaimsKey, claims)
		c.Next()
	}
}

func AdminOnly(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "unauthorized"})
			c.Abort()
			return
		}
		if authService == nil || !authService.IsAdminSubject(claims.LinuxDOSubject) {
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

func extractBearer(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func readTokenFromCookie(c *gin.Context) string {
	token, err := c.Cookie("wf_access_token")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(token)
}
