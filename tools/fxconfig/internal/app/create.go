/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/audit"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

const (
	mspPolicyType       = "msp"
	thresholdPolicyType = "threshold"
)

// CreateNamespace generates a namespace transaction without endorsement or submission.
// Returns transaction ID and unsigned transaction for later processing.
func (*AdminApp) CreateNamespace(_ context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, error) {
	auditLogger := audit.MustGetAuditLogger(nil)

	auditLogger.NamespaceCreationStarted(context.Background(), audit.NamespaceCreationStartedEvent{
		EventMeta:   audit.NewEventMeta(),
		NamespaceID: input.NsID,
		Version:     input.Version,
	})

	nsPolicy, err := createPolicy(input.Policy)
	if err != nil {
		auditLogger.NamespaceCreated(context.Background(), audit.NamespaceCreatedEvent{
			EventMeta:   audit.NewEventMeta(),
			Namespace:   input.NsID,
			Version:     input.Version,
			Result:      "failure",
			ErrorMsg:    err.Error(),
		})
		return nil, err
	}

	out := &DeployNamespaceOutput{
		TxID: transaction.GenerateTxID(),
		Tx:   transaction.CreateNamespacesTx(nsPolicy, input.NsID, input.Version),
	}

	policyInfo := audit.PolicyInfo{Type: string(input.Policy.Type)}
	if input.Policy.Type == mspPolicyType {
		policyInfo.MSPExpression = input.Policy.MSP.Expression
	} else if input.Policy.Type == thresholdPolicyType {
		policyInfo.VerificationKeyPath = input.Policy.Threshold.VerificationKeyPath
	}

	auditLogger.NamespaceCreated(context.Background(), audit.NamespaceCreatedEvent{
		EventMeta:  audit.NewEventMeta(),
		TxID:       out.TxID,
		Namespace:  input.NsID,
		Version:    input.Version,
		Policy:     policyInfo,
		Result:     "success",
	})

	return out, nil
}

// createPolicy creates a namespace policy from configuration.
// Supports MSP-based and threshold ECDSA policies.
func createPolicy(cfg PolicyConfig) (*applicationpb.NamespacePolicy, error) {
	switch cfg.Type {
	case mspPolicyType:
		auditLogger := audit.MustGetAuditLogger(nil)
		auditLogger.PolicyValidationStarted(context.Background(), audit.PolicyValidationStartedEvent{
			EventMeta:  audit.NewEventMeta(),
			PolicyType: "msp",
			Expression: cfg.MSP.Expression,
		})
		policy, err := transaction.CreateMspPolicy(cfg.MSP.Expression)
		if err != nil {
			auditLogger.PolicyValidated(context.Background(), audit.PolicyValidatedEvent{
				EventMeta:  audit.NewEventMeta(),
				PolicyType: "msp",
				Expression: cfg.MSP.Expression,
				Result:     "failure",
				ErrorMsg:   err.Error(),
			})
			return nil, err
		}
		auditLogger.PolicyValidated(context.Background(), audit.PolicyValidatedEvent{
			EventMeta:  audit.NewEventMeta(),
			PolicyType: "msp",
			Expression: cfg.MSP.Expression,
			Result:     "success",
		})
		return policy, nil

	case thresholdPolicyType:
		auditLogger := audit.MustGetAuditLogger(nil)
		auditLogger.PolicyValidationStarted(context.Background(), audit.PolicyValidationStartedEvent{
			EventMeta:          audit.NewEventMeta(),
			PolicyType:        "threshold",
			Expression:         cfg.Threshold.VerificationKeyPath,
		})
		policy, err := transaction.CreateThresholdPolicy(cfg.Threshold.VerificationKeyPath)
		if err != nil {
			auditLogger.PolicyValidated(context.Background(), audit.PolicyValidatedEvent{
				EventMeta:   audit.NewEventMeta(),
				PolicyType:  "threshold",
				Expression:  cfg.Threshold.VerificationKeyPath,
				Result:      "failure",
				ErrorMsg:    err.Error(),
			})
			return nil, err
		}
		auditLogger.PolicyValidated(context.Background(), audit.PolicyValidatedEvent{
			EventMeta:   audit.NewEventMeta(),
			PolicyType:  "threshold",
			Expression:  cfg.Threshold.VerificationKeyPath,
			Result:      "success",
		})
		return policy, nil

	default:
		return nil, fmt.Errorf("unknown policy type: %s", cfg.Type)
	}
}
