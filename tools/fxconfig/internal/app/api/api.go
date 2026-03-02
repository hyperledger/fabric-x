package api

import (
	"context"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
)

type MspProvider interface {
	Get() (msp.SigningIdentity, error)
	Validate() error
}

// OrdererClient is the outbound port for submitting transactions to the ordering service.
// txID is intentionally absent: envelope construction (including any ID the transport layer
// needs) is the adapter's responsibility, not the domain's.
type OrdererClient interface {
	Broadcast(ctx context.Context, signer msp.SigningIdentity, txID string, tx *applicationpb.Tx) error
}

type OrdererProvider interface {
	Get() (OrdererClient, error)
	Validate() error
}

type QueryClient interface {
	GetNamespacePolicies(ctx context.Context) (*applicationpb.NamespacePolicies, error)
}
type QueryProvider interface {
	Get() (QueryClient, error)
	Validate() error
}
