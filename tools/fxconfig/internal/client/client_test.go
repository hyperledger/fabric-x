/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/connectivity"

	"github.com/hyperledger/fabric-x-common/tools/pkg/comm"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

const testLocalhost = "localhost"

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// createTempFile creates a temporary file with the given content.
func createTempFile(t *testing.T, content []byte) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.pem")
	require.NoError(t, err)
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	return tmpFile.Name()
}

// createUnreadableFile creates a file with no read permissions.
// Skips the test if running as root, since root bypasses file permissions.
func createUnreadableFile(t *testing.T) string {
	t.Helper()
	if os.Getuid() == 0 {
		t.Skip("skipping: root can read files regardless of permissions")
	}
	path := filepath.Join(t.TempDir(), "unreadable.pem")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0o000))
	return path
}

// generateCertificate creates a certificate signed by the CA and returns key and cert as PEM bytes.
//
//nolint:revive
func generateCertificate(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey,
	commonName string, dnsNames []string, extKeyUsage x509.ExtKeyUsage,
) (keyPEM, certPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   commonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{extKeyUsage},
	}
	for _, name := range dnsNames {
		if ip := net.ParseIP(name); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, name)
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	require.NoError(t, err)

	keyBytes, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return keyPEM, certPEM
}

// generateServerConfig creates a comm.ServerConfig with generated ECDSA certificates.
//
//nolint:revive
func generateServerConfig(t *testing.T, tlsMode string) (
	serverConfig comm.ServerConfig,
	caCertPath string,
	clientKeyPath string,
	clientCertPath string,
) {
	t.Helper()

	if tlsMode == "none" {
		return comm.ServerConfig{}, "", "", ""
	}

	tmpDir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	caCertPath = filepath.Join(tmpDir, "ca.pem")
	require.NoError(t, os.WriteFile(caCertPath, caCertPEM, 0o600))

	serverKey, serverCertPEM := generateCertificate(t, caTemplate, caKey, "server",
		[]string{testLocalhost, "127.0.0.1"}, x509.ExtKeyUsageServerAuth)

	serverConfig.SecOpts.UseTLS = true
	serverConfig.SecOpts.Certificate = serverCertPEM
	serverConfig.SecOpts.Key = serverKey

	if tlsMode == "mtls" {
		clientKey, clientCertPEM := generateCertificate(t, caTemplate, caKey, "client",
			nil, x509.ExtKeyUsageClientAuth)

		clientKeyPath = filepath.Join(tmpDir, "client-key.pem")
		require.NoError(t, os.WriteFile(clientKeyPath, clientKey, 0o600))

		clientCertPath = filepath.Join(tmpDir, "client-cert.pem")
		require.NoError(t, os.WriteFile(clientCertPath, clientCertPEM, 0o600))

		serverConfig.SecOpts.RequireClientCert = true
		serverConfig.SecOpts.ClientRootCAs = [][]byte{caCertPEM}
	}

	return serverConfig, caCertPath, clientKeyPath, clientCertPath
}

// startTestServer creates and starts a gRPC server with the provided configuration.
func startTestServer(t *testing.T, serverConfig comm.ServerConfig) (address string, cleanup func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address = lis.Addr().String()

	grpcServer, err := comm.NewGRPCServerFromListener(lis, serverConfig)
	require.NoError(t, err)

	go func() {
		_ = grpcServer.Start()
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		c, dialErr := net.DialTimeout("tcp", address, 10*time.Millisecond)
		if dialErr == nil {
			_ = c.Close()
			break
		}
		require.True(t, time.Now().Before(deadline), "server did not start within 5s")
		time.Sleep(time.Millisecond)
	}

	cleanup = func() {
		grpcServer.Stop()
	}

	return address, cleanup
}

func TestLoadFile_Success(t *testing.T) {
	t.Parallel()

	content := []byte("test content")
	path := createTempFile(t, content)

	result, err := loadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, result)
}

func TestLoadFile_Success_EmptyFile(t *testing.T) {
	t.Parallel()

	path := createTempFile(t, []byte{})

	result, err := loadFile(path)
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestLoadFile_Success_LargeFile(t *testing.T) {
	t.Parallel()

	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	path := createTempFile(t, content)

	result, err := loadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, result)
}

