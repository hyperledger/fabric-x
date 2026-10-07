/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

const (
	mspPolicyType       = "msp"
	thresholdPolicyType = "threshold"
)

// CreateNamespace generates a namespace transaction without endorsement or submission.
// Returns transaction ID and unsigned transaction for later processing.
func (d *AdminApp) CreateNamespace(ctx context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, error) {
	nsPolicy, err := createPolicy(input.Policy)
	if err != nil {
		return nil, err
	}

	metaNsVersion, err := d.metaNamespaceVersion(ctx)
	if err != nil {
		return nil, err
	}

	out := &DeployNamespaceOutput{
		TxID: transaction.GenerateTxID(),
		Tx:   transaction.CreateNamespacesTx(nsPolicy, input.NsID, input.Version, metaNsVersion),
	}

	return out, nil
}

// metaNamespaceVersion returns the current version of the meta namespace.
func (d *AdminApp) metaNamespaceVersion(ctx context.Context) (uint64, error) {
	if d.QueryProvider == nil {
		return 0, fmt.Errorf("query provider is required")
	}

	qc, err := d.QueryProvider.Get()
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = qc.Close()
	}()

	res, err := qc.GetNamespacePolicies(ctx)
	if err != nil {
		return 0, fmt.Errorf("cannot query existing namespaces: %w", err)
	}

	for _, p := range res.GetPolicies() {
		if p.GetNamespace() == committerpb.MetaNamespaceID {
			return p.GetVersion(), nil
		}
	}

	// On fresh networks the meta namespace policy may not be listed yet.
	// In that case, start with version 0 and let subsequent updates advance it.
	return 0, nil
}

// createPolicy creates a namespace policy from configuration.
// Supports MSP-based and threshold ECDSA policies.
func createPolicy(cfg PolicyConfig) (*applicationpb.NamespacePolicy, error) {
	switch cfg.Type {
	case mspPolicyType:
		return transaction.CreateMspPolicy(cfg.MSP.Expression)

	case thresholdPolicyType:
		return transaction.CreateThresholdPolicy(cfg.Threshold.VerificationKeyPath)

	default:
		return nil, fmt.Errorf("unknown policy type: %s", cfg.Type)
	}
}
