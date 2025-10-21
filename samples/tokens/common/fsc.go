/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"net/http"
	"os"

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

// WithAnyCORS adds permissive CORS headers to all responses
func WithAnyCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
