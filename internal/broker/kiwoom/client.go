package kiwoom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/salt-ux/stock-bot/internal/broker"
	"github.com/salt-ux/stock-bot/internal/config"
)

type Client struct {
	baseURL   string
	tokenPath string
	appKey    string
	appSecret string
	http      *http.Client
}

type tokenRequest struct {
	GrantType string `json:"grant_type"`
	AppKey    string `json:"appkey"`
	SecretKey string `json:"secretkey"`
}

func NewClient(cfg config.KiwoomConfig) *Client {
	return &Client{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		tokenPath: cfg.TokenPath,
		appKey:    cfg.AppKey,
		appSecret: cfg.AppSecret,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) ValidateCredentials(ctx context.Context) broker.ValidationResult {
	if c.appKey == "" || c.appSecret == "" {
		return broker.ValidationResult{Valid: false, Message: "키움 앱 키/시크릿이 설정되지 않았습니다"}
	}
	if c.baseURL == "" {
		return broker.ValidationResult{Valid: false, Message: "KIWOOM_BASE_URL이 설정되지 않았습니다"}
	}

	payload, err := json.Marshal(tokenRequest{
		GrantType: "client_credentials",
		AppKey:    c.appKey,
		SecretKey: c.appSecret,
	})
	if err != nil {
		return broker.ValidationResult{Valid: false, Message: fmt.Sprintf("요청 생성 실패: %v", err)}
	}

	endpoint := c.baseURL + c.tokenPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return broker.ValidationResult{Valid: false, Message: fmt.Sprintf("요청 생성 실패: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return broker.ValidationResult{Valid: false, Message: fmt.Sprintf("키움 인증 API 호출 실패: %v", err)}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return broker.ValidationResult{Valid: true, StatusCode: resp.StatusCode, Message: "키움 API 키가 유효합니다"}
	}

	return broker.ValidationResult{
		Valid:      false,
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("키움 인증 실패: %s", strings.TrimSpace(string(body))),
	}
}
