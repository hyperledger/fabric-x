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
// Returns transaction ID and unsigned transaction for later processing.
func (*AdminApp) CreateNamespace(_ context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, error) {
	nsPolicy, err := createPolicy(input.Policy)
	if err != nil {
		return nil, err
	}

	txID, err := transaction.GenerateTxID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate transaction ID: %w", err)
	}

	out := &DeployNamespaceOutput{
		TxID: txID,
		Tx:   transaction.CreateNamespacesTx(nsPolicy, input.NsID, input.Version),
	}

	return out, nil
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
