// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-committer/service/verifier/policy"
	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/audit"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// DeployNamespaceInput contains parameters for namespace deployment.
type DeployNamespaceInput struct {
	NsID    string       `json:"name" yaml:"name"`
	Version int          `json:"version" yaml:"version"`
	Policy  PolicyConfig `json:"policy" yaml:"policy"`

	Endorse bool
	Submit  bool
	Wait    bool
}

// Validate validates namespace configuration.
// Checks namespace ID, version, and policy.
func (c *DeployNamespaceInput) Validate(vctx validation.Context) error {
	auditLogger := audit.MustGetAuditLogger(nil)

	auditLogger.NamespaceDeployInputValidation(context.Background(), audit.NamespaceDeployInputValidationEvent{
		EventMeta:  audit.NewEventMeta(),
		NamespaceID: c.NsID,
		Version:    c.Version,
		PolicyType: string(c.Policy.Type),
		Endorse:    c.Endorse,
		Submit:     c.Submit,
		Wait:       c.Wait,
		Result:     "pending",
	})

	if err := policy.ValidateNamespaceID(c.NsID); err != nil {
		auditLogger.NamespaceDeployInputValidation(context.Background(), audit.NamespaceDeployInputValidationEvent{
			EventMeta:   audit.NewEventMeta(),
			NamespaceID: c.NsID,
			Version:    c.Version,
			PolicyType: string(c.Policy.Type),
			Result:     "failure",
			ErrorMsg:   fmt.Errorf("invalid namespaceID: %w", err).Error(),
		})
		return fmt.Errorf("invalid namespaceID: %w", err)
	}

	if err := transaction.ValidateVersion(c.Version); err != nil {
		auditLogger.NamespaceDeployInputValidation(context.Background(), audit.NamespaceDeployInputValidationEvent{
			EventMeta:   audit.NewEventMeta(),
			NamespaceID: c.NsID,
			Version:    c.Version,
			PolicyType: string(c.Policy.Type),
			Result:     "failure",
			ErrorMsg:   fmt.Errorf("invalid version: %w", err).Error(),
		})
		return fmt.Errorf("invalid version: %w", err)
	}

	if err := c.Policy.Validate(vctx); err != nil {
		auditLogger.NamespaceDeployInputValidation(context.Background(), audit.NamespaceDeployInputValidationEvent{
			EventMeta:   audit.NewEventMeta(),
			NamespaceID: c.NsID,
			Version:    c.Version,
			PolicyType: string(c.Policy.Type),
			Result:     "failure",
			ErrorMsg:   fmt.Errorf("invalid policy: %w", err).Error(),
		})
		return fmt.Errorf("invalid policy: %w", err)
	}

	auditLogger.NamespaceDeployInputValidation(context.Background(), audit.NamespaceDeployInputValidationEvent{
		EventMeta:  audit.NewEventMeta(),
		NamespaceID: c.NsID,
		Version:    c.Version,
		PolicyType: string(c.Policy.Type),
		Endorse:    c.Endorse,
		Submit:     c.Submit,
		Wait:       c.Wait,
		Result:     "success",
	})

	return nil
}

// DeployNamespaceOutput contains the generated transaction and ID.
type DeployNamespaceOutput struct {
	TxID string
	Tx   *applicationpb.Tx
}

// DeployNamespace creates a namespace transaction and submits it to the ordering service.
func (d *AdminApp) DeployNamespace(
	ctx context.Context,
	input *DeployNamespaceInput,
) (*DeployNamespaceOutput, TxStatus, error) {
	// input validation
	if err := input.Validate(d.Validators); err != nil {
		return nil, UnknownStatus, err
	}

	// create namespace tx
	out, err := d.CreateNamespace(ctx, input)
	if err != nil {
		return nil, UnknownStatus, err
	}

	if !input.Endorse {
		return out, UnknownStatus, nil
	}

	// Endorse transaction
	out.Tx, err = d.EndorseTransaction(ctx, out.TxID, out.Tx)
	if err != nil {
		return nil, UnknownStatus, err
	}

	// note that we enforce submit if wait is set
	if !input.Submit && !input.Wait {
		return out, UnknownStatus, nil
	}

	// submit transaction
	if input.Wait {
		status, err := d.SubmitTransactionWithWait(ctx, out.TxID, out.Tx)
		if err != nil {
			return nil, UnknownStatus, err
		}
		return nil, status, nil
	}
	if err := d.SubmitTransaction(ctx, out.TxID, out.Tx); err != nil {
		return nil, UnknownStatus, err
	}

	return nil, UnknownStatus, nil
}
