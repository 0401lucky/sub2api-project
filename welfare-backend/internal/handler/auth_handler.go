package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"welfare-backend/internal/middleware"
	"welfare-backend/internal/service"
	"welfare-backend/internal/util"

	"github.com/gin-gonic/gin"
)

const (
	oauthStateCookie    = "wf_oauth_state"
	oauthVerifierCookie = "wf_oauth_verifier"
	oauthRedirectCookie = "wf_oauth_redirect"
)

type AuthHandler struct {
	linuxdo             *service.LinuxDoService
	authService         *service.AuthService
	jwtService          *service.JWTService
	frontendCallbackURL string
	cookieSecure        bool
}

func NewAuthHandler(
	linuxdo *service.LinuxDoService,
	authService *service.AuthService,
	jwtService *service.JWTService,
	frontendCallbackURL string,
	cookieSecure bool,
) *AuthHandler {
	return &AuthHandler{
		linuxdo:             linuxdo,
		authService:         authService,
		jwtService:          jwtService,
		frontendCallbackURL: frontendCallbackURL,
		cookieSecure:        cookieSecure,
	}
}

func (h *AuthHandler) StartLinuxDoOAuth(c *gin.Context) {
	state, err := util.RandomToken(16)
	if err != nil {
		Error(c, http.StatusInternalServerError, "generate oauth state failed")
		return
	}
	verifier, err := util.RandomToken(32)
	if err != nil {
		Error(c, http.StatusInternalServerError, "generate oauth verifier failed")
		return
	}
	challenge := sha256Base64URL(verifier)
	redirectTo := sanitizeRedirect(c.Query("redirect"))

	setCookie(c, oauthStateCookie, state, 600, h.cookieSecure)
	setCookie(c, oauthVerifierCookie, verifier, 600, h.cookieSecure)
	setCookie(c, oauthRedirectCookie, redirectTo, 600, h.cookieSecure)

	authURL, err := h.linuxdo.BuildAuthorizeURL(state, challenge)
	if err != nil {
		Error(c, http.StatusInternalServerError, "build oauth url failed")
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) LinuxDoCallback(c *gin.Context) {
	frontendCB := strings.TrimSpace(h.frontendCallbackURL)
	if frontendCB == "" {
		frontendCB = "/auth/callback"
	}
	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	if state == "" || code == "" {
		h.redirectError(c, frontendCB, "missing_params", "missing oauth code/state")
		return
	}
	cookieState, err := c.Cookie(oauthStateCookie)
	if err != nil || strings.TrimSpace(cookieState) == "" || cookieState != state {
		h.redirectError(c, frontendCB, "invalid_state", "oauth state mismatch")
		return
	}
	verifier, err := c.Cookie(oauthVerifierCookie)
	if err != nil || strings.TrimSpace(verifier) == "" {
		h.redirectError(c, frontendCB, "missing_verifier", "oauth verifier missing")
		return
	}
	redirectTo, _ := c.Cookie(oauthRedirectCookie)
	redirectTo = sanitizeRedirect(redirectTo)

	profile, err := h.linuxdo.Authenticate(c.Request.Context(), code, verifier)
	if err != nil {
		h.redirectError(c, frontendCB, "oauth_failed", err.Error())
		return
	}

	binding, err := h.authService.ResolveAndBindUser(c.Request.Context(), profile.Subject, profile.Username)
	if err != nil {
		if err == service.ErrSub2APIUserNotFound {
			h.redirectError(c, frontendCB, "user_not_found", "请先在 sub2api 完成首次登录/注册")
			return
		}
		h.redirectError(c, frontendCB, "bind_failed", err.Error())
		return
	}

	isAdmin := h.authService.IsAdminSubject(profile.Subject)
	token, expiresAt, err := h.jwtService.Sign(service.AuthClaims{
		LinuxDOSubject: profile.Subject,
		LinuxDOName:    profile.Username,
		Sub2APIUserID:  binding.Sub2APIUserID,
		Sub2APIEmail:   binding.Sub2APIEmail,
		IsAdmin:        isAdmin,
	})
	if err != nil {
		h.redirectError(c, frontendCB, "token_failed", "issue token failed")
		return
	}

	clearCookie(c, oauthStateCookie, h.cookieSecure)
	clearCookie(c, oauthVerifierCookie, h.cookieSecure)
	clearCookie(c, oauthRedirectCookie, h.cookieSecure)

	fragment := url.Values{}
	fragment.Set("access_token", token)
	fragment.Set("expires_at", expiresAt.Format(time.RFC3339))
	fragment.Set("redirect", redirectTo)
	redirectURL := frontendCB + "#" + fragment.Encode()
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) Me(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	Success(c, gin.H{
		"linuxdo_subject": claims.LinuxDOSubject,
		"linuxdo_name":    claims.LinuxDOName,
		"sub2api_user_id": claims.Sub2APIUserID,
		"sub2api_email":   claims.Sub2APIEmail,
		"is_admin":        claims.IsAdmin,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	Success(c, gin.H{"ok": true})
}

func (h *AuthHandler) redirectError(c *gin.Context, callback, code, msg string) {
	fragment := url.Values{}
	fragment.Set("error", code)
	fragment.Set("error_description", msg)
	c.Redirect(http.StatusFound, callback+"#"+fragment.Encode())
}

func sha256Base64URL(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func sanitizeRedirect(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/"
	}
	if strings.HasPrefix(path, "//") || strings.Contains(path, "://") || strings.ContainsAny(path, "\n\r") {
		return "/"
	}
	return path
}

func setCookie(c *gin.Context, name, value string, maxAge int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
