/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app/api"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

type Application interface {
	DeployNamespace(ctx context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, TxStatus, error)
	ListNamespaces(ctx context.Context) ([]NamespaceQueryResult, error)
	EndorseTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error)
	SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error
	SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) (TxStatus, error)
	MergeTransactions(ctx context.Context, txs []*applicationpb.Tx) (*applicationpb.Tx, error)
}

type AdminApp struct {
	Validators           validation.Context
	MspProvider          api.MspProvider
	QueryProvider        api.QueryProvider
	OrdererProvider      api.OrdererProvider
	NotificationProvider api.NotificationProvider
}
