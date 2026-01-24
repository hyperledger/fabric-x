/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package routes

import (
	"context"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger/fabric-samples/token-sdk/owner/service"
)

//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=./oapi-server.yaml ../../swagger.yaml

type Server struct {
	fsc *service.FabricSmartClient
}

func NewServer(fsc *service.FabricSmartClient) Server {
	return Server{fsc: fsc}
}

// Get all accounts on this node and their balances of each type
// (GET /owner/accounts)
func (s Server) OwnerAccounts(ctx context.Context, request OwnerAccountsRequestObject) (OwnerAccountsResponseObject, error) {
	return nil, fmt.Errorf("not implemented: %s", "/owner/accounts") // TODO: Implement
}

// Get an account and its balances of each token type
// (GET /owner/accounts/{id})
func (s Server) OwnerAccount(ctx context.Context, request OwnerAccountRequestObject) (OwnerAccountResponseObject, error) {
	// balance of one type
	if request.Params.Code != nil {
		bal, err := s.fsc.Balance(ctx, string(request.Id), string(*request.Params.Code))
		if err != nil {
			return nil, err
		}
		return OwnerAccount200JSONResponse{AccountSuccessJSONResponse{
			Message: "ok",
			Payload: Account{
				Id:      request.Id,
				Balance: []Amount{{Code: bal.Code, Value: bal.Value}},
			},
		}}, nil
	}

	// all balances
	bals, err := s.fsc.Balances(ctx, string(request.Id))
	if err != nil {
		return nil, err
	}
	balance := make([]Amount, len(bals))
	for i := range bals {
		balance[i] = Amount{Code: bals[i].Code, Value: bals[i].Value}
	}
	return OwnerAccount200JSONResponse{AccountSuccessJSONResponse{
		Message: "ok",
		Payload: Account{
			Id:      request.Id,
			Balance: balance,
		}},
	}, nil
}

// Redeem (burn) tokens
// (POST /owner/accounts/{id}/redeem)
func (s Server) Redeem(ctx context.Context, request RedeemRequestObject) (RedeemResponseObject, error) {
	var msg string
	if request.Body.Message != nil {
		msg = *request.Body.Message
	}

	res, err := s.fsc.Redeem(ctx, request.Body.Amount.Code, request.Body.Amount.Value, request.Id, msg)
	if err != nil {
		return nil, err
	}

	return Redeem200JSONResponse{RedeemSuccessJSONResponse{
		Message: "ok",
		Payload: res,
	}}, nil
}

// Get all transactions for an account
// (GET /owner/accounts/{id}/transactions)
func (s Server) OwnerTransactions(ctx context.Context, request OwnerTransactionsRequestObject) (OwnerTransactionsResponseObject, error) {
	txs, err := s.fsc.GetTransactions(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	res := make([]TransactionRecord, len(txs))
	for i, tx := range txs {
		res[i] = TransactionRecord{
			Amount: Amount{
				Code:  string(tx.TokenType),
				Value: tx.Amount.Uint64(),
			},
			Id:        tx.TxID,
			Recipient: tx.RecipientEID,
			Sender:    tx.SenderEID,
			Status:    driver.TxStatusMessage[tx.Status],
			Timestamp: tx.Timestamp,
			Message:   string(tx.ApplicationMetadata["message"]),
		}
	}

	return OwnerTransactions200JSONResponse{
		TransactionsSuccessJSONResponse{
			Message: "ok",
			Payload: res,
		},
	}, nil
}

// Transfer tokens to another account
// (POST /owner/accounts/{id}/transfer)
func (s Server) Transfer(ctx context.Context, request TransferRequestObject) (TransferResponseObject, error) {
	var msg string
	if request.Body.Message != nil {
		msg = *request.Body.Message
	}

	res, err := s.fsc.Transfer(ctx, request.Body.Amount.Code, request.Body.Amount.Value, request.Id, request.Body.Counterparty.Account, request.Body.Counterparty.Node, msg)
	if err != nil {
		return nil, err
	}

	return Transfer200JSONResponse{TransferSuccessJSONResponse{
		Message: "ok",
		Payload: res,
	}}, nil
}

// Returns 200 if the service is healthy
// (GET /healthz)
func (s Server) Healthz(ctx context.Context, request HealthzRequestObject) (HealthzResponseObject, error) {
	return Healthz200JSONResponse{HealthSuccessJSONResponse{Message: "ok"}}, nil
}

// Returns 200 if the service is ready to accept calls
// (GET /readyz)
func (s Server) Readyz(ctx context.Context, request ReadyzRequestObject) (ReadyzResponseObject, error) {
	return Readyz200JSONResponse{HealthSuccessJSONResponse{Message: "ok"}}, nil
}
