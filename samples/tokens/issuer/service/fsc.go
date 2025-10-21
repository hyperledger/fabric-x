/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"context"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/node"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = logging.MustGetLogger() // TODO

type FabricSmartClient struct {
	node *node.Node
}

func NewFSC(node *node.Node) *FabricSmartClient {
	return &FabricSmartClient{node: node}
}

// Amount The amount to issue, transfer or redeem.
type Amount struct {
	// Code the code of the token
	Code string

	// Value value in base units (usually cents)
	Value uint64
}

var (
	ErrWalletNotFound = errors.New("wallet not found")
	ErrBalance        = errors.New("error getting balance")
	ErrTechnicalError = errors.New("server error")
)

// Issue issues an amount of tokens to a wallet. It connects to the other node, prepares the transaction,
// gets it approved by the auditor and sends it to the blockchain for endorsement and commit.
func (f FabricSmartClient) Issue(ctx context.Context, tokenType string, quantity uint64, recipient string, recipientNode string, message string) (string, error) {
	logger.Infof("going to issue %d %s to [%s] on [%s] with message [%s]", quantity, tokenType, recipient, recipientNode, message)
	mgr, err := viewregistry.GetManager(f.node)
	if err != nil {
		return "", err
	}
	res, err := mgr.InitiateView(&IssueCashView{
		IssueCash: &IssueCash{
			TokenType:     tokenType,
			Quantity:      quantity,
			Recipient:     recipient,
			RecipientNode: recipientNode,
			Message:       message,
		},
	}, ctx)
	if err != nil {
		logger.Errorf("error issuing: %s", err.Error())
		return "", err
	}
	txID, ok := res.(string)
	if !ok {
		return "", errors.New("cannot parse issue response")
	}
	logger.Infof("issued %d %s to [%s] on [%s] with message [%s]. ID: [%s]", quantity, tokenType, recipient, recipientNode, message, txID)
	return txID, nil
}

// VIEW

// IssueCash contains the input information to issue a token
type IssueCash struct {
	// TokenType is the type of token to issue
	TokenType string
	// Quantity represent the number of units of a certain token type stored in the token
	Quantity uint64
	// Recipient is an identifier of the recipient identity
	Recipient string
	// RecipientNode is the identifier of the node of the recipient
	RecipientNode string
	// Message is the message that will be visible to the recipient and the auditor
	Message string
	// Auditor is the optional auditor to sign the transaction
	Auditor string
}

type IssueCashView struct {
	*IssueCash
}

func (v *IssueCashView) Call(vctx view.Context) (interface{}, error) {
	ctx := vctx.Context()
	wallet := ttx.MyIssuerWallet(vctx)
	if wallet == nil {
		return "", fmt.Errorf("issuer wallet not found")
	}

	rec := view.Identity(v.Recipient)
	eps := endpoint.GetService(vctx)
	err := eps.Bind(ctx, view.Identity(v.RecipientNode), rec)
	if err != nil {
		return "", fmt.Errorf("error binding %s to %s", v.Recipient, v.RecipientNode)
	}

	// As a first step operation, the issuer contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	recipient, err := ttx.RequestRecipientIdentity(vctx, rec)
	if err != nil {
		return "", fmt.Errorf("failed getting recipient identity from %s: %w", v.RecipientNode, err)
	}

	tx, err := ttx.NewTransaction(
		vctx,
		nil, // default signer
		ttx.WithAuditor(view.Identity(v.Auditor)),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed creating transaction")
	}

	// You can set any metadata you want. It is shared with the recipient and
	// auditor but not committed to the ledger. We used 'message' here to let
	// the user share messages that will be shown in the transaction history.
	if v.Message != "" {
		tx.SetApplicationMetadata("message", []byte(v.Message))
	}

	// The issuer adds a new issue operation to the transaction to issue
	// the amount to the recipient id recieved from the owner's node.
	if err = tx.Issue(
		wallet,
		recipient,
		token.Type(v.TokenType),
		v.Quantity,
	); err != nil {
		return "", errors.Wrap(err, "failed adding new issued token")
	}

	// The issuer is ready to collect all the required signatures.
	// This includes the auditor (if provided) and the endorsers.
	logger.Infof("collecting signatures and submitting transaction to chaincode: [%s]", tx.ID())
	_, err = vctx.RunView(ttx.NewCollectEndorsementsView(tx))
	if err != nil {
		return "", errors.Wrap(err, "failed to sign transaction")
	}

	// The issuer sends the transaction for ordering.
	logger.Infof("submitting fabric transaction to orderer for final settlemement: [%s]", tx.ID())
	_, err = vctx.RunView(ttx.NewOrderingAndFinalityView(tx))
	if err != nil {
		return nil, errors.Wrap(err, "failed asking ordering")
	}

	return tx.ID(), nil
}
