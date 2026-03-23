package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type okAuthStore struct{}

func (okAuthStore) Register(_, _ string) error {
	return nil
}

func (okAuthStore) Authenticate(_, _ string) error {
	return nil
}

func TestLoginHandlerSetsSessionCookie(t *testing.T) {
	handler := loginHandler(okAuthStore{})
	reqBody := []byte(`{"id":"admin","password":"12345"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}

	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == authSessionCookieName {
			found = true
			if c.Value != authSessionCookieValue {
				t.Fatalf("unexpected cookie value: %s", c.Value)
			}
			if !c.HttpOnly {
				t.Fatalf("expected httpOnly cookie")
			}
			if c.Path != "/" {
				t.Fatalf("unexpected cookie path: %s", c.Path)
			}
		}
	}
	if !found {
		t.Fatalf("expected auth session cookie")
	}

	var body loginResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RedirectTo != "/" {
		t.Fatalf("unexpected redirect: %s", body.RedirectTo)
	}
}

func TestRootHandlerRedirectsBySessionCookie(t *testing.T) {
	noAuthReq := httptest.NewRequest(http.MethodGet, "/", nil)
	noAuthRR := httptest.NewRecorder()
	rootHandler(noAuthRR, noAuthReq)
	if noAuthRR.Code != http.StatusFound {
		t.Fatalf("unexpected status without auth: %d", noAuthRR.Code)
	}
	if location := noAuthRR.Header().Get("Location"); location != "/login" {
		t.Fatalf("unexpected redirect without auth: %s", location)
	}

	withAuthReq := httptest.NewRequest(http.MethodGet, "/", nil)
	withAuthReq.AddCookie(&http.Cookie{
		Name:  authSessionCookieName,
		Value: authSessionCookieValue,
		Path:  "/",
	})
	withAuthRR := httptest.NewRecorder()
	rootHandler(withAuthRR, withAuthReq)
	if withAuthRR.Code != http.StatusFound {
		t.Fatalf("unexpected status with auth: %d", withAuthRR.Code)
	}
	if location := withAuthRR.Header().Get("Location"); location != "/trade" {
		t.Fatalf("unexpected redirect with auth: %s", location)
	}
}

func TestLoginPageHandlerRedirectsWhenAuthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{
		Name:  authSessionCookieName,
		Value: authSessionCookieValue,
		Path:  "/",
	})
	rr := httptest.NewRecorder()

	loginPageHandler(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/trade" {
		t.Fatalf("unexpected redirect: %s", location)
	}
}

func TestLogoutHandlerClearsSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rr := httptest.NewRecorder()

	logoutHandler(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/login" {
		t.Fatalf("unexpected redirect: %s", location)
	}

	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == authSessionCookieName {
			found = true
			if c.MaxAge >= 0 {
				t.Fatalf("expected cookie to be expired, maxAge=%d", c.MaxAge)
			}
		}
	}
	if !found {
		t.Fatalf("expected clear-session cookie")
	}
}
