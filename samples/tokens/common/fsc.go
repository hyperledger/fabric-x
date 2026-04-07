/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/node"
)

// Default allowed origins for CORS - supports Swagger UI on common localhost ports.
// Override with ALLOWED_ORIGINS environment variable (comma-separated list).
// Set ALLOWED_ORIGINS=* to allow all origins (not recommended for production).
var defaultAllowedOrigins = []string{
	"http://localhost:8080",
	"http://127.0.0.1:8080",
	"http://localhost:3000",
	"http://127.0.0.1:3000",
}

// StartFSC starts a new node.
func StartFSC(confPath, datadir string) (*node.Node, error) {
	if len(datadir) != 0 {
		if err := os.MkdirAll(datadir, 0755); err != nil {
			return nil, fmt.Errorf("error creating data directory %s: %w", datadir, err)
		}
	}

	fsc := node.NewWithConfPath(confPath)
	if err := fsc.InstallSDK(NewSDK(fsc)); err != nil {
		return nil, fmt.Errorf("error installing fsc: %w", err)
	}
	if err := fsc.Start(); err != nil {
		return nil, fmt.Errorf("error starting fsc: %w", err)
	}

	return fsc, nil
}

// GetBindAddress returns the address to bind to.
// Defaults to 0.0.0.0 for Docker compatibility, but can be overridden
// via BIND_ADDRESS environment variable (e.g., "127.0.0.1" for local-only).
func GetBindAddress() string {
	if addr := os.Getenv("BIND_ADDRESS"); addr != "" {
		return addr
	}
	return "0.0.0.0"
}

// getAllowedOrigins returns the list of allowed CORS origins.
// Uses ALLOWED_ORIGINS env var if set, otherwise returns defaults.
func getAllowedOrigins() ([]string, bool) {
	if env := os.Getenv("ALLOWED_ORIGINS"); env != "" {
		if env == "*" {
			return nil, true // allow all
		}
		origins := strings.Split(env, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		return origins, false
	}
	return defaultAllowedOrigins, false
}

// isOriginAllowed checks if the given origin is in the allowed list.
func isOriginAllowed(origin string, allowedOrigins []string, allowAll bool) bool {
	if allowAll {
		return true
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// WithCORS adds configurable CORS headers to responses.
// By default, allows requests from common localhost origins (for Swagger UI).
// Configure via environment variables:
//   - ALLOWED_ORIGINS: comma-separated list of allowed origins, or "*" for all
//   - ALLOW_AUTH_HEADER: set to "true" to include Authorization in allowed headers
func WithCORS(next http.Handler) http.Handler {
	allowedOrigins, allowAll := getAllowedOrigins()
	allowAuthHeader := os.Getenv("ALLOW_AUTH_HEADER") == "true"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// No Origin header means same-origin request, allow it
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !isOriginAllowed(origin, allowedOrigins, allowAll) {
			// Reject cross-origin request from unknown origin
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			// For non-preflight requests, proceed but don't set CORS headers
			// Browser will block the response
			next.ServeHTTP(w, r)
			return
		}

		// Set CORS headers for allowed origins
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		allowedHeaders := "Content-Type"
		if allowAuthHeader {
			allowedHeaders += ", Authorization"
		}
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
