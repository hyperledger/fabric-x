/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"

	msppb "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	fxmsppb "github.com/hyperledger/fabric-x-common/api/msppb"
	"github.com/hyperledger/fabric-x-common/msp"
)

const (
	testNs1   = "ns1"
	testNs2   = "ns2"
	testNs3   = "ns3"
	testOrg1  = "Org1MSP"
)

// mockSigningIdentity is a mock signing identity for testing.
type mockSigningIdentity struct {
	signFunc    func([]byte) ([]byte, error)
	certPEMFunc func() ([]byte, error)
	mspID       string
}

func (m *mockSigningIdentity) Sign(msg []byte) ([]byte, error) {
	if m.signFunc != nil {
		return m.signFunc(msg)
	}
	return []byte("mock-signature"), nil
}

func (m *mockSigningIdentity) GetCertificatePEM() ([]byte, error) {
	if m.certPEMFunc != nil {
		return m.certPEMFunc()
	}
	return []byte("-----BEGIN CERTIFICATE-----\nMOCK\n-----END CERTIFICATE-----"), nil
}

func (m *mockSigningIdentity) GetIdentifier() *msp.IdentityIdentifier {
	return &msp.IdentityIdentifier{
		Mspid: m.mspID,
		Id:    "mock-id",
	}
}

func (m *mockSigningIdentity) Serialize() ([]byte, error) {
	sid := &fxmsppb.Identity{
		MspId: m.mspID,
	}
	b, err := proto.Marshal(sid)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (*mockSigningIdentity) SerializeWithIDOfCert() ([]byte, error) {
	return []byte("serialized-identity-with-cert-id"), nil
}

//nolint:ireturn
func (*mockSigningIdentity) GetPublicVersion() msp.Identity {
	return nil
}

func (*mockSigningIdentity) Verify(_, _ []byte) error {
	return nil
}

func (*mockSigningIdentity) GetOrganizationalUnits() []*msp.OUIdentifier {
	return nil
}

func (*mockSigningIdentity) Anonymous() bool {
	return false
}

func (*mockSigningIdentity) ExpiresAt() time.Time {
	return time.Time{}
}

func (m *mockSigningIdentity) GetMSPIdentifier() string {
	return m.mspID
}

func (*mockSigningIdentity) Validate() error {
	return nil
}

func (*mockSigningIdentity) SatisfiesPrincipal(_ *msppb.MSPPrincipal) error {
	return nil
}

// TestEndorse tests the Endorse function.
func TestEndorse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tx          *applicationpb.Tx
		signer      *mockSigningIdentity
		txID        string
		expectError bool
		description string
	}{
		{
			name: "successful endorsement with single namespace",
			tx: &applicationpb.Tx{
				Namespaces: []*applicationpb.TxNamespace{
					{
						NsId:      "test-ns",
						NsVersion: 0,
					},
				},
			},
			signer: &mockSigningIdentity{
				mspID: testOrg1,
			},
			txID:        "tx-123",
			expectError: false,
			description: "Should successfully Endorse single namespace",
		},
		{
			name: "successful endorsement with multiple namespaces",
			tx: &applicationpb.Tx{
				Namespaces: []*applicationpb.TxNamespace{
					{NsId: testNs1, NsVersion: 0},
					{NsId: testNs2, NsVersion: 1},
					{NsId: testNs3, NsVersion: 2},
				},
			},
			signer: &mockSigningIdentity{
				mspID: testOrg1,
			},
			txID:        "tx-456",
			expectError: false,
			description: "Should successfully Endorse multiple namespaces",
		},
		{
			name: "signing failure",
			tx: &applicationpb.Tx{
				Namespaces: []*applicationpb.TxNamespace{
					{NsId: "test-ns", NsVersion: 0},
				},
			},
			signer: &mockSigningIdentity{
				mspID: testOrg1,
				signFunc: func([]byte) ([]byte, error) {
					return nil, errors.New("signing failed")
				},
			},
			txID:        "tx-789",
			expectError: true,
			description: "Should error when signing fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Endorse(tt.signer, tt.txID, tt.tx)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
				require.NotNil(t, result)

				// Verify endorsements structure
				require.Len(t, result.Endorsements, len(tt.tx.Namespaces),
					"Should have one endorsement per namespace")

				// Verify each endorsement has signature
				for _, endorsementSet := range result.Endorsements {
					require.Len(t, endorsementSet.EndorsementsWithIdentity, 1,
						"Endorsement set must have one endorsement")
				}
			}
		})
	}
}

func TestGenerateTxID(t *testing.T) {
	t.Parallel()

	t.Run("returns valid hex string", func(t *testing.T) {
		t.Parallel()

		id := GenerateTxID()
		require.NotEmpty(t, id)

		// SHA-256 produces 32 bytes → 64 hex characters
		require.Len(t, id, 64)
		_, err := hex.DecodeString(id)
		require.NoError(t, err, "txID should be valid hex")
	})

	t.Run("each call returns a unique ID", func(t *testing.T) {
		t.Parallel()
		a, b := GenerateTxID(), GenerateTxID()
		require.NotEqual(t, a, b)
	})
}
