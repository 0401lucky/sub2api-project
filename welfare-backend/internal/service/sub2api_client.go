package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Sub2APIClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type Sub2APIUser struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Status   string `json:"status"`
}

type sub2apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type sub2apiListData[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type sub2apiBalanceHistoryItem struct {
	Notes string `json:"notes"`
}

func NewSub2APIClient(baseURL, apiKey string, timeout time.Duration) *Sub2APIClient {
	return &Sub2APIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *Sub2APIClient) FindUserBySyntheticEmail(ctx context.Context, email string) (*Sub2APIUser, error) {
	targetEmail := strings.TrimSpace(email)
	const pageSize = 100
	for page := 1; page <= 50; page++ {
		q := url.Values{}
		q.Set("search", targetEmail)
		q.Set("page", strconv.Itoa(page))
		q.Set("page_size", strconv.Itoa(pageSize))
		endpoint := c.baseURL + "/api/v1/admin/users?" + q.Encode()

		var env sub2apiEnvelope
		if err := c.do(ctx, http.MethodGet, endpoint, nil, &env); err != nil {
			return nil, err
		}
		if env.Code != 0 {
			return nil, fmt.Errorf("sub2api error: %s", env.Message)
		}
		var data sub2apiListData[Sub2APIUser]
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return nil, fmt.Errorf("parse sub2api users: %w", err)
		}
		for i := range data.Items {
			if strings.EqualFold(strings.TrimSpace(data.Items[i].Email), targetEmail) {
				return &data.Items[i], nil
			}
		}
		if len(data.Items) < pageSize {
			break
		}
	}
	return nil, nil
}

func (c *Sub2APIClient) AddBalance(ctx context.Context, userID int64, amount float64, notes string) error {
	payload := map[string]interface{}{
		"balance":   amount,
		"operation": "add",
		"notes":     notes,
	}
	endpoint := c.baseURL + "/api/v1/admin/users/" + strconv.FormatInt(userID, 10) + "/balance"
	var env sub2apiEnvelope
	if err := c.do(ctx, http.MethodPost, endpoint, payload, &env); err != nil {
		return err
	}
	if env.Code != 0 {
		return fmt.Errorf("sub2api add balance failed: %s", env.Message)
	}
	return nil
}

func (c *Sub2APIClient) HasBalanceRecordByNoteToken(ctx context.Context, userID int64, noteToken string) (bool, error) {
	token := strings.TrimSpace(noteToken)
	const pageSize = 100
	for page := 1; page <= 50; page++ {
		q := url.Values{}
		q.Set("type", "admin_balance")
		q.Set("page", strconv.Itoa(page))
		q.Set("page_size", strconv.Itoa(pageSize))
		endpoint := c.baseURL + "/api/v1/admin/users/" + strconv.FormatInt(userID, 10) + "/balance-history?" + q.Encode()

		var env sub2apiEnvelope
		if err := c.do(ctx, http.MethodGet, endpoint, nil, &env); err != nil {
			return false, err
		}
		if env.Code != 0 {
			return false, fmt.Errorf("sub2api list balance history failed: %s", env.Message)
		}
		var data sub2apiListData[sub2apiBalanceHistoryItem]
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return false, fmt.Errorf("parse sub2api balance history: %w", err)
		}
		for _, item := range data.Items {
			if strings.Contains(item.Notes, token) {
				return true, nil
			}
		}
		if len(data.Items) < pageSize {
			break
		}
	}
	return false, nil
}

func (c *Sub2APIClient) do(ctx context.Context, method, endpoint string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sub2api http status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawResp)))
	}
	if err := json.Unmarshal(rawResp, out); err != nil {
		return fmt.Errorf("parse sub2api response: %w", err)
	}
	return nil
}
