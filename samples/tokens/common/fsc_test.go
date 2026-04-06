/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBindAddress_DefaultAndOverride(t *testing.T) {
	t.Setenv("BIND_ADDR", "")
	require.Equal(t, "0.0.0.0", BindAddress())

	t.Setenv("BIND_ADDR", "127.0.0.1")
	require.Equal(t, "127.0.0.1", BindAddress())
}

func TestWithCORS_DefaultAllowsSwaggerLocalhost(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("ALLOW_AUTH_HEADER", "")

	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	require.Equal(t, http.StatusNoContent, rw.Code)
	require.Equal(t, "http://localhost:8080", rw.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "Content-Type", rw.Header().Get("Access-Control-Allow-Headers"))
}

func TestWithCORS_DefaultRejectsUnknownOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")

	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	require.Equal(t, http.StatusForbidden, rw.Code)
}

func TestWithCORS_CustomAllowedOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "http://example.com, http://localhost:8080")

	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	require.Equal(t, http.StatusNoContent, rw.Code)
	require.Equal(t, "http://example.com", rw.Header().Get("Access-Control-Allow-Origin"))
}

func TestWithCORS_AllowAnyOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "*")

	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	require.Equal(t, http.StatusNoContent, rw.Code)
	require.Equal(t, "*", rw.Header().Get("Access-Control-Allow-Origin"))
}

func TestWithCORS_AllowAuthorizationHeader(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:8080")
	t.Setenv("ALLOW_AUTH_HEADER", "true")

	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	require.Equal(t, http.StatusNoContent, rw.Code)
	require.Equal(t, "Content-Type, Authorization", rw.Header().Get("Access-Control-Allow-Headers"))
}
