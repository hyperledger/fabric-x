package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-committer/service/verifier/policy"
	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

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
	if err := policy.ValidateNamespaceID(c.NsID); err != nil {
		return fmt.Errorf("invalid namespaceID: %w", err)
	}

	if err := transaction.ValidateVersion(c.Version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	if err := c.Policy.Validate(vctx); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	return nil
}

type DeployNamespaceOutput struct {
	TxID string
	Tx   *applicationpb.Tx
}

// DeployNamespace creates a namespace transaction and submits it to the ordering service.
func (d *AdminApp) DeployNamespace(ctx context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, TxStatus, error) {
	// input validation
	err := d.validate(input)
	if err != nil {
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

	if !input.Submit {
		return out, UnknownStatus, nil
	}

	// submit transaction
	if input.Wait {
		if status, err := d.SubmitTransactionWithWait(ctx, out.TxID, out.Tx); err != nil {
			return nil, status, err
		}
	}
	if err := d.SubmitTransaction(ctx, out.TxID, out.Tx); err != nil {
		return nil, UnknownStatus, err
	}

	return nil, UnknownStatus, nil
}

func (d *AdminApp) validate(input *DeployNamespaceInput) error {
	// input validation
	if err := input.Validate(d.Validators); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	// msp validation
	if input.Endorse {
		if err := d.MspProvider.Validate(); err != nil {
			return fmt.Errorf("invalid msp configuration: %w", err)
		}
	}

	// orderer validation
	if input.Submit {
		if err := d.OrdererProvider.Validate(); err != nil {
			return fmt.Errorf("invalid ordering service configuration: %w", err)
		}
	}

	return nil
}
