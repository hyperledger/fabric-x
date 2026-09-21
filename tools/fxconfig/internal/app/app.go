/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package app provides namespace lifecycle management for Fabric-X.
// It handles creation, deployment, endorsement, and querying of namespaces.
package app

import (
	"context"
	"errors"
	"sync"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
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
	Close() error
}

// AdminApp implements Application interface with provider-based dependencies.
type AdminApp struct {
	Validators           validation.Context
	MspProvider          *provider.Provider[msp.SigningIdentity, *config.MSPConfig]
	QueryProvider        *provider.Provider[adapters.QueryClient, *config.QueriesConfig]
	OrdererProvider      *provider.Provider[adapters.OrdererClient, *config.OrdererConfig]
	NotificationProvider *provider.Provider[adapters.NotificationClient, *config.NotificationsConfig]
	closeOnce            sync.Once
	closeErr             error
}

// Close releases provider-managed resources owned by the application.
func (d *AdminApp) Close() error {
	d.closeOnce.Do(func() {
		var errs []error

		if d.NotificationProvider != nil {
			errs = append(errs, d.NotificationProvider.Close())
		}
		if d.OrdererProvider != nil {
			errs = append(errs, d.OrdererProvider.Close())
		}
		if d.QueryProvider != nil {
			errs = append(errs, d.QueryProvider.Close())
		}
		if d.MspProvider != nil {
			errs = append(errs, d.MspProvider.Close())
		}

		d.closeErr = errors.Join(errs...)
	})

	return d.closeErr
}
