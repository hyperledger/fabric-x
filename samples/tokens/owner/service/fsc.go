/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/node"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/pagination"
	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	tok "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var (
	ErrCounterpartyAccountNotFound = errors.New("counterparty account not found")
	ErrInsufficientFunds           = errors.New("insufficient funds")
	ErrConnectionError             = errors.New("could not connect to counterparty")
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
)

func (f FabricSmartClient) Balances(ctx context.Context, wallet string) ([]Amount, error) {
	mgmt, err := token.GetManagementService(f.node)
	if err != nil {
		panic("configuration error, check the logs")
	}

	wal := mgmt.WalletManager().OwnerWallet(ctx, wallet)
	if wal == nil {
		return nil, ErrWalletNotFound
	}
	tokens, err := wal.ListUnspentTokens(token.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBalance, err)
	}

	amMap := make(map[string]uint64)
	for _, t := range tokens.Tokens {
		val, err := strconv.ParseUint(t.Quantity, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrBalance, err)
		}
		amMap[string(t.Type)] += val
	}

	am := make([]Amount, 0, len(amMap))
	for code, val := range amMap {
		am = append(am, Amount{Code: code, Value: val})
	}
	return am, nil
}

func (f FabricSmartClient) Balance(ctx context.Context, wallet, code string) (Amount, error) {
	mgmt, err := token.GetManagementService(f.node)
	if err != nil {
		panic("configuration error, check the logs")
	}
	wal := mgmt.WalletManager().OwnerWallet(ctx, wallet)
	if wal == nil {
		return Amount{}, ErrWalletNotFound
	}
	val, err := wal.Balance(ctx, token.WithContext(ctx), token.WithType(tok.Type(code)))
	if err != nil {
		return Amount{}, fmt.Errorf("%w: %w", ErrBalance, err)
	}
	return Amount{Code: code, Value: val}, nil
}

// Transfer transfers an amount of a certain token. It connects to the other node, prepares the transaction,
// gets it approved by the auditor and sends it to the blockchain for endorsement and commit.
func (f FabricSmartClient) Transfer(ctx context.Context, tokenType string, quantity uint64, sender string, recipient string, recipientNode string, message string) (txID string, err error) {
	logger.Infof("going to issue %d %s to [%s] on [%s] with message [%s]", quantity, tokenType, recipient, recipientNode, message)
	mgr, err := viewregistry.GetManager(f.node)
	if err != nil {
		return "", err
	}
	res, err := mgr.InitiateView(&TransferView{
		TransferOptions: &TransferOptions{
			Wallet:        sender,
			TokenType:     tokenType,
			Quantity:      quantity,
			Recipient:     recipient,
			RecipientNode: recipientNode,
			Message:       message,
		},
	}, ctx)
	if err != nil {
		logger.Errorf("error transferring: %s", err.Error())
		return "", err
	}
	txID, ok := res.(string)
	if !ok {
		return "", errors.New("cannot parse issue response")
	}
	logger.Infof("transferred %d %s from [%s] to [%s] on [%s] with message [%s]. ID: [%s]", quantity, tokenType, sender, recipient, recipientNode, message, txID)
	return txID, nil
}

func (f FabricSmartClient) Redeem(ctx context.Context, tokenType string, quantity uint64, sender string, message string) (txID string, err error) {
	return "", fmt.Errorf("not implemented: %s", "/owner/redeem") // TODO: Implement
}

// GetTransactions returns the full transaction history for an owner.
func (f FabricSmartClient) GetTransactions(ctx context.Context, wallet string) ([]ttx.TransactionRecord, error) {
	logger.Debugf("getting history for %s", wallet)
	txs := []ttx.TransactionRecord{}
	params := ttxdb.QueryTransactionsParams{
		SenderWallet:    wallet,
		RecipientWallet: wallet,
		Statuses:        []driver.TxStatus{driver.Confirmed, driver.Pending},
		ExcludeToSelf:   true,
	}
	mgmt, err := token.GetManagementService(f.node)
	if err != nil {
		panic("configuration error, check the logs")
	}
	owner := ttx.NewOwner(f.node, mgmt)
	if owner == nil {
		panic("configuration error, check the logs")
	}

	it, err := owner.Transactions(ctx, params, pagination.None())
	if err != nil || it == nil {
		return txs, errors.Wrap(err, "failed querying transactions from db")
	}

	defer it.Items.Close()
	for tx, err := it.Items.Next(); tx != nil || err != nil; tx, err = it.Items.Next() {
		if err != nil {
			return txs, err
		}
		txs = append(txs, *tx)
	}
	return txs, nil
}

