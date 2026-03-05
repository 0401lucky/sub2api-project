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
	CORSAllowedOrigins  []string

	JWTSecret string
	JWTExpire time.Duration

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
	cfg := &Config{
		ServerAddr:             getenv("WELFARE_SERVER_ADDR", ":8080"),
		FrontendCallbackURL:    getenv("WELFARE_FRONTEND_CALLBACK_URL", "http://localhost:5173/auth/callback"),
		CookieSecure:           getenvBool("WELFARE_COOKIE_SECURE", false),
		CORSAllowedOrigins:     parseList(getenv("WELFARE_CORS_ALLOWED_ORIGINS", "")),
		JWTSecret:              strings.TrimSpace(getenv("WELFARE_JWT_SECRET", "")),
		JWTExpire:              getenvDuration("WELFARE_JWT_EXPIRE", 24*time.Hour),
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
		Sub2APITimeout:         getenvDuration("SUB2API_TIMEOUT", 10*time.Second),
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

func getenvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
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
