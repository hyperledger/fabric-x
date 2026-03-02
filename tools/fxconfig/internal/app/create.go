/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package app provides functionality for namespace lifecycle operations.
// It handles creating, updating, and listing namespaces in Fabric-X.
package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

func (d *AdminApp) CreateNamespace(_ context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, error) {
	nsPolicy, err := createPolicy(input.Policy)
	if err != nil {
		return nil, err
	}

	out := &DeployNamespaceOutput{
		TxID: transaction.GenerateTxID(),
		Tx:   transaction.CreateNamespacesTx(nsPolicy, input.NsID, input.Version),
	}

	return out, nil
}

// createPolicy creates a namespace policy from configuration.
// Supports MSP-based and threshold ECDSA policies.
func createPolicy(cfg PolicyConfig) (*applicationpb.NamespacePolicy, error) {
	switch cfg.Type {
	case "msp":
		return transaction.CreateMspPolicy(cfg.MSP.Expression)

	case "threshold":
		return transaction.CreateThresholdPolicy(cfg.Threshold.VerificationKeyPath)

	default:
		return nil, fmt.Errorf("unknown policy type: %s", cfg.Type)
	}
}
