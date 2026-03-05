/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// TestGetPubKeyFromPemData tests the getPubKeyFromPemData function.
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

func TestCreateMspPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"valid OR expression", "OR('Org1MSP.member')", false},
		{"valid AND expression", "AND('Org1MSP.member', 'Org2MSP.member')", false},
		{"invalid expression", "NOT_VALID", true},
		{"empty expression", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			policy, err := CreateMspPolicy(tt.expr)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, policy)
			} else {
				require.NoError(t, err)
				require.NotNil(t, policy)
				_, ok := policy.Rule.(*applicationpb.NamespacePolicy_MspRule)
				require.True(t, ok, "expected MspRule policy type")
			}
		})
	}
}

func TestCreateThresholdPolicy(t *testing.T) {
	t.Parallel()

	// generate an ECDSA key and write it to a temp file
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "key.pem")
	require.NoError(t, os.WriteFile(keyFile, pubKeyPEM, 0o600))

	t.Run("valid key file", func(t *testing.T) {
		t.Parallel()

		policy, err := CreateThresholdPolicy(keyFile)
		require.NoError(t, err)
		require.NotNil(t, policy)

		rule, ok := policy.Rule.(*applicationpb.NamespacePolicy_ThresholdRule)
		require.True(t, ok, "expected ThresholdRule policy type")
		require.Equal(t, "ECDSA", rule.ThresholdRule.GetScheme())
		require.NotEmpty(t, rule.ThresholdRule.GetPublicKey())
	})

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()

		policy, err := CreateThresholdPolicy(filepath.Join(tmpDir, "missing.pem"))
		require.Error(t, err)
		require.Nil(t, policy)
	})
}
