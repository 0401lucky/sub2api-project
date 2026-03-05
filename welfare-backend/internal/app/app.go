package app

import (
	"net/http"

	"welfare-backend/internal/config"
	"welfare-backend/internal/database"
	"welfare-backend/internal/handler"
	"welfare-backend/internal/router"
	"welfare-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func Build() (*gin.Engine, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	db, err := database.New(cfg)
	if err != nil {
		return nil, nil, err
	}

	sub2api := service.NewSub2APIClient(cfg.Sub2APIBaseURL, cfg.Sub2APIAdminAPIKey, cfg.Sub2APITimeout)
	jwtService := service.NewJWTService(cfg.JWTSecret, cfg.JWTExpire, cfg.JWTIssuer, cfg.JWTAudience)
	revocationService := service.NewTokenRevocationService()
	authService := service.NewAuthService(db, sub2api, cfg.Sub2APISyntheticDomain, cfg.AdminSubjects)
	checkinService := service.NewCheckinService(db, sub2api)
	linuxdoService := service.NewLinuxDoService(
		cfg.LinuxDoClientID,
		cfg.LinuxDoClientSecret,
		cfg.LinuxDoAuthorizeURL,
		cfg.LinuxDoTokenURL,
		cfg.LinuxDoUserInfoURL,
		cfg.LinuxDoScopes,
		cfg.LinuxDoRedirectURL,
		cfg.LinuxDoUserIDField,
		cfg.LinuxDoUserNameField,
		&http.Client{Timeout: cfg.Sub2APITimeout},
	)

	authHandler := handler.NewAuthHandler(
		linuxdoService,
		authService,
		jwtService,
		revocationService,
		cfg.FrontendCallbackURL,
		cfg.CookieSecure,
		cfg.JWTExpire,
	)
	checkinHandler := handler.NewCheckinHandler(checkinService)
	adminHandler := handler.NewAdminHandler(checkinService)

	r, err := router.New(cfg, router.Handlers{
		Auth:    authHandler,
		Checkin: checkinHandler,
		Admin:   adminHandler,
	}, jwtService, authService, revocationService)
	if err != nil {
		return nil, nil, err
	}
	return r, cfg, nil
}
