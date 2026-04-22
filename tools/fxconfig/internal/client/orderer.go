/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	ab "github.com/hyperledger/fabric-protos-go-apiv2/orderer"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// OrdererClient provides a gRPC client for submitting transactions to the Fabric-X ordering service.
// It handles connection management, TLS configuration, and transaction envelope creation.
type OrdererClient struct {
	cfg    config.OrdererConfig
	client ab.AtomicBroadcastClient
	closeF func()
}

// NewOrdererClient creates a new orderer client with the provided configuration and signing identity.
// It establishes a gRPC connection with optional TLS and returns an error if connection fails.
func NewOrdererClient(cfg config.OrdererConfig) (*OrdererClient, error) {
	conn, err := newClientConn(&cfg.EndpointServiceConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot get grpc client: %w", err)
	}

	return &OrdererClient{
		cfg:    cfg,
		client: ab.NewAtomicBroadcastClient(conn),
		closeF: func() {
			_ = conn.Close()
		},
	}, nil
}

// Close terminates the gRPC connection to the ordering service.
func (oc *OrdererClient) Close() error {
	if oc.closeF != nil {
		oc.closeF()
	}
	return nil
}

// Broadcast sends the signed envelope to the ordering service.
// It establishes a gRPC connection, sends the envelope, and waits for acknowledgment.
func (oc *OrdererClient) Broadcast(
	ctx context.Context,
	signer msp.SigningIdentity,
	txID string,
	tx *applicationpb.Tx,
) error {
	env, err := oc.createSignedEnvelope(signer, txID, tx)
	if err != nil {
		return err
	}

	return oc.send(ctx, env)
}

// ConnectionTimeout returns the configured connection timeout for this orderer client.
func (oc *OrdererClient) ConnectionTimeout() time.Duration {
	return oc.cfg.ConnectionTimeout
}

// send transmits the envelope to the orderer and waits for acknowledgment.
// Returns an error if the orderer rejects the transaction or communication fails.
func (oc *OrdererClient) send(ctx context.Context, env *cb.Envelope) error {
	if oc.client == nil {
		return errors.New("require client")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	abc, err := oc.client.Broadcast(ctx)
	if err != nil {
		return err
	}

	err = abc.Send(env)
	if err != nil {
		return err
	}

	status, err := abc.Recv()
	if err != nil {
		return err
	}

	if status.GetStatus() != cb.Status_SUCCESS {
		return fmt.Errorf("got error %#v", status.GetStatus())
	}

	return nil
}

// createSignedEnvelope wraps the transaction in a signed envelope for submission to the orderer.
// The envelope contains the channel header, signature header, and transaction payload.
func (oc *OrdererClient) createSignedEnvelope(
	signer msp.SigningIdentity,
	txID string,
	tx *applicationpb.Tx,
) (*cb.Envelope, error) {
	if signer == nil {
		return nil, errors.New("require Signer")
	}

	signatureHdr := protoutil.NewSignatureHeaderOrPanic(signer)

	// prepare transaction submission
	// create signed envelope
	channelHdr := protoutil.MakeChannelHeader(cb.HeaderType_MESSAGE, 0, oc.cfg.Channel, 0)
	channelHdr.TxId = txID

	payloadHdr := protoutil.MakePayloadHeader(channelHdr, signatureHdr)
	txBytes := protoutil.MarshalOrPanic(tx)

	payloadBytes := protoutil.MarshalOrPanic(
		&cb.Payload{
			Header: payloadHdr,
			Data:   txBytes,
		},
	)

	sig, err := signer.Sign(payloadBytes)
	if err != nil {
		return nil, err
	}

	return &cb.Envelope{
		Payload:   payloadBytes,
		Signature: sig,
	}, nil
}
