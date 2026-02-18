/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	fxmsp "github.com/hyperledger/fabric-x-common/msp"
	"github.com/stretchr/testify/require"
)

// Mock signing identity for testing
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

func (m *mockSigningIdentity) GetIdentifier() *fxmsp.IdentityIdentifier {
	return &fxmsp.IdentityIdentifier{
		Mspid: m.mspID,
		Id:    "mock-id",
	}
}

func (m *mockSigningIdentity) Serialize() ([]byte, error) {
	return []byte("serialized-identity"), nil
}

func (m *mockSigningIdentity) SerializeWithIDOfCert() ([]byte, error) {
	return []byte("serialized-identity-with-cert-id"), nil
}

func (m *mockSigningIdentity) GetPublicVersion() fxmsp.Identity {
	return nil
}

func (m *mockSigningIdentity) Verify(msg []byte, sig []byte) error {
	return nil
}

func (m *mockSigningIdentity) GetOrganizationalUnits() []*fxmsp.OUIdentifier {
	return nil
}

func (m *mockSigningIdentity) Anonymous() bool {
	return false
}

func (m *mockSigningIdentity) ExpiresAt() time.Time {
	return time.Time{}
}

func (m *mockSigningIdentity) GetMSPIdentifier() string {
	return m.mspID
}

func (m *mockSigningIdentity) Validate() error {
	return nil
}

func (m *mockSigningIdentity) SatisfiesPrincipal(principal *msp.MSPPrincipal) error {
	return nil
}

