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
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
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

func (*mockSigningIdentity) Serialize() ([]byte, error) {
	return []byte("serialized-identity"), nil
}

func (*mockSigningIdentity) SerializeWithIDOfCert() ([]byte, error) {
	return []byte("serialized-identity-with-cert-id"), nil
}

func (*mockSigningIdentity) GetPublicVersion() msp.Identity { //nolint:ireturn
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

// TestEndorse tests the endorse function.
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
				mspID: "Org1MSP",
			},
			txID:        "tx-123",
			expectError: false,
			description: "Should successfully endorse single namespace",
		},
		{
			name: "successful endorsement with multiple namespaces",
			tx: &applicationpb.Tx{
				Namespaces: []*applicationpb.TxNamespace{
					{NsId: "ns1", NsVersion: 0},
					{NsId: "ns2", NsVersion: 1},
					{NsId: "ns3", NsVersion: 2},
				},
			},
			signer: &mockSigningIdentity{
				mspID: "Org1MSP",
			},
			txID:        "tx-456",
			expectError: false,
			description: "Should successfully endorse multiple namespaces",
		},
		{
			name: "signing failure",
			tx: &applicationpb.Tx{
				Namespaces: []*applicationpb.TxNamespace{
					{NsId: "test-ns", NsVersion: 0},
				},
			},
			signer: &mockSigningIdentity{
				mspID: "Org1MSP",
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

			result, err := transaction.Endorse(tt.signer, tt.txID, tt.tx)

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
