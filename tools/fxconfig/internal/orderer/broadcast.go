/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orderer

import (
	"context"
	"fmt"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	ab "github.com/hyperledger/fabric-protos-go-apiv2/orderer"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/cmd/common/comm"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// Broadcast sends the signed envelope to the ordering service.
// It establishes a gRPC connection, sends the envelope, and waits for acknowledgment.
func Broadcast(
	cfg config.OrdererConfig,
	sid msp.SigningIdentity,
	txID string,
	tx *applicationpb.Tx,
) error {
	signatureHdr := protoutil.NewSignatureHeaderOrPanic(sid)

	// prepare transaction submission
	// create signed envelope
	channelHdr := protoutil.MakeChannelHeader(cb.HeaderType_MESSAGE, 0, cfg.Channel, 0)
	channelHdr.TxId = txID

	env, err := createSignedEnvelope(sid, tx, channelHdr, signatureHdr)
	if err != nil {
		return err
	}

	return send(cfg, env)
}

// createSignedEnvelope wraps the transaction in a signed envelope for submission to the orderer.
// The envelope contains the channel header, signature header, and transaction payload.
func createSignedEnvelope(
	signer msp.SigningIdentity,
	tx *applicationpb.Tx,
	channelHdr *cb.ChannelHeader,
	signatureHdr *cb.SignatureHeader,
) (*cb.Envelope, error) {
	payloadHdr := protoutil.MakePayloadHeader(channelHdr, signatureHdr)
	txBytes := protoutil.MarshalOrPanic(tx)

	payloadBytes := protoutil.MarshalOrPanic(
		&cb.Payload{
			Header: payloadHdr,
			Data:   txBytes,
		},
	)

	var sig []byte
	if signer != nil {
		var err error
		sig, err = signer.Sign(payloadBytes)
		if err != nil {
			return nil, err
		}
	}

	return &cb.Envelope{
		Payload:   payloadBytes,
		Signature: sig,
	}, nil
}

func send(cfg config.OrdererConfig, env *cb.Envelope) error {
	clientCfg := comm.Config{
		Timeout: cfg.ConnectionTimeout,
	}

	// TLS config
	if cfg.TLS.IsEnabled() {
		clientCfg.CertPath = cfg.TLS.ClientCertPath
		clientCfg.KeyPath = cfg.TLS.ClientKeyPath
		clientCfg.PeerCACertPath = cfg.TLS.RootCertPaths[0]
	}

	cl, err := comm.NewClient(clientCfg)
	if err != nil {
		return fmt.Errorf("cannot get grpc client: %w", err)
	}

	conn, err := cl.NewDialer(cfg.Address)()
	if err != nil {
		return fmt.Errorf("cannot get grpc client: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	occ := ab.NewAtomicBroadcastClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	abc, err := occ.Broadcast(ctx)
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
