/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	msppb "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	ab "github.com/hyperledger/fabric-protos-go-apiv2/orderer"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// mockBroadcastStream implements grpc.BidiStreamingClient[cb.Envelope, ab.BroadcastResponse].
type mockBroadcastStream struct {
	sendErr         error
	recvResp        *ab.BroadcastResponse
	recvErr         error
	closeSendErr    error
	closeSendCalled bool
}

func (m *mockBroadcastStream) Send(_ *cb.Envelope) error            { return m.sendErr }
func (m *mockBroadcastStream) Recv() (*ab.BroadcastResponse, error) { return m.recvResp, m.recvErr }
func (*mockBroadcastStream) Header() (metadata.MD, error)           { return nil, nil }
func (*mockBroadcastStream) Trailer() metadata.MD                   { return nil }
func (m *mockBroadcastStream) CloseSend() error {
	m.closeSendCalled = true
	return m.closeSendErr
}
func (*mockBroadcastStream) Context() context.Context { return context.Background() }
func (*mockBroadcastStream) SendMsg(_ any) error      { return nil }
func (*mockBroadcastStream) RecvMsg(_ any) error      { return nil }

// mockAtomicBroadcastClient implements ab.AtomicBroadcastClient.
type mockAtomicBroadcastClient struct {
	stream       grpc.BidiStreamingClient[cb.Envelope, ab.BroadcastResponse]
	broadcastErr error
}

func (m *mockAtomicBroadcastClient) Broadcast(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[cb.Envelope, ab.BroadcastResponse], error) {
	return m.stream, m.broadcastErr
}

func (*mockAtomicBroadcastClient) Deliver(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[cb.Envelope, ab.DeliverResponse], error) {
	return nil, errors.New("not implemented")
}

// testSigningIdentity is a minimal mock of msp.SigningIdentity.
type testSigningIdentity struct {
	signErr error
}

func (m *testSigningIdentity) Sign(_ []byte) ([]byte, error) {
	if m.signErr != nil {
		return nil, m.signErr
	}
	return []byte("mock-sig"), nil
}

func (*testSigningIdentity) SerializeWithIDOfCert() ([]byte, error)         { return []byte{}, nil }
func (*testSigningIdentity) Serialize() ([]byte, error)                     { return []byte{}, nil }
func (*testSigningIdentity) GetCertificatePEM() ([]byte, error)             { return nil, nil }
func (*testSigningIdentity) GetIdentifier() *msp.IdentityIdentifier         { return nil }
func (*testSigningIdentity) GetPublicVersion() msp.Identity                 { return nil } //nolint:ireturn
func (*testSigningIdentity) Verify(_, _ []byte) error                       { return nil }
func (*testSigningIdentity) GetOrganizationalUnits() []*msp.OUIdentifier    { return nil }
func (*testSigningIdentity) Anonymous() bool                                { return false }
func (*testSigningIdentity) ExpiresAt() time.Time                           { return time.Time{} }
func (*testSigningIdentity) GetMSPIdentifier() string                       { return "TestMSP" }
func (*testSigningIdentity) Validate() error                                { return nil }
func (*testSigningIdentity) SatisfiesPrincipal(_ *msppb.MSPPrincipal) error { return nil }

func newTestOrdererClient(mock ab.AtomicBroadcastClient) *OrdererClient {
	return &OrdererClient{
		cfg: config.OrdererConfig{
			EndpointServiceConfig: config.EndpointServiceConfig{
				ConnectionTimeout: time.Second,
			},
			Channel: "mychannel",
		},
		client: mock,
	}
}

func someBroadcastTx() *applicationpb.Tx {
	return &applicationpb.Tx{Namespaces: []*applicationpb.TxNamespace{{NsId: "ns1"}}}
}

func TestOrdererClient_Broadcast_NilClient(t *testing.T) {
	t.Parallel()

	oc := &OrdererClient{}
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, "tx-1", someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_NilSigner(t *testing.T) {
	t.Parallel()

	// Stream is never opened because createSignedEnvelope returns early on nil signer.
	stream := &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_SUCCESS}}
	oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
	err := oc.Broadcast(t.Context(), nil, "tx-1", someBroadcastTx())
	require.Error(t, err)
	require.False(t, stream.closeSendCalled, "CloseSend must not be called when stream is never opened")
}

func TestOrdererClient_Broadcast_SignerError(t *testing.T) {
	t.Parallel()

	// Stream is never opened because createSignedEnvelope returns early on sign error.
	stream := &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_SUCCESS}}
	oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
	err := oc.Broadcast(t.Context(),
		&testSigningIdentity{signErr: errors.New("sign failed")}, "tx-1", someBroadcastTx())
	require.Error(t, err)
	require.False(t, stream.closeSendCalled, "CloseSend must not be called when stream is never opened")
}

func TestOrdererClient_Broadcast_StreamError(t *testing.T) {
	t.Parallel()

	oc := newTestOrdererClient(&mockAtomicBroadcastClient{
		broadcastErr: errors.New("stream unavailable"),
	})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, "tx-1", someBroadcastTx())
	require.Error(t, err)
	// Stream was never obtained, so CloseSend is not called.
}

func TestOrdererClient_Broadcast_SendError(t *testing.T) {
	t.Parallel()

	stream := &mockBroadcastStream{sendErr: errors.New("send failed")}
	oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, "tx-1", someBroadcastTx())
	require.Error(t, err)
	require.True(t, stream.closeSendCalled, "CloseSend must be called on Send error")
}

func TestOrdererClient_Broadcast_RecvError(t *testing.T) {
	t.Parallel()

	stream := &mockBroadcastStream{recvErr: errors.New("recv failed")}
	oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, "tx-1", someBroadcastTx())
	require.Error(t, err)
	require.True(t, stream.closeSendCalled, "CloseSend must be called on Recv error")
}

func TestOrdererClient_Broadcast_NonSuccessStatus(t *testing.T) {
	t.Parallel()

	stream := &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_BAD_REQUEST}}
	oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, "tx-1", someBroadcastTx())
	require.Error(t, err)
	require.True(t, stream.closeSendCalled, "CloseSend must be called on non-success status")
}

func TestOrdererClient_Broadcast_Success(t *testing.T) {
	t.Parallel()

	stream := &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_SUCCESS}}
	oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, "tx-1", someBroadcastTx())
	require.NoError(t, err)
	require.True(t, stream.closeSendCalled, "CloseSend must be called on success")
}

func TestOrdererClient_Close_CallsCloseFunc(t *testing.T) {
	t.Parallel()

	closed := false
	oc := &OrdererClient{closeF: func() { closed = true }}
	require.NoError(t, oc.Close())
	require.True(t, closed)
}

func TestOrdererClient_Close_NilFunc(t *testing.T) {
	t.Parallel()

	oc := &OrdererClient{}
	require.NoError(t, oc.Close())
}
