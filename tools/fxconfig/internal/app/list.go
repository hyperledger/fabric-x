/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"google.golang.org/protobuf/proto"
)

// ListNamespaces queries the committer service for installed namespaces.
// It connects to the query service, retrieves all namespace policies, and formats
// the Output showing namespace names, versions, and policy data in hexadecimal.
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
			NsID:      p.GetNamespace(),
			Version:   int(p.GetVersion()), //nolint:gosec
			Policy:    p.GetPolicy(),
			PolicyStr: parsePolicy(p.GetPolicy()),
		}
	}

	return results, nil
}

// NamespaceQueryResult represents a namespace retrieved from the query service.
type NamespaceQueryResult struct {
	NsID      string `json:"name" yaml:"name"`
	Version   int    `json:"version" yaml:"version"`
	Policy    []byte `json:"-" yaml:"-"`
	PolicyStr string `json:"policy" yaml:"policy"`
}

// parsePolicy extracts and formats policy information from serialized bytes.
// Returns base64-encoded public key for threshold policies or string representation for MSP policies.
func parsePolicy(b []byte) string {
	var p applicationpb.NamespacePolicy
	if err := proto.Unmarshal(b, &p); err != nil {
		return "invalid policy"
	}

	switch r := p.Rule.(type) {
	case *applicationpb.NamespacePolicy_ThresholdRule:
		return base64.StdEncoding.EncodeToString(r.ThresholdRule.GetPublicKey())

	case *applicationpb.NamespacePolicy_MspRule:
		var en common.SignaturePolicyEnvelope
		if err := proto.Unmarshal(r.MspRule, &en); err != nil {
			return "invalid MSP policy"
		}

		return en.String()

	default:
		return "error parsing policy"
	}
}
