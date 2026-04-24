// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package adapters provides the adapter apis.
package adapters

import (
	"context"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
)

// MspProvider supplies signing identities for transaction authentication.
type MspProvider interface {
	// Get returns the configured signing identity.
	Get() (msp.SigningIdentity, error)
	// Validate checks if the provider configuration is valid.
	Validate() error
}

// OrdererClient submits transactions to the ordering service.
type OrdererClient interface {
	// Broadcast sends a signed transaction to the ordering service.
	Broadcast(ctx context.Context, signer msp.SigningIdentity, txID string, tx *applicationpb.Tx) error
	// Close releases resources held by the client.
	Close() error
}

// OrdererProvider creates and validates OrdererClient instances.
type OrdererProvider interface {
	// Get returns a configured orderer client.
	Get() (OrdererClient, error)
	// Validate checks if the provider configuration is valid.
	Validate() error
}

// QueryClient retrieves state information from the committer query service.
type QueryClient interface {
	// GetNamespacePolicies fetches current namespace policy configurations.
	GetNamespacePolicies(ctx context.Context) (*applicationpb.NamespacePolicies, error)
	// Close releases resources held by the client.
	Close() error
}

// QueryProvider creates and validates QueryClient instances.
type QueryProvider interface {
	// Get returns a configured query client.
	Get() (QueryClient, error)
	// Validate checks if the provider configuration is valid.
	Validate() error
}

// NotificationClient subscribes to transaction confirmation events.
type NotificationClient interface {
	// Subscribe creates a subscription channel for the specified transaction ID.
	Subscribe(ctx context.Context, txID string) (chan int, error)
	// WaitForEvent blocks until a transaction event is received on the subscription.
	WaitForEvent(ctx context.Context, subscription chan int) (int, error)
	// Close releases resources held by the client.
	Close() error
}

// NotificationProvider creates and validates NotificationClient instances.
type NotificationProvider interface {
	// Get returns a configured notification client.
	Get() (NotificationClient, error)
	// Validate checks if the provider configuration is valid.
	Validate() error
}
