/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestBindAddress_DefaultLocalhost(t *testing.T) {
	t.Setenv("BIND_ADDRESS", "")
	if got, want := BindAddress("9000"), "127.0.0.1:9000"; got != want {
		t.Fatalf("BindAddress() = %q, want %q", got, want)
	}
}

func TestBindAddress_FromEnv(t *testing.T) {
	t.Setenv("BIND_ADDRESS", "0.0.0.0")
	if got, want := BindAddress("9000"), "0.0.0.0:9000"; got != want {
		t.Fatalf("BindAddress() = %q, want %q", got, want)
	}
}

func TestWithCORS_RejectsUnknownOriginByDefault(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("ALLOW_AUTH_HEADER", "")

	h := WithCORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called for forbidden origin")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/owner/accounts/alice", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestWithCORS_AllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:8080")
	t.Setenv("ALLOW_AUTH_HEADER", "")

	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/owner/accounts/alice", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8080" {
		t.Fatalf("allow-origin = %q, want %q", got, "http://localhost:8080")
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Fatalf("allow-headers = %q, want %q", got, "Content-Type")
	}
}

func TestWithCORS_AllowsAuthorizationHeaderWhenEnabled(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:8080")
	t.Setenv("ALLOW_AUTH_HEADER", "true")

	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/owner/accounts/alice", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Fatalf("allow-headers = %q, want %q", got, "Content-Type, Authorization")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	_ = os.Unsetenv("BIND_ADDRESS")
	_ = os.Unsetenv("ALLOWED_ORIGINS")
	_ = os.Unsetenv("ALLOW_AUTH_HEADER")
	os.Exit(code)
}
