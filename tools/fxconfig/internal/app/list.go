/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	msppb "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"google.golang.org/protobuf/proto"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// ListNamespaces queries the committer service for installed namespaces.
// It connects to the query service, retrieves all namespace policies, and returns
// results with human-readable policy strings alongside the raw policy bytes.
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
		policyStr, err := parsePolicy(p.GetPolicy())
		if err != nil {
			policyStr = fmt.Sprintf("%x", p.GetPolicy())
		}
		results[i] = NamespaceQueryResult{
			NsID:      p.GetNamespace(),
			Version:   int(p.GetVersion()), //nolint:gosec
			Policy:    p.GetPolicy(),
			PolicyStr: policyStr,
		}
	}

	return results, nil
}

// NamespaceQueryResult represents a namespace retrieved from the query service.
type NamespaceQueryResult struct {
	NsID      string `json:"name" yaml:"name"`
	Version   int    `json:"version" yaml:"version"`
	Policy    []byte `json:"policy" yaml:"policy"`
	PolicyStr string `json:"policyStr" yaml:"policyStr"`
}

// parsePolicy decodes raw NamespacePolicy bytes into a human-readable string.
// For MSP policies it returns the DSL expression e.g. OR('Org1MSP.member', 'Org2MSP.member').
// For threshold policies it returns the base64-encoded public key.
// Falls back to hex on unmarshal errors.
func parsePolicy(b []byte) (string, error) {
	var nsPolicy applicationpb.NamespacePolicy
	if err := proto.Unmarshal(b, &nsPolicy); err != nil {
		return "", fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	switch r := nsPolicy.Rule.(type) {
	case *applicationpb.NamespacePolicy_ThresholdRule:
		pubKey := base64.StdEncoding.EncodeToString(r.ThresholdRule.GetPublicKey())
		return fmt.Sprintf("Threshold(ECDSA, %s)", pubKey), nil
	case *applicationpb.NamespacePolicy_MspRule:
		var env cb.SignaturePolicyEnvelope
		if err := proto.Unmarshal(r.MspRule, &env); err != nil {
			return "", fmt.Errorf("failed to unmarshal MSP rule: %w", err)
		}
		return signaturePolicyToString(env.GetRule(), env.GetIdentities())
	default:
		return fmt.Sprintf("%x", b), nil
	}
}

// signaturePolicyToString recursively converts a SignaturePolicy tree into a DSL string.
// AND/OR are derived from NOutOf: n==len(rules) → AND, n==1 → OR, otherwise OutOf(n,...).
func signaturePolicyToString(rule *cb.SignaturePolicy, identities []*msppb.MSPPrincipal) (string, error) {
	if rule == nil {
		return "", nil
	}

	switch t := rule.Type.(type) {
	case *cb.SignaturePolicy_SignedBy:
		idx := t.SignedBy
		if int(idx) >= len(identities) {
			return "", fmt.Errorf("identity index %d out of range (have %d)", idx, len(identities))
		}
		s, err := principalToString(identities[idx])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("'%s'", s), nil
	case *cb.SignaturePolicy_NOutOf_:
		n := t.NOutOf.GetN()
		rules := t.NOutOf.GetRules()
		parts := make([]string, 0, len(rules))
		for _, r := range rules {
			s, err := signaturePolicyToString(r, identities)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		switch {
		case n == 1:
			return fmt.Sprintf("OR(%s)", strings.Join(parts, ", ")), nil
		case int(n) == len(rules):
			return fmt.Sprintf("AND(%s)", strings.Join(parts, ", ")), nil
		default:
			return fmt.Sprintf("OutOf(%d, %s)", n, strings.Join(parts, ", ")), nil
		}
	default:
		return fmt.Sprintf("%v", rule), nil
	}
}

// principalToString converts an MSPPrincipal to a human-readable string like "Org1MSP.member".
func principalToString(p *msppb.MSPPrincipal) (string, error) {
	if p.GetPrincipalClassification() == msppb.MSPPrincipal_ROLE {
		var role msppb.MSPRole
		if err := proto.Unmarshal(p.GetPrincipal(), &role); err != nil {
			return "", fmt.Errorf("failed to unmarshal MSP role: %w", err)
		}
		return fmt.Sprintf("%s.%s", role.GetMspIdentifier(), strings.ToLower(role.GetRole().String())), nil
	}
	return fmt.Sprintf("%x", p.GetPrincipal()), nil
}
