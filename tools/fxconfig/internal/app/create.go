/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

const (
	mspPolicyType       = "msp"
	thresholdPolicyType = "threshold"
)

// CreateNamespace generates a namespace transaction without endorsement or submission.
// It queries the committer for the current channel config version, which the transaction must carry.
// Returns transaction ID and unsigned transaction for later processing.
func (d *AdminApp) CreateNamespace(ctx context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, error) {
	nsPolicy, err := createPolicy(input.Policy)
	if err != nil {
		return nil, err
	}

	configVersion, err := d.configVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get channel config version from query service: %w", err)
	}

	out := &DeployNamespaceOutput{
		TxID: transaction.GenerateTxID(),
		Tx:   transaction.CreateNamespacesTx(nsPolicy, input.NsID, input.Version, configVersion),
	}

	return out, nil
}

// configVersion fetches the latest committed channel config version from the query service.
func (d *AdminApp) configVersion(ctx context.Context) (uint64, error) {
	qc, err := d.QueryProvider.Get()
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = qc.Close()
	}()

	return qc.GetConfigVersion(ctx)
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
