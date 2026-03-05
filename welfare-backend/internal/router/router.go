package router

import (
	"fmt"
	"net/http"
	"strings"

	"welfare-backend/internal/config"
	"welfare-backend/internal/handler"
	"welfare-backend/internal/middleware"
	"welfare-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Auth    *handler.AuthHandler
	Checkin *handler.CheckinHandler
	Admin   *handler.AdminHandler
}

func New(
	cfg *config.Config,
	h Handlers,
	jwtService *service.JWTService,
	authService *service.AuthService,
	revocation *service.TokenRevocationService,
) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors(cfg.CORSAllowedOrigins))
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	{
		auth.GET("/linuxdo/start", h.Auth.StartLinuxDoOAuth)
		auth.GET("/linuxdo/callback", h.Auth.LinuxDoCallback)
	}

	authed := v1.Group("")
	authed.Use(middleware.Auth(jwtService, revocation))
	{
		authed.GET("/auth/me", h.Auth.Me)
		authed.POST("/auth/logout", h.Auth.Logout)

		checkinLimiter := middleware.NewRateLimiter(5.0/60.0, 5, func(c *gin.Context) string {
			claims, ok := middleware.GetClaims(c)
			if !ok {
				return c.ClientIP()
			}
			return fmt.Sprintf("%d:%s", claims.Sub2APIUserID, c.ClientIP())
		})
		authed.GET("/checkin/status", h.Checkin.Status)
		authed.POST("/checkin/daily", checkinLimiter.Middleware(), h.Checkin.Daily)
		authed.GET("/checkin/history", h.Checkin.History)
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.Auth(jwtService, revocation), middleware.AdminOnly(authService))
	{
		admin.GET("/checkin/config", h.Admin.GetCheckinConfig)
		admin.PUT("/checkin/config", h.Admin.UpdateCheckinConfig)
		admin.GET("/checkin/records", h.Admin.ListCheckinRecords)
		admin.GET("/risk/blocks", h.Admin.ListRiskBlocks)
		admin.POST("/risk/blocks", h.Admin.CreateRiskBlock)
		admin.DELETE("/risk/blocks/:id", h.Admin.DeleteRiskBlock)
	}

	return r, nil
}

func cors(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		v := strings.TrimSpace(origin)
		if v != "" {
			allowed[v] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Vary", "Origin")
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
