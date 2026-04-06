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

// BindAddress returns the host used by HTTP services.
// By default we keep 0.0.0.0 for docker/native compatibility.
func BindAddress() string {
	if bindAddr := strings.TrimSpace(os.Getenv("BIND_ADDR")); bindAddr != "" {
		return bindAddr
	}
	return "0.0.0.0"
}

func allowedOriginsFromEnv() map[string]struct{} {
	allowedOrigins := map[string]struct{}{}

	// Safe defaults that still allow Swagger UI at localhost:8080 out of the box.
	for _, origin := range []string{"http://localhost:8080", "http://127.0.0.1:8080"} {
		allowedOrigins[origin] = struct{}{}
	}

	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		return allowedOrigins
	}

	custom := map[string]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(item)
		if origin == "" {
			continue
		}
		custom[origin] = struct{}{}
	}

	if len(custom) == 0 {
		return allowedOrigins
	}

	return custom
}

func allowAuthHeader() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ALLOW_AUTH_HEADER")))
	return v == "1" || v == "true" || v == "yes"
}

// WithCORS adds configurable CORS headers.
func WithCORS(next http.Handler) http.Handler {
	allowedOrigins := allowedOriginsFromEnv()
	allowAnyOrigin := false
	if _, ok := allowedOrigins["*"]; ok {
		allowAnyOrigin = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			isAllowed := allowAnyOrigin
			if !isAllowed {
				_, isAllowed = allowedOrigins[origin]
			}

			if !isAllowed {
				if r.Method == http.MethodOptions {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}

			if allowAnyOrigin {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			if allowAuthHeader() {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
