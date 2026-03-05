package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type LinuxDoService struct {
	clientID      string
	clientSecret  string
	authorizeURL  string
	tokenURL      string
	userinfoURL   string
	scopes        string
	redirectURL   string
	userIDField   string
	userNameField string
	httpClient    *http.Client
}

type LinuxDoProfile struct {
	Subject  string
	Username string
}

func NewLinuxDoService(
	clientID, clientSecret, authorizeURL, tokenURL, userinfoURL, scopes, redirectURL, userIDField, userNameField string,
	httpClient *http.Client,
) *LinuxDoService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &LinuxDoService{
		clientID:      clientID,
		clientSecret:  clientSecret,
		authorizeURL:  authorizeURL,
		tokenURL:      tokenURL,
		userinfoURL:   userinfoURL,
		scopes:        scopes,
		redirectURL:   redirectURL,
		userIDField:   userIDField,
		userNameField: userNameField,
		httpClient:    httpClient,
	}
}

func (s *LinuxDoService) BuildAuthorizeURL(state, codeChallenge string) (string, error) {
	u, err := url.Parse(s.authorizeURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", s.clientID)
	q.Set("redirect_uri", s.redirectURL)
	q.Set("scope", s.scopes)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *LinuxDoService) Authenticate(ctx context.Context, code, codeVerifier string) (*LinuxDoProfile, error) {
	accessToken, tokenType, err := s.exchangeCode(ctx, code, codeVerifier)
	if err != nil {
		return nil, err
	}
	payload, err := s.fetchUserInfo(ctx, accessToken, tokenType)
	if err != nil {
		return nil, err
	}
	subject := getStringFromPayload(payload, s.userIDField)
	username := getStringFromPayload(payload, s.userNameField)
	if subject == "" {
		return nil, errors.New("linuxdo userinfo missing subject")
	}
	if username == "" {
		username = subject
	}
	return &LinuxDoProfile{Subject: subject, Username: username}, nil
}

func (s *LinuxDoService) exchangeCode(ctx context.Context, code, codeVerifier string) (string, string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", s.redirectURL)
	form.Set("client_id", s.clientID)
	form.Set("client_secret", s.clientSecret)
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("linuxdo token exchange status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return "", "", errors.New("linuxdo token response missing access_token")
	}
	if strings.TrimSpace(parsed.TokenType) == "" {
		parsed.TokenType = "Bearer"
	}
	return parsed.AccessToken, parsed.TokenType, nil
}

func (s *LinuxDoService) fetchUserInfo(ctx context.Context, accessToken, tokenType string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.userinfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", tokenType+" "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("linuxdo userinfo status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func getStringFromPayload(payload map[string]interface{}, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	parts := strings.Split(path, ".")
	var current interface{} = payload
	for _, part := range parts {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = obj[part]
		if !ok {
			return ""
		}
	}
	switch v := current.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
