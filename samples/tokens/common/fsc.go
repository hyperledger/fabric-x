/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"net"
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

// BindAddress returns the API bind address. It defaults to localhost and can
// be overridden via BIND_ADDRESS (for example, 0.0.0.0 in containers).
func BindAddress(port string) string {
	bindHost := strings.TrimSpace(os.Getenv("BIND_ADDRESS"))
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}

	return net.JoinHostPort(bindHost, port)
}

// WithCORS enforces explicit CORS defaults.
//
// Behaviour:
//   - If ALLOWED_ORIGINS is not configured, cross-origin requests are rejected.
//   - If ALLOWED_ORIGINS is configured, only those origins are allowed.
//   - Authorization is only exposed when ALLOW_AUTH_HEADER=true.
func WithCORS(next http.Handler) http.Handler {
	allowedOrigins := parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))
	allowAuthHeader := strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_AUTH_HEADER")), "true")
	allowHeaders := "Content-Type"
	if allowAuthHeader {
		allowHeaders = "Content-Type, Authorization"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if !isOriginAllowed(origin, allowedOrigins) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseAllowedOrigins(origins string) map[string]struct{} {
	allowed := map[string]struct{}{}
	for _, item := range strings.Split(origins, ",") {
		origin := strings.TrimSpace(item)
		if origin == "" {
			continue
		}
		allowed[origin] = struct{}{}
	}

	return allowed
}

func isOriginAllowed(origin string, allowedOrigins map[string]struct{}) bool {
	if len(allowedOrigins) == 0 {
		return false
	}
	_, ok := allowedOrigins[origin]
	return ok
}