// TestGetPubKeyFromPemData tests the getPubKeyFromPemData function
func TestGetPubKeyFromPemData(t *testing.T) {
	t.Parallel()

	// Generate a test ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Marshal public key to DER format
	pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	// Create PEM encoded public key
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyDER,
	})

	// Create a self-signed certificate
	template := &x509.Certificate{
		SerialNumber: nil,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	tests := []struct {
		name        string
		pemContent  []byte
		expectError bool
		description string
	}{
		{
			name:        "valid ECDSA public key",
			pemContent:  pubKeyPEM,
			expectError: false,
			description: "Should successfully extract public key from PEM",
		},
		{
			name:        "valid X.509 certificate",
			pemContent:  certPEM,
			expectError: false,
			description: "Should successfully extract public key from certificate",
		},
		{
			name: "multiple PEM blocks with key",
			pemContent: append(
				[]byte("-----BEGIN COMMENT-----\nSome comment\n-----END COMMENT-----\n"),
				pubKeyPEM...,
			),
			expectError: false,
			description: "Should find key in multiple PEM blocks",
		},
		{
			name:        "invalid PEM data",
			pemContent:  []byte("not a valid PEM"),
			expectError: true,
			description: "Should error on invalid PEM data",
		},
		{
			name:        "empty input",
			pemContent:  []byte(""),
			expectError: true,
			description: "Should error on empty input",
		},
		{
			name: "PEM without ECDSA key",
			pemContent: []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBogIBAAJBALRiMLAA
-----END RSA PRIVATE KEY-----`),
			expectError: true,
			description: "Should error when no ECDSA key found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := getPubKeyFromPemData(tt.pemContent)

			if tt.expectError {
				require.Error(t, err, tt.description)
				require.Nil(t, result)
			} else {
				require.NoError(t, err, tt.description)
				require.NotNil(t, result)
				// Verify result is valid PEM
				block, _ := pem.Decode(result)
				require.NotNil(t, block)
				require.Equal(t, "PUBLIC KEY", block.Type)
			}
		})
	}
}

// TestEndorse tests the endorse function
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

			result, err := endorse(tt.signer, tt.txID, tt.tx)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
				require.NotNil(t, result)

				// Verify endorsements structure
				require.Len(t, result.Endorsements, len(tt.tx.Namespaces), "Should have one endorsement per namespace")
				nsEndorsements := result.Endorsements

				// Verify each endorsement has signature
				for _, endorsementSet := range nsEndorsements {
					require.Equal(t, len(endorsementSet.EndorsementsWithIdentity), 1, "Endorsement set must have one endorsement")
					eid := endorsementSet.EndorsementsWithIdentity[0]
					require.NotNil(t, eid, "Endorsement should exist")
				}
			}
		})
	}
}

// TestCreateNamespacesTx tests the createNamespacesTx function
func TestCreateNamespacesTx(t *testing.T) {
	t.Parallel()

	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_ThresholdRule{
			ThresholdRule: &applicationpb.ThresholdRule{
				Scheme:    "ECDSA",
				PublicKey: []byte("test-public-key"),
			},
		},
	}

	tests := []struct {
		name        string
		nsPolicy    *applicationpb.NamespacePolicy
		nsID        string
		nsVersion   int
		description string
	}{
		{
			name:        "create new namespace (version -1)",
			nsPolicy:    nsPolicy,
			nsID:        "new-namespace",
			nsVersion:   -1,
			description: "Should create transaction for new namespace",
		},
		{
			name:        "update existing namespace (version 0)",
			nsPolicy:    nsPolicy,
			nsID:        "existing-namespace",
			nsVersion:   0,
			description: "Should create transaction for namespace update",
		},
		{
			name:        "update existing namespace (version 5)",
			nsPolicy:    nsPolicy,
			nsID:        "existing-namespace",
			nsVersion:   5,
			description: "Should create transaction for namespace update with higher version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := createNamespacesTx(tt.nsPolicy, tt.nsID, tt.nsVersion)

			require.NotNil(t, result, tt.description)
			require.Len(t, result.Namespaces, 1, "Should have one namespace entry")

			ns := result.Namespaces[0]
			require.Equal(t, "_meta", ns.NsId, "Should target meta-namespace")
			require.Equal(t, uint64(0), ns.NsVersion, "Meta-namespace version should be 0")
			require.Len(t, ns.ReadWrites, 1, "Should have one read-write entry")

			rw := ns.ReadWrites[0]
			require.Equal(t, []byte(tt.nsID), rw.Key, "Key should be namespace ID")
			require.NotEmpty(t, rw.Value, "Value should contain serialized policy")

			// Verify version is set correctly
			if tt.nsVersion >= 0 {
				require.NotNil(t, rw.Version, "Version should be set for updates")
				require.Equal(t, uint64(tt.nsVersion), *rw.Version, "Version should match input")
			} else {
				require.Nil(t, rw.Version, "Version should be nil for creates")
			}
		})
	}
}

// TestValidateVersion tests the validateVersion function
func TestValidateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		version     int
		expectError bool
		description string
	}{
		{
			name:        "version -1 (create)",
			version:     -1,
			expectError: false,
			description: "Version -1 should be valid for create operations",
		},
		{
			name:        "version -2 (invalid)",
			version:     -2,
			expectError: true,
			description: "Version -2 should be invalid",
		},
		{
			name:        "version -999 (invalid)",
			version:     -999,
			expectError: true,
			description: "Large negative version should be invalid",
		},
		{
			name:        "version 0 (update)",
			version:     0,
			expectError: false,
			description: "Version 0 should be valid for updates",
		},
		{
			name:        "version 1 (update)",
			version:     1,
			expectError: false,
			description: "Version 1 should be valid for updates",
		},
		{
			name:        "version 999999 (large positive)",
			version:     999999,
			expectError: false,
			description: "Large positive version should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nsCfg := NsConfig{
				Version: tt.version,
			}

			err := validateVersion(nsCfg)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// TestMustHavePolicy tests the mustHavePolicy function
func TestMustHavePolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		policyPath  string
		expectError bool
		description string
	}{
		{
			name:        "empty policy path",
			policyPath:  "",
			expectError: true,
			description: "Empty policy path should fail",
		},
		{
			name:        "valid policy path",
			policyPath:  "/path/to/policy.pem",
			expectError: false,
			description: "Valid policy path should pass",
		},
		{
			name:        "whitespace-only policy path",
			policyPath:  "   ",
			expectError: true,
			description: "Whitespace-only policy path should fail",
		},
		{
			name:        "policy path with spaces",
			policyPath:  "/path/with spaces/policy.pem",
			expectError: false,
			description: "Policy path with spaces should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nsCfg := NsConfig{
				ThresholdPolicyVerificationKeyPath: tt.policyPath,
			}

			err := mustHavePolicy(nsCfg)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// TestIsEmpty tests the isEmpty helper function
func TestIsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "non-empty string",
			input:    "test",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: true,
		},
		{
			name:     "single space",
			input:    " ",
			expected: true,
		},
		{
			name:     "tab character",
			input:    "\t",
			expected: true,
		},
		{
			name:     "newline character",
			input:    "\n",
			expected: true,
		},
		{
			name:     "mixed whitespace",
			input:    " \t\n ",
			expected: true,
		},
		{
			name:     "string with leading/trailing spaces",
			input:    "  test  ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isEmpty(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
