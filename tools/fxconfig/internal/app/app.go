/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package app provides namespace lifecycle management for Fabric-X.
// It handles creation, deployment, endorsement, and querying of namespaces.
package app

import (
	"context"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/api"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// Application defines the core namespace management operations.
type Application interface {
	DeployNamespace(ctx context.Context, input *DeployNamespaceInput) (*DeployNamespaceOutput, TxStatus, error)
	ListNamespaces(ctx context.Context) ([]NamespaceQueryResult, error)
	EndorseTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error)
	SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error
	SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) (TxStatus, error)
	MergeTransactions(ctx context.Context, txs []*applicationpb.Tx) (*applicationpb.Tx, error)
}

// AdminApp implements Application interface with provider-based dependencies.
type AdminApp struct {
	Validators           validation.Context
	MspProvider          api.MspProvider
	QueryProvider        api.QueryProvider
	OrdererProvider      api.OrdererProvider
	NotificationProvider api.NotificationProvider
}