type TransferOptions struct {
	// Wallet is the identifier of the wallet that owns the tokens to transfer
	Wallet string
	// TokenType of tokens to transfer
	TokenType string
	// Quantity to transfer
	Quantity uint64
	// RecipientNode is the identity of the recipient's FSC node
	RecipientNode string
	// Recipient is the identity of the recipient's wallet
	Recipient string
	// Message is an optional user message sent with the transaction.
	// It's stored in the ApplicationMetadata and is sent in the transient field.
	Message string
}

type TransferView struct {
	*TransferOptions
}

func (v *TransferView) Call(vctx view.Context) (interface{}, error) {
	// The sender will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(vctx, v.Wallet)
	if senderWallet == nil {
		return "", errors.Errorf("sender wallet [%s] not found", v.Wallet)
	}

	var recipient view.Identity
	var err error

	// Internal transaction: identity must exist in a wallet
	if len(v.RecipientNode) == 0 {
		logger.Infof("getting local identity for %s", v.Recipient)
		w := ttx.GetWallet(vctx, v.Recipient)
		if w == nil {
			return nil, ErrCounterpartyAccountNotFound
		}

		recipient, err = w.GetRecipientIdentity(vctx.Context())
		if err != nil {
			logger.Errorf("failed getting %s identity from own node: %s", v.Recipient, err.Error())
			return nil, ErrCounterpartyAccountNotFound
		}
	} else {
		recipient, err = getRemoteIdentity(vctx, v.Recipient, v.RecipientNode)
		if err != nil {
			return nil, err
		}
	}

	// Create the envelope for the transaction
	tx, err := ttx.NewTransaction(vctx, nil)
	if err != nil {
		return tx, errors.Wrap(err, "failed to create transaction")
	}
	if v.Message != "" {
		// You can set any metadata you want. It is shared with the recipient but not verified or committed to the ledger.
		tx.SetApplicationMetadata("message", []byte(v.Message))
	}

	// The sender adds a new transfer operation to the transaction.
	err = tx.Transfer(
		senderWallet,
		tok.Type(v.TokenType),
		[]uint64{v.Quantity},
		[]view.Identity{recipient},
		// token.WithPublicTransferMetadata("pub."+tx.ID(), []byte("public data")),
	)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient funds") {
			return "", ErrInsufficientFunds
		}
		return "", errors.Wrap(err, "failed preparing transfer")
	}

	// The sender is ready to collect all the required signatures.
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

func getRemoteIdentity(vctx view.Context, recipientID, recipientNode string) (view.Identity, error) {
	var recipient view.Identity
	var err error
	ctx := vctx.Context()

	// Bind identity names to tell FSC to contact the right node
	logger.Infof("binding [%s] to node [%s]", recipientID, recipientNode)
	if err = endpoint.GetService(vctx).Bind(ctx, view.Identity(recipientNode), view.Identity(recipientID)); err != nil {
		return nil, err
	}

	// Request recipient identity from other node
	logger.Infof("requesting [%s] identity from [%s]", recipientID, recipientNode)
	recipient, err = ttx.RequestRecipientIdentity(vctx, view.Identity(recipientID))
	if err != nil {
		if strings.Contains(err.Error(), "] not found") {
			return recipient, ErrCounterpartyAccountNotFound
		}
		if strings.Contains(err.Error(), "failed to dial") {
			return recipient, ErrConnectionError
		}
		return recipient, fmt.Errorf("failed getting recipient identity from %s: %w", recipientNode, err)
	}
	if recipient.IsNone() {
		return recipient, ErrCounterpartyAccountNotFound
	}

	return recipient, nil
}
