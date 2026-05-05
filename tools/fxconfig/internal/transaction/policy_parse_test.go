/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseCertificateOrPublicKey_RSACertReturnsAlgorithmError(t *testing.T) {
	t.Parallel()

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-rsa"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
	require.NoError(t, err)

	_, gotErr := parseCertificateOrPublicKey(certDER)
	require.Error(t, gotErr)
	require.Contains(t, gotErr.Error(), "certificate uses")
	require.Contains(t, gotErr.Error(), "expected ECDSA")
}

func TestParseCertificateOrPublicKey_CorruptBytesReturnsParseError(t *testing.T) {
	t.Parallel()

	corrupt := []byte("this is definitely not valid ASN.1 DER data")
	_, gotErr := parseCertificateOrPublicKey(corrupt)
	require.Error(t, gotErr)
	require.True(
		t,
		strings.Contains(gotErr.Error(), "failed to parse public key") || strings.Contains(gotErr.Error(), "asn1"),
		"unexpected error: %v",
		gotErr,
	)
}

func TestParseCertificateOrPublicKey_ValidECDSAKeySucceeds(t *testing.T) {
	t.Parallel()

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubDER, err := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
	require.NoError(t, err)

	result, err := parseCertificateOrPublicKey(pubDER)
	require.NoError(t, err)
	require.NotEmpty(t, result)
}

func TestParseCertificateOrPublicKey_ValidECDSACertSucceeds(t *testing.T) {
	t.Parallel()

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-ecdsa"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &ecKey.PublicKey, ecKey)
	require.NoError(t, err)

	result, err := parseCertificateOrPublicKey(certDER)
	require.NoError(t, err)
	require.NotEmpty(t, result)
}
