package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultLinuxDoDomain = "linuxdo-connect.invalid"

type Config struct {
	ServerAddr          string
	FrontendCallbackURL string
	CookieSecure        bool
	CookieSameSite      string
	CORSAllowedOrigins  []string
	TrustedProxies      []string

	JWTSecret   string
	JWTExpire   time.Duration
	JWTIssuer   string
	JWTAudience string

	DatabaseDriver string
	DatabaseDSN    string

	LinuxDoClientID      string
	LinuxDoClientSecret  string
	LinuxDoAuthorizeURL  string
	LinuxDoTokenURL      string
	LinuxDoUserInfoURL   string
	LinuxDoScopes        string
	LinuxDoRedirectURL   string
	LinuxDoUserIDField   string
	LinuxDoUserNameField string

	Sub2APIBaseURL         string
	Sub2APIAdminAPIKey     string
	Sub2APITimeout         time.Duration
	Sub2APISyntheticDomain string

	AdminSubjects   map[string]struct{}
	CheckinTimezone string
}

func Load() (*Config, error) {
	cookieSecure, err := getenvBool("WELFARE_COOKIE_SECURE", false)
	if err != nil {
		return nil, err
	}
	jwtExpire, err := getenvDuration("WELFARE_JWT_EXPIRE", 24*time.Hour)
	if err != nil {
		return nil, err
	}
	sub2apiTimeout, err := getenvDuration("SUB2API_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		ServerAddr:             getenv("WELFARE_SERVER_ADDR", ":8080"),
		FrontendCallbackURL:    getenv("WELFARE_FRONTEND_CALLBACK_URL", "http://localhost:5173/auth/callback"),
		CookieSecure:           cookieSecure,
		CookieSameSite:         strings.ToLower(strings.TrimSpace(getenv("WELFARE_COOKIE_SAMESITE", "lax"))),
		CORSAllowedOrigins:     parseList(getenv("WELFARE_CORS_ALLOWED_ORIGINS", "")),
		TrustedProxies:         parseList(getenv("WELFARE_TRUSTED_PROXIES", "")),
		JWTSecret:              strings.TrimSpace(getenv("WELFARE_JWT_SECRET", "")),
		JWTExpire:              jwtExpire,
		JWTIssuer:              strings.TrimSpace(getenv("WELFARE_JWT_ISSUER", "welfare-backend")),
		JWTAudience:            strings.TrimSpace(getenv("WELFARE_JWT_AUDIENCE", "welfare-frontend")),
		DatabaseDriver:         strings.ToLower(strings.TrimSpace(getenv("WELFARE_DATABASE_DRIVER", "sqlite"))),
		DatabaseDSN:            strings.TrimSpace(getenv("WELFARE_DATABASE_DSN", "welfare.db")),
		LinuxDoClientID:        strings.TrimSpace(getenv("LINUXDO_CLIENT_ID", "")),
		LinuxDoClientSecret:    strings.TrimSpace(getenv("LINUXDO_CLIENT_SECRET", "")),
		LinuxDoAuthorizeURL:    strings.TrimSpace(getenv("LINUXDO_AUTHORIZE_URL", "https://connect.linux.do/oauth2/authorize")),
		LinuxDoTokenURL:        strings.TrimSpace(getenv("LINUXDO_TOKEN_URL", "https://connect.linux.do/oauth2/token")),
		LinuxDoUserInfoURL:     strings.TrimSpace(getenv("LINUXDO_USERINFO_URL", "https://connect.linux.do/api/user")),
		LinuxDoScopes:          strings.TrimSpace(getenv("LINUXDO_SCOPES", "openid profile email")),
		LinuxDoRedirectURL:     strings.TrimSpace(getenv("LINUXDO_REDIRECT_URL", "")),
		LinuxDoUserIDField:     strings.TrimSpace(getenv("LINUXDO_USERINFO_ID_FIELD", "id")),
		LinuxDoUserNameField:   strings.TrimSpace(getenv("LINUXDO_USERINFO_USERNAME_FIELD", "username")),
		Sub2APIBaseURL:         strings.TrimRight(strings.TrimSpace(getenv("SUB2API_BASE_URL", "")), "/"),
		Sub2APIAdminAPIKey:     strings.TrimSpace(getenv("SUB2API_ADMIN_API_KEY", "")),
		Sub2APITimeout:         sub2apiTimeout,
		Sub2APISyntheticDomain: strings.TrimSpace(getenv("SUB2API_SYNTHETIC_DOMAIN", defaultLinuxDoDomain)),
		AdminSubjects:          parseSet(getenv("WELFARE_ADMIN_SUBJECTS", "")),
		CheckinTimezone:        strings.TrimSpace(getenv("WELFARE_CHECKIN_TIMEZONE", "Asia/Shanghai")),
	}

	if cfg.JWTSecret == "" {
		return nil, errors.New("WELFARE_JWT_SECRET is required")
	}
	if cfg.LinuxDoClientID == "" || cfg.LinuxDoClientSecret == "" || cfg.LinuxDoRedirectURL == "" {
		return nil, errors.New("LINUXDO_CLIENT_ID, LINUXDO_CLIENT_SECRET, LINUXDO_REDIRECT_URL are required")
	}
	if cfg.Sub2APIBaseURL == "" || cfg.Sub2APIAdminAPIKey == "" {
		return nil, errors.New("SUB2API_BASE_URL and SUB2API_ADMIN_API_KEY are required")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return nil, errors.New("WELFARE_CORS_ALLOWED_ORIGINS is required")
	}
	switch cfg.CookieSameSite {
	case "lax", "strict", "none":
	default:
		return nil, errors.New("WELFARE_COOKIE_SAMESITE must be one of: lax, strict, none")
	}
	if cfg.CookieSameSite == "none" && !cfg.CookieSecure {
		return nil, errors.New("WELFARE_COOKIE_SECURE must be true when WELFARE_COOKIE_SAMESITE=none")
	}
	if cfg.JWTExpire <= 0 {
		return nil, errors.New("WELFARE_JWT_EXPIRE must be greater than 0")
	}
	if cfg.Sub2APITimeout <= 0 {
		return nil, errors.New("SUB2API_TIMEOUT must be greater than 0")
	}
	if cfg.JWTIssuer == "" {
		return nil, errors.New("WELFARE_JWT_ISSUER is required")
	}
	if cfg.JWTAudience == "" {
		return nil, errors.New("WELFARE_JWT_AUDIENCE is required")
	}
	if _, err := time.LoadLocation(cfg.CheckinTimezone); err != nil {
		return nil, fmt.Errorf("invalid WELFARE_CHECKIN_TIMEZONE: %w", err)
	}
	if cfg.DatabaseDriver != "sqlite" && cfg.DatabaseDriver != "postgres" {
		return nil, errors.New("WELFARE_DATABASE_DRIVER must be sqlite or postgres")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getenvBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a valid bool: %w", key, err)
	}
	return v, nil
}

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	return d, nil
}

func parseSet(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}

func parseList(raw string) []string {
	out := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
