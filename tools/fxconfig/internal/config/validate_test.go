/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// TestErrorIfEmpty tests the errorIfEmpty helper function.
func TestErrorIfEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "non-empty string",
			input:       "test",
			expectError: false,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: true,
		},
		{
			name:        "single space",
			input:       " ",
			expectError: true,
		},
		{
			name:        "tab character",
			input:       "\t",
			expectError: true,
		},
		{
			name:        "newline character",
			input:       "\n",
			expectError: true,
		},
		{
			name:        "mixed whitespace",
			input:       " \t\n ",
			expectError: true,
		},
		{
			name:        "string with leading/trailing spaces",
			input:       "  test  ",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := errorIfEmpty(tt.input, "test error message")
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestTLSConfigValidate_ExpiredCertificate verifies that validation rejects an expired client certificate.
func TestTLSConfigValidate_ExpiredCertificate(t *testing.T) {
	t.Parallel()

	keyPEM, certPEM, caCertPEM := generateSelfSignedCert(t, time.Now().Add(-48*time.Hour), time.Now().Add(-1*time.Hour))

	dir := t.TempDir()
	keyPath := writeTempFile(t, dir, "client.key", keyPEM)
	certPath := writeTempFile(t, dir, "client.crt", certPEM)
	caPath := writeTempFile(t, dir, "ca.crt", caCertPEM)

	enabled := true
	cfg := &TLSConfig{
		Enabled:        &enabled,
		ClientKeyPath:  keyPath,
		ClientCertPath: certPath,
		RootCertPaths:  []string{caPath},
	}

	err := cfg.Validate(validation.NewValidationContext())

	require.Error(t, err)
	require.Contains(t, err.Error(), "expired")
}

// TestTLSConfigValidate_ValidCertificate verifies that a currently valid certificate passes validation.
func TestTLSConfigValidate_ValidCertificate(t *testing.T) {
	t.Parallel()

	keyPEM, certPEM, caCertPEM := generateSelfSignedCert(t, time.Now().Add(-1*time.Hour), time.Now().Add(24*time.Hour))

	dir := t.TempDir()
	keyPath := writeTempFile(t, dir, "client.key", keyPEM)
	certPath := writeTempFile(t, dir, "client.crt", certPEM)
	caPath := writeTempFile(t, dir, "ca.crt", caCertPEM)

	enabled := true
	cfg := &TLSConfig{
		Enabled:        &enabled,
		ClientKeyPath:  keyPath,
		ClientCertPath: certPath,
		RootCertPaths:  []string{caPath},
	}

	err := cfg.Validate(validation.NewValidationContext())

	require.NoError(t, err)
}

// generateSelfSignedCert creates a self-signed ECDSA certificate with the given validity window.
// Returns the key, certificate, and CA certificate as PEM-encoded bytes.
func generateSelfSignedCert(t *testing.T, notBefore, notAfter time.Time) (keyPEM, certPEM, caCertPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	keyBytes, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	caCertPEM = certPEM

	return keyPEM, certPEM, caCertPEM
}

func writeTempFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := dir + "/" + name
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}