func TestLoadFile_Error_FileNotFound(t *testing.T) {
	t.Parallel()

	path := "/nonexistent/file.pem"

	result, err := loadFile(path)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed opening file")
	require.Contains(t, err.Error(), path)
}

func TestLoadFile_Error_Directory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := loadFile(dir)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed opening file")
}

func TestLoadFile_Error_Unreadable(t *testing.T) {
	t.Parallel()

	path := createUnreadableFile(t)

	result, err := loadFile(path)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed opening file")
}

func TestLoadFile_Error_EmptyPath(t *testing.T) {
	t.Parallel()

	result, err := loadFile("")
	require.Error(t, err)
	require.Nil(t, result)
}

func TestCreateSecOpts_NoTLS_ExplicitlyDisabled(t *testing.T) {
	t.Parallel()

	cfg := &config.TLSConfig{
		Enabled: boolPtr(false),
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.NotNil(t, secOpts)
	require.False(t, secOpts.UseTLS)
	require.Nil(t, secOpts.ServerRootCAs)
	require.Nil(t, secOpts.Key)
	require.Nil(t, secOpts.Certificate)
	require.Empty(t, secOpts.ServerNameOverride)
}

func TestCreateSecOpts_NoTLS_ConfigNil(t *testing.T) {
	t.Parallel()

	secOpts, err := createSecOpts(nil)
	require.NoError(t, err)
	require.NotNil(t, secOpts)
	require.False(t, secOpts.UseTLS)
}

func TestCreateSecOpts_NoTLS_EnabledFlagNil(t *testing.T) {
	t.Parallel()

	cfg := &config.TLSConfig{
		Enabled: nil,
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.NotNil(t, secOpts)
	require.False(t, secOpts.UseTLS)
}

func TestCreateSecOpts_TLS_Success(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")
	caPath := createTempFile(t, caCert)

	cfg := &config.TLSConfig{
		Enabled:            boolPtr(true),
		RootCertPaths:      []string{caPath},
		ServerNameOverride: "orderer.example.com",
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.NotNil(t, secOpts)
	require.True(t, secOpts.UseTLS)
	require.Len(t, secOpts.ServerRootCAs, 1)
	require.Equal(t, caCert, secOpts.ServerRootCAs[0])
	require.Equal(t, "orderer.example.com", secOpts.ServerNameOverride)
	require.False(t, secOpts.RequireClientCert)
	require.Nil(t, secOpts.Key)
	require.Nil(t, secOpts.Certificate)
}

func TestCreateSecOpts_TLS_MultipleRootCerts(t *testing.T) {
	t.Parallel()

	ca1 := []byte("-----BEGIN CERTIFICATE-----\nca1\n-----END CERTIFICATE-----")
	ca2 := []byte("-----BEGIN CERTIFICATE-----\nca2\n-----END CERTIFICATE-----")
	ca3 := []byte("-----BEGIN CERTIFICATE-----\nca3\n-----END CERTIFICATE-----")

	ca1Path := createTempFile(t, ca1)
	ca2Path := createTempFile(t, ca2)
	ca3Path := createTempFile(t, ca3)

	cfg := &config.TLSConfig{
		Enabled:       boolPtr(true),
		RootCertPaths: []string{ca1Path, ca2Path, ca3Path},
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.Len(t, secOpts.ServerRootCAs, 3)
	require.Equal(t, ca1, secOpts.ServerRootCAs[0])
	require.Equal(t, ca2, secOpts.ServerRootCAs[1])
	require.Equal(t, ca3, secOpts.ServerRootCAs[2])
}

func TestCreateSecOpts_TLS_NoServerNameOverride(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")
	caPath := createTempFile(t, caCert)

	cfg := &config.TLSConfig{
		Enabled:            boolPtr(true),
		RootCertPaths:      []string{caPath},
		ServerNameOverride: "",
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.True(t, secOpts.UseTLS)
	require.Empty(t, secOpts.ServerNameOverride)
}

func TestCreateSecOpts_TLS_Error_RootCertNotFound(t *testing.T) {
	t.Parallel()

	cfg := &config.TLSConfig{
		Enabled:       boolPtr(true),
		RootCertPaths: []string{"/nonexistent/ca-cert.pem"},
	}

	secOpts, err := createSecOpts(cfg)
	require.Error(t, err)
	require.Nil(t, secOpts)
	require.Contains(t, err.Error(), "failed opening file")
	require.Contains(t, err.Error(), "/nonexistent/ca-cert.pem")
}

func TestCreateSecOpts_TLS_Error_RootCertUnreadable(t *testing.T) {
	t.Parallel()

	unreadablePath := createUnreadableFile(t)

	cfg := &config.TLSConfig{
		Enabled:       boolPtr(true),
		RootCertPaths: []string{unreadablePath},
	}

	secOpts, err := createSecOpts(cfg)
	require.Error(t, err)
	require.Nil(t, secOpts)
	require.Contains(t, err.Error(), "failed opening file")
}

func TestCreateSecOpts_TLS_Error_EmptyRootCertPath(t *testing.T) {
	t.Parallel()

	cfg := &config.TLSConfig{
		Enabled:       boolPtr(true),
		RootCertPaths: []string{""},
	}

	secOpts, err := createSecOpts(cfg)
	require.Error(t, err)
	require.Nil(t, secOpts)
}

func TestCreateSecOpts_mTLS_Success(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----")
	clientKey := []byte("-----BEGIN EC PRIVATE KEY-----\nkey\n-----END EC PRIVATE KEY-----")
	clientCert := []byte("-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----")

	caPath := createTempFile(t, caCert)
	keyPath := createTempFile(t, clientKey)
	certPath := createTempFile(t, clientCert)

	cfg := &config.TLSConfig{
		Enabled:            boolPtr(true),
		RootCertPaths:      []string{caPath},
		ClientKeyPath:      keyPath,
		ClientCertPath:     certPath,
		ServerNameOverride: "orderer.example.com",
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.NotNil(t, secOpts)
	require.True(t, secOpts.UseTLS)
	require.Len(t, secOpts.ServerRootCAs, 1)
	require.Equal(t, caCert, secOpts.ServerRootCAs[0])
	require.True(t, secOpts.RequireClientCert)
	require.Equal(t, clientKey, secOpts.Key)
	require.Equal(t, clientCert, secOpts.Certificate)
	require.Equal(t, "orderer.example.com", secOpts.ServerNameOverride)
}

func TestCreateSecOpts_mTLS_MultipleRootCerts(t *testing.T) {
	t.Parallel()

	ca1 := []byte("-----BEGIN CERTIFICATE-----\nca1\n-----END CERTIFICATE-----")
	ca2 := []byte("-----BEGIN CERTIFICATE-----\nca2\n-----END CERTIFICATE-----")
	clientKey := []byte("-----BEGIN EC PRIVATE KEY-----\nkey\n-----END EC PRIVATE KEY-----")
	clientCert := []byte("-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----")

	ca1Path := createTempFile(t, ca1)
	ca2Path := createTempFile(t, ca2)
	keyPath := createTempFile(t, clientKey)
	certPath := createTempFile(t, clientCert)

	cfg := &config.TLSConfig{
		Enabled:        boolPtr(true),
		RootCertPaths:  []string{ca1Path, ca2Path},
		ClientKeyPath:  keyPath,
		ClientCertPath: certPath,
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.Len(t, secOpts.ServerRootCAs, 2)
	require.True(t, secOpts.RequireClientCert)
	require.NotNil(t, secOpts.Key)
	require.NotNil(t, secOpts.Certificate)
}

func TestCreateSecOpts_mTLS_Error_ClientKeyNotFound(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----")
	clientCert := []byte("-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----")

	caPath := createTempFile(t, caCert)
	certPath := createTempFile(t, clientCert)

	cfg := &config.TLSConfig{
		Enabled:        boolPtr(true),
		RootCertPaths:  []string{caPath},
		ClientKeyPath:  "/nonexistent/client-key.pem",
		ClientCertPath: certPath,
	}

	secOpts, err := createSecOpts(cfg)
	require.Error(t, err)
	require.Nil(t, secOpts)
	require.Contains(t, err.Error(), "failed opening file")
	require.Contains(t, err.Error(), "/nonexistent/client-key.pem")
}

func TestCreateSecOpts_mTLS_Error_ClientCertNotFound(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----")
	clientKey := []byte("-----BEGIN EC PRIVATE KEY-----\nkey\n-----END EC PRIVATE KEY-----")

	caPath := createTempFile(t, caCert)
	keyPath := createTempFile(t, clientKey)

	cfg := &config.TLSConfig{
		Enabled:        boolPtr(true),
		RootCertPaths:  []string{caPath},
		ClientKeyPath:  keyPath,
		ClientCertPath: "/nonexistent/client-cert.pem",
	}

	secOpts, err := createSecOpts(cfg)
	require.Error(t, err)
	require.Nil(t, secOpts)
	require.Contains(t, err.Error(), "failed opening file")
	require.Contains(t, err.Error(), "/nonexistent/client-cert.pem")
}

func TestCreateSecOpts_mTLS_Error_ClientKeyUnreadable(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----")
	clientCert := []byte("-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----")

	caPath := createTempFile(t, caCert)
	certPath := createTempFile(t, clientCert)
	unreadableKeyPath := createUnreadableFile(t)

	cfg := &config.TLSConfig{
		Enabled:        boolPtr(true),
		RootCertPaths:  []string{caPath},
		ClientKeyPath:  unreadableKeyPath,
		ClientCertPath: certPath,
	}

	secOpts, err := createSecOpts(cfg)
	require.Error(t, err)
	require.Nil(t, secOpts)
	require.Contains(t, err.Error(), "failed opening file")
}

func TestCreateSecOpts_mTLS_Error_ClientCertUnreadable(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----")
	clientKey := []byte("-----BEGIN EC PRIVATE KEY-----\nkey\n-----END EC PRIVATE KEY-----")

	caPath := createTempFile(t, caCert)
	keyPath := createTempFile(t, clientKey)
	unreadableCertPath := createUnreadableFile(t)

	cfg := &config.TLSConfig{
		Enabled:        boolPtr(true),
		RootCertPaths:  []string{caPath},
		ClientKeyPath:  keyPath,
		ClientCertPath: unreadableCertPath,
	}

	secOpts, err := createSecOpts(cfg)
	require.Error(t, err)
	require.Nil(t, secOpts)
	require.Contains(t, err.Error(), "failed opening file")
}

func TestCreateSecOpts_mTLS_OnlyClientKeyProvided(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----")
	clientKey := []byte("-----BEGIN EC PRIVATE KEY-----\nkey\n-----END EC PRIVATE KEY-----")

	caPath := createTempFile(t, caCert)
	keyPath := createTempFile(t, clientKey)

	cfg := &config.TLSConfig{
		Enabled:        boolPtr(true),
		RootCertPaths:  []string{caPath},
		ClientKeyPath:  keyPath,
		ClientCertPath: "",
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.True(t, secOpts.UseTLS)
	require.Nil(t, secOpts.Key)
	require.Nil(t, secOpts.Certificate)
}

func TestCreateSecOpts_mTLS_OnlyClientCertProvided(t *testing.T) {
	t.Parallel()

	caCert := []byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----")
	clientCert := []byte("-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----")

	caPath := createTempFile(t, caCert)
	certPath := createTempFile(t, clientCert)

	cfg := &config.TLSConfig{
		Enabled:        boolPtr(true),
		RootCertPaths:  []string{caPath},
		ClientKeyPath:  "",
		ClientCertPath: certPath,
	}

	secOpts, err := createSecOpts(cfg)
	require.NoError(t, err)
	require.True(t, secOpts.UseTLS)
	require.Nil(t, secOpts.Key)
	require.Nil(t, secOpts.Certificate)
}

func TestNewClientConn_NoTLS_Success(t *testing.T) {
	t.Parallel()

	serverConfig, _, _, _ := generateServerConfig(t, "none")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 5 * time.Second,
		TLS: &config.TLSConfig{
			Enabled: boolPtr(false),
		},
	}

	conn, err := newClientConn(cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close() //nolint:errcheck

	require.Equal(t, connectivity.Ready, conn.GetState())
}

func TestNewClientConn_NoTLS_TLSConfigNil(t *testing.T) {
	t.Parallel()

	serverConfig, _, _, _ := generateServerConfig(t, "none")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 5 * time.Second,
		TLS:               nil,
	}

	conn, err := newClientConn(cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close() //nolint:errcheck
}

func TestNewClientConn_TLS_Success(t *testing.T) {
	t.Parallel()

	serverConfig, caCertPath, _, _ := generateServerConfig(t, "tls")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 5 * time.Second,
		TLS: &config.TLSConfig{
			Enabled:            boolPtr(true),
			RootCertPaths:      []string{caCertPath},
			ServerNameOverride: testLocalhost,
		},
	}

	conn, err := newClientConn(cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close() //nolint:errcheck

	require.Equal(t, connectivity.Ready, conn.GetState())
}

func TestNewClientConn_TLS_MultipleRootCerts(t *testing.T) {
	t.Parallel()

	serverConfig1, caCertPath1, _, _ := generateServerConfig(t, "tls")
	_, caCertPath2, _, _ := generateServerConfig(t, "tls")

	address, cleanup := startTestServer(t, serverConfig1)
	defer cleanup()

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 5 * time.Second,
		TLS: &config.TLSConfig{
			Enabled:            boolPtr(true),
			RootCertPaths:      []string{caCertPath1, caCertPath2},
			ServerNameOverride: testLocalhost,
		},
	}

	conn, err := newClientConn(cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close() //nolint:errcheck
}

func TestNewClientConn_TLS_Error_RootCertNotFound(t *testing.T) {
	t.Parallel()

	serverConfig, _, _, _ := generateServerConfig(t, "tls")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 5 * time.Second,
		TLS: &config.TLSConfig{
			Enabled:       boolPtr(true),
			RootCertPaths: []string{"/nonexistent/ca-cert.pem"},
		},
	}

	conn, err := newClientConn(cfg)
	require.Error(t, err)
	require.Nil(t, conn)
	require.Contains(t, err.Error(), "failed opening file")
	require.Contains(t, err.Error(), "/nonexistent/ca-cert.pem")
}

func TestNewClientConn_TLS_Error_WrongCA(t *testing.T) {
	t.Parallel()

	serverConfig, _, _, _ := generateServerConfig(t, "tls")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	_, wrongCACertPath, _, _ := generateServerConfig(t, "tls")

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 2 * time.Second,
		TLS: &config.TLSConfig{
			Enabled:            boolPtr(true),
			RootCertPaths:      []string{wrongCACertPath},
			ServerNameOverride: testLocalhost,
		},
	}

	conn, err := newClientConn(cfg)
	require.Error(t, err)
	require.Nil(t, conn)
}

func TestNewClientConn_mTLS_Success(t *testing.T) {
	t.Parallel()

	serverConfig, caCertPath, clientKeyPath, clientCertPath := generateServerConfig(t, "mtls")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 5 * time.Second,
		TLS: &config.TLSConfig{
			Enabled:            boolPtr(true),
			RootCertPaths:      []string{caCertPath},
			ClientKeyPath:      clientKeyPath,
			ClientCertPath:     clientCertPath,
			ServerNameOverride: testLocalhost,
		},
	}

	conn, err := newClientConn(cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close() //nolint:errcheck

	require.Equal(t, connectivity.Ready, conn.GetState())
}

func TestNewClientConn_mTLS_NoClientCert(t *testing.T) {
	t.Parallel()

	serverConfig, caCertPath, _, _ := generateServerConfig(t, "mtls")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 2 * time.Second,
		TLS: &config.TLSConfig{
			Enabled:            boolPtr(true),
			RootCertPaths:      []string{caCertPath},
			ServerNameOverride: testLocalhost,
		},
	}

	conn, err := newClientConn(cfg)
	require.Error(t, err)
	require.Nil(t, conn)
}

func TestNewClientConn_TLS_ClientToNoTLSServer(t *testing.T) {
	t.Parallel()

	serverConfig, _, _, _ := generateServerConfig(t, "none")
	address, cleanup := startTestServer(t, serverConfig)
	defer cleanup()

	_, caCertPath, _, _ := generateServerConfig(t, "tls")

	cfg := &config.EndpointServiceConfig{
		Address:           address,
		ConnectionTimeout: 2 * time.Second,
		TLS: &config.TLSConfig{
			Enabled:       boolPtr(true),
			RootCertPaths: []string{caCertPath},
		},
	}

	conn, err := newClientConn(cfg)
	require.Error(t, err)
	require.Nil(t, conn)
}