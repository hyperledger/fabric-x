/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"errors"
	"testing"
	"time"

	msppb "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
)

// testSigningIdentity is a minimal mock of msp.SigningIdentity for app-layer tests.
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

func makeMSPProvider(
	sid msp.SigningIdentity,
	err error,
) *provider.Provider[msp.SigningIdentity, *config.MSPConfig] {
	return provider.New(func(_ *config.MSPConfig) (msp.SigningIdentity, error) {
		return sid, err
	}, &config.MSPConfig{LocalMspID: "Org1MSP", ConfigPath: "/fake"}, fakeValidationContext())
}

func TestEndorseTransaction_MspProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{MspProvider: makeMSPProvider(nil, errors.New("msp unavailable"))}

	_, err := a.EndorseTransaction(t.Context(), "tx-1", &applicationpb.Tx{
		Namespaces: []*applicationpb.TxNamespace{{NsId: "ns1"}},
	})
	require.Error(t, err)
}

func TestEndorseTransaction_Success(t *testing.T) {
	t.Parallel()

	a := &AdminApp{MspProvider: makeMSPProvider(&testSigningIdentity{}, nil)}
	tx := &applicationpb.Tx{
		Namespaces: []*applicationpb.TxNamespace{{NsId: "ns1"}},
	}

	result, err := a.EndorseTransaction(t.Context(), "tx-1", tx)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Endorsements, 1)
}
