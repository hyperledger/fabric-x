/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"fmt"
)

// ListNamespaces queries the committer service for installed namespaces.
// It connects to the query service and retrieves all namespace policies.
func (d *AdminApp) ListNamespaces(ctx context.Context) ([]NamespaceQueryResult, error) {
	// get query service instance
	qc, err := d.QueryProvider.Get()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = qc.Close()
	}()

	res, err := qc.GetNamespacePolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot query existing namespaces: %w", err)
	}

	results := make([]NamespaceQueryResult, len(res.GetPolicies()))
	for i, p := range res.GetPolicies() {
		results[i] = NamespaceQueryResult{
			NsID:    p.GetNamespace(),
			Version: int(p.GetVersion()), //nolint:gosec
			Policy:  p.GetPolicy(),
		}
	}

	return results, nil
}

// NamespaceQueryResult represents a namespace retrieved from the query service.
type NamespaceQueryResult struct {
	NsID    string `json:"name" yaml:"name"`
	Version int    `json:"version" yaml:"version"`
	Policy  []byte `json:"policy" yaml:"policy"`
}
