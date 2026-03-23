package kiwoom

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/salt-ux/stock-bot/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestValidateCredentialsSuccess(t *testing.T) {
	client := NewClient(config.KiwoomConfig{
		BaseURL:   "https://api.example.com",
		TokenPath: "/oauth2/token",
		AppKey:    "abcdefgh",
		AppSecret: "abcdefgh",
	})
	client.http = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/oauth2/token" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"t"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	res := client.ValidateCredentials(context.Background())
	if !res.Valid {
		t.Fatalf("expected valid response, got: %+v", res)
	}
}

func TestValidateCredentialsMissingKey(t *testing.T) {
	client := NewClient(config.KiwoomConfig{})
	res := client.ValidateCredentials(context.Background())
	if res.Valid {
		t.Fatalf("expected invalid response, got: %+v", res)
	}
}
