package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"log"
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
	accessTokenCookie   = "wf_access_token"
)

type AuthHandler struct {
	linuxdo             *service.LinuxDoService
	authService         *service.AuthService
	jwtService          *service.JWTService
	revocationService   *service.TokenRevocationService
	frontendCallbackURL string
	cookieSecure        bool
	jwtExpire           time.Duration
}

func NewAuthHandler(
	linuxdo *service.LinuxDoService,
	authService *service.AuthService,
	jwtService *service.JWTService,
	revocationService *service.TokenRevocationService,
	frontendCallbackURL string,
	cookieSecure bool,
	jwtExpire time.Duration,
) *AuthHandler {
	return &AuthHandler{
		linuxdo:             linuxdo,
		authService:         authService,
		jwtService:          jwtService,
		revocationService:   revocationService,
		frontendCallbackURL: frontendCallbackURL,
		cookieSecure:        cookieSecure,
		jwtExpire:           jwtExpire,
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
		h.redirectError(c, frontendCB, "missing_params", "登录参数不完整，请重试", nil)
		return
	}
	cookieState, err := c.Cookie(oauthStateCookie)
	if err != nil || strings.TrimSpace(cookieState) == "" || cookieState != state {
		h.redirectError(c, frontendCB, "invalid_state", "登录状态校验失败，请重新发起登录", nil)
		return
	}
	verifier, err := c.Cookie(oauthVerifierCookie)
	if err != nil || strings.TrimSpace(verifier) == "" {
		h.redirectError(c, frontendCB, "missing_verifier", "登录会话已过期，请重新发起登录", nil)
		return
	}
	redirectTo, _ := c.Cookie(oauthRedirectCookie)
	redirectTo = sanitizeRedirect(redirectTo)

	profile, err := h.linuxdo.Authenticate(c.Request.Context(), code, verifier)
	if err != nil {
		h.redirectError(c, frontendCB, "oauth_failed", "第三方登录失败，请重试", err)
		return
	}

	binding, err := h.authService.ResolveAndBindUser(c.Request.Context(), profile.Subject, profile.Username)
	if err != nil {
		if err == service.ErrSub2APIUserNotFound {
			h.redirectError(c, frontendCB, "user_not_found", "请先在 sub2api 完成首次登录/注册", nil)
			return
		}
		h.redirectError(c, frontendCB, "bind_failed", "账号绑定失败，请稍后重试", err)
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
		h.redirectError(c, frontendCB, "token_failed", "登录凭证签发失败", err)
		return
	}

	clearCookie(c, oauthStateCookie, h.cookieSecure)
	clearCookie(c, oauthVerifierCookie, h.cookieSecure)
	clearCookie(c, oauthRedirectCookie, h.cookieSecure)
	setCookie(c, accessTokenCookie, token, int(h.jwtExpire.Seconds()), h.cookieSecure)

	fragment := url.Values{}
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
		"is_admin":        h.authService.IsAdminSubject(claims.LinuxDOSubject),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if claims, ok := middleware.GetClaims(c); ok && h.revocationService != nil && claims.ExpiresAt != nil {
		h.revocationService.Revoke(claims.ID, claims.ExpiresAt.Time)
	}
	clearCookie(c, accessTokenCookie, h.cookieSecure)
	Success(c, gin.H{"ok": true})
}

func (h *AuthHandler) redirectError(c *gin.Context, callback, code, msg string, detail error) {
	if detail != nil {
		traceID, _ := util.RandomToken(6)
		log.Printf("oauth callback failed: trace_id=%s code=%s err=%v", traceID, code, detail)
	}
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
	if !isSafeRedirectPath(path) {
		return "/"
	}
	return path
}

func isSafeRedirectPath(path string) bool {
	if path == "" || !strings.HasPrefix(path, "/") {
		return false
	}
	if strings.HasPrefix(path, "//") || strings.Contains(path, "://") || strings.ContainsAny(path, "\n\r") {
		return false
	}
	switch path {
	case "/", "/admin":
		return true
	default:
		return false
	}
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
