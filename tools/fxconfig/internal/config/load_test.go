/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "TestMSP")
	t.Setenv("FXCONFIG_MSP_CONFIGPATH", "/tmp/msp")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "localhost:7050")
	t.Setenv("FXCONFIG_QUERIES_ADDRESS", "localhost:7001")
	t.Setenv("FXCONFIG_NOTIFICATIONS_ADDRESS", "localhost:7002")

	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.Orderer.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, cfg.Queries.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, cfg.Notifications.ConnectionTimeout)
}

func TestLoad_ChannelDefault(t *testing.T) {
	t.Parallel()

	cfg, err := Load(WithOverride("msp.localMspID", "TestMSP"))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "mychannel", cfg.Orderer.Channel)
}

func TestLoad_WaitingTimeoutDefault(t *testing.T) {
	t.Parallel()

	cfg, err := Load(WithOverride("msp.localMspID", "TestMSP"))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.Notifications.WaitingTimeout)
}

func TestLoad_LoggingDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(WithOverride("msp.localMspID", "TestMSP"))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "error", cfg.Logging.Level)
	assert.Empty(t, cfg.Logging.Format)
}

func TestLoad_WithConfigFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: TestMSP
  configPath: /path/to/msp

orderer:
  address: orderer.example.com:7050
  connectionTimeout: 45s
  tls:
    rootCerts:
      - /path/to/ca.pem
    clientCert: /path/to/cert.pem
    clientKey: /path/to/key.pem
    serverNameOverride: orderer.override.com

queries:
  address: query.example.com:7001
  connectionTimeout: 60s

notifications:
  address: notify.example.com:7002
  connectionTimeout: 90s
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "TestMSP", cfg.MSP.LocalMspID)
	assert.Equal(t, "/path/to/msp", cfg.MSP.ConfigPath)

	assert.Equal(t, "orderer.example.com:7050", cfg.Orderer.Address)
	assert.Equal(t, 45*time.Second, cfg.Orderer.ConnectionTimeout)
	assert.Len(t, cfg.Orderer.TLS.RootCertPaths, 1)
	assert.Equal(t, "/path/to/ca.pem", cfg.Orderer.TLS.RootCertPaths[0])
	assert.Equal(t, "/path/to/cert.pem", cfg.Orderer.TLS.ClientCertPath)
	assert.Equal(t, "/path/to/key.pem", cfg.Orderer.TLS.ClientKeyPath)
	assert.Equal(t, "orderer.override.com", cfg.Orderer.TLS.ServerNameOverride)

	assert.Equal(t, "query.example.com:7001", cfg.Queries.Address)
	assert.Equal(t, 60*time.Second, cfg.Queries.ConnectionTimeout)

	assert.Equal(t, "notify.example.com:7002", cfg.Notifications.Address)
	assert.Equal(t, 90*time.Second, cfg.Notifications.ConnectionTimeout)
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "EnvMSP")
	t.Setenv("FXCONFIG_MSP_CONFIGPATH", "/env/msp")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "env-orderer:7050")
	t.Setenv("FXCONFIG_ORDERER_CONNECTIONTIMEOUT", "15s")
	t.Setenv("FXCONFIG_QUERIES_ADDRESS", "env-query:7001")

	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "EnvMSP", cfg.MSP.LocalMspID)
	assert.Equal(t, "/env/msp", cfg.MSP.ConfigPath)
	assert.Equal(t, "env-orderer:7050", cfg.Orderer.Address)
	assert.Equal(t, 15*time.Second, cfg.Orderer.ConnectionTimeout)
	assert.Equal(t, "env-query:7001", cfg.Queries.Address)
}

func TestLoad_WithOverride(t *testing.T) {
	t.Parallel()

	cfg, err := Load(
		WithOverride("msp.localMspID", "OverrideMSP"),
		WithOverride("orderer.address", "override-orderer:7050"),
		WithOverride("orderer.connectionTimeout", 20*time.Second),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "OverrideMSP", cfg.MSP.LocalMspID)
	assert.Equal(t, "override-orderer:7050", cfg.Orderer.Address)
	assert.Equal(t, 20*time.Second, cfg.Orderer.ConnectionTimeout)
}

func TestLoad_ConfigHierarchy(t *testing.T) {
	// file < env < override
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: FileMSP
  configPath: /file/msp

orderer:
  address: file-orderer:7050
  connectionTimeout: 45s
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "EnvMSP")

	cfg, err := Load(
		WithConfigFile(configPath),
		WithOverride("msp.localMspID", "OverrideMSP"),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "OverrideMSP", cfg.MSP.LocalMspID, "Override should take precedence")
	assert.Equal(t, "/file/msp", cfg.MSP.ConfigPath, "File value should be used")
	assert.Equal(t, "file-orderer:7050", cfg.Orderer.Address, "File value should be used")
}

func TestLoad_EnvVarsOverrideConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: FileMSP
  configPath: /file/msp

orderer:
  address: file-orderer:7050
  connectionTimeout: 60s

queries:
  address: file-query:7001

notifications:
  address: file-notify:7002
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "EnvMSP")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "env-orderer:7050")

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "EnvMSP", cfg.MSP.LocalMspID, "Env var should override file")
	assert.Equal(t, "env-orderer:7050", cfg.Orderer.Address, "Env var should override file")
	assert.Equal(t, "/file/msp", cfg.MSP.ConfigPath, "Should use file value when no env var")
}

func TestLoad_ConfigFileOverridesDefaults(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: TestMSP
  configPath: /tmp/msp

orderer:
  address: localhost:7050
  connectionTimeout: 60s

queries:
  address: localhost:7001
  connectionTimeout: 45s

notifications:
  address: localhost:7002
  connectionTimeout: 90s
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 60*time.Second, cfg.Orderer.ConnectionTimeout, "Should override default 30s")
	assert.Equal(t, 45*time.Second, cfg.Queries.ConnectionTimeout, "Should override default 30s")
	assert.Equal(t, 90*time.Second, cfg.Notifications.ConnectionTimeout, "Should override default 30s")
}

func TestLoad_ProjectConfig(t *testing.T) { //nolint:paralleltest
	// Note: This test is not parallel because it changes working directory

	tmpDir := t.TempDir()
	projectConfigDir := filepath.Join(tmpDir, ".fxconfig")
	err := os.MkdirAll(projectConfigDir, 0o750)
	require.NoError(t, err)

	configPath := filepath.Join(projectConfigDir, "config.yaml")
	configContent := `
msp:
  localMspID: ProjectMSP
  configPath: /project/msp
`
	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Chdir(tmpDir)

	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "ProjectMSP", cfg.MSP.LocalMspID)
	assert.Equal(t, "/project/msp", cfg.MSP.ConfigPath)
}

func TestLoad_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)

	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "TestMSP")
	t.Setenv("FXCONFIG_MSP_CONFIGPATH", "/tmp/msp")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "localhost:7050")
	t.Setenv("FXCONFIG_QUERIES_ADDRESS", "localhost:7001")
	t.Setenv("FXCONFIG_NOTIFICATIONS_ADDRESS", "localhost:7002")

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.Orderer.ConnectionTimeout)
}

func TestLoad_PartialConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: PartialMSP
  configPath: /partial/msp

orderer:
  address: partial-orderer:7050

queries:
  address: partial-query:7001

notifications:
  address: partial-notify:7002
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "PartialMSP", cfg.MSP.LocalMspID)
	assert.Equal(t, "/partial/msp", cfg.MSP.ConfigPath)
	assert.Equal(t, "partial-orderer:7050", cfg.Orderer.Address)
	assert.Equal(t, 30*time.Second, cfg.Orderer.ConnectionTimeout, "Should use default")
	assert.Equal(t, "partial-query:7001", cfg.Queries.Address)
}

func TestLoad_TLSConfiguration(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
orderer:
  tls:
    clientKey: /path/to/client.key
    clientCert: /path/to/client.crt
    rootCerts:
      - /path/to/ca.crt
    serverNameOverride: orderer.override.com
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "/path/to/client.key", cfg.Orderer.TLS.ClientKeyPath)
	assert.Equal(t, "/path/to/client.crt", cfg.Orderer.TLS.ClientCertPath)
	assert.Len(t, cfg.Orderer.TLS.RootCertPaths, 1)
	assert.Equal(t, "/path/to/ca.crt", cfg.Orderer.TLS.RootCertPaths[0])
	assert.Equal(t, "orderer.override.com", cfg.Orderer.TLS.ServerNameOverride)
}

// TestLoad_GlobalTLSInheritance verifies that a top-level tls: section is
// propagated to all services by ResolveTLS inside Load.
func TestLoad_GlobalTLSInheritance(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
tls:
  enabled: true
  rootCerts:
    - /path/to/ca.pem
  serverNameOverride: override.example.com

orderer:
  address: localhost:7050

queries:
  address: localhost:7001

notifications:
  address: localhost:7002
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.True(t, cfg.Orderer.TLS.IsEnabled(), "Orderer should inherit global TLS enabled")
	assert.Equal(t, []string{"/path/to/ca.pem"}, cfg.Orderer.TLS.RootCertPaths)
	assert.Equal(t, "override.example.com", cfg.Orderer.TLS.ServerNameOverride)

	assert.True(t, cfg.Queries.TLS.IsEnabled(), "Queries should inherit global TLS enabled")
	assert.Equal(t, []string{"/path/to/ca.pem"}, cfg.Queries.TLS.RootCertPaths)

	assert.True(t, cfg.Notifications.TLS.IsEnabled(), "Notifications should inherit global TLS enabled")
	assert.Equal(t, []string{"/path/to/ca.pem"}, cfg.Notifications.TLS.RootCertPaths)
}

// TestLoad_TLSEnabledFlag verifies that the tls.enabled boolean flag
// is correctly round-tripped through viper unmarshal.
func TestLoad_TLSEnabledFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
orderer:
  address: localhost:7050
  tls:
    enabled: true
    rootCerts:
      - /path/to/ca.pem

queries:
  address: localhost:7001
  tls:
    enabled: false

notifications:
  address: localhost:7002
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.True(t, cfg.Orderer.TLS.IsEnabled())
	assert.Equal(t, []string{"/path/to/ca.pem"}, cfg.Orderer.TLS.RootCertPaths)
	assert.False(t, cfg.Queries.TLS.IsEnabled())
	assert.False(t, cfg.Notifications.TLS.IsEnabled())
}

func TestLoad_LoggingConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
logging:
  level: debug
  format: json
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoad_NestedConfigStructure(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: TestMSP
  configPath: /tmp/msp

orderer:
  address: localhost:7050
  connectionTimeout: 30s
  tls:
    clientKey: /path/to/client.key
    clientCert: /path/to/client.crt
    rootCerts:
      - /path/to/ca.crt
    serverNameOverride: orderer.override.com

queries:
  address: localhost:7001
  connectionTimeout: 30s
  tls:
    rootCerts:
      - /path/to/query-ca.crt

notifications:
  address: localhost:7002
  connectionTimeout: 30s
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "/path/to/client.key", cfg.Orderer.TLS.ClientKeyPath)
	assert.Equal(t, "/path/to/client.crt", cfg.Orderer.TLS.ClientCertPath)
	assert.Len(t, cfg.Orderer.TLS.RootCertPaths, 1)
	assert.Equal(t, "/path/to/ca.crt", cfg.Orderer.TLS.RootCertPaths[0])
	assert.Equal(t, "orderer.override.com", cfg.Orderer.TLS.ServerNameOverride)
	assert.Len(t, cfg.Queries.TLS.RootCertPaths, 1)
	assert.Equal(t, "/path/to/query-ca.crt", cfg.Queries.TLS.RootCertPaths[0])
	assert.Empty(t, cfg.Queries.TLS.ClientKeyPath, "Should be empty when not specified")
}

// TestLoad_MalformedYAML tests that a syntactically invalid YAML file returns an error.
func TestLoad_MalformedYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	malformedYAML := `
msp:
  localMspID: TestMSP
  configPath: /tmp/msp
orderer:
  address: localhost:7050
  connectionTimeout: 30s
  invalid yaml here [[[
`
	err := os.WriteFile(configPath, []byte(malformedYAML), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.Error(t, err)
	require.Nil(t, cfg)
}

// TestLoad_FilePermissionError tests loading a file without read permissions.
func TestLoad_FilePermissionError(t *testing.T) {
	t.Parallel()

	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: TestMSP
  configPath: /tmp/msp
`
	err := os.WriteFile(configPath, []byte(configContent), 0o000)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "permission denied")
}

// TestLoad_InvalidDurationFormat tests loading config with an invalid duration string.
func TestLoad_InvalidDurationFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: TestMSP
  configPath: /tmp/msp

orderer:
  address: localhost:7050
  connectionTimeout: "invalid-duration"

queries:
  address: localhost:7001

notifications:
  address: localhost:7002
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.Error(t, err)
	require.Nil(t, cfg)
}

// TestLoad_NonExistentConfigFile tests that a missing config file returns an error.
func TestLoad_NonExistentConfigFile(t *testing.T) {
	t.Parallel()

	cfg, err := Load(WithConfigFile("/non/existent/path/config.yaml"))

	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "error reading config file")
}

// TestLoad_LargeTimeoutValues tests loading config with very large timeout values.
func TestLoad_LargeTimeoutValues(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: TestMSP
  configPath: /tmp/msp

orderer:
  address: localhost:7050
  connectionTimeout: 24h

queries:
  address: localhost:7001
  connectionTimeout: 168h

notifications:
  address: localhost:7002
  connectionTimeout: 720h
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 24*time.Hour, cfg.Orderer.ConnectionTimeout)
	assert.Equal(t, 168*time.Hour, cfg.Queries.ConnectionTimeout)
	assert.Equal(t, 720*time.Hour, cfg.Notifications.ConnectionTimeout)
}

// TestLoad_SpecialCharactersInPaths tests loading config with special characters in paths.
func TestLoad_SpecialCharactersInPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: "Test-MSP_123"
  configPath: "/path/with spaces/and-dashes/msp_config"

orderer:
  address: "orderer-node.example.com:7050"
  connectionTimeout: 30s
  tls:
    clientKey: "/path/with spaces/client.key"
    clientCert: "/path/with spaces/client.crt"
    rootCerts:
      - "/path/with spaces/ca.crt"
    serverNameOverride: "orderer-override.example.com"

queries:
  address: "query-service.example.com:7001"
  connectionTimeout: 30s

notifications:
  address: "notify-service.example.com:7002"
  connectionTimeout: 30s
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load(WithConfigFile(configPath))

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "Test-MSP_123", cfg.MSP.LocalMspID)
	assert.Equal(t, "/path/with spaces/and-dashes/msp_config", cfg.MSP.ConfigPath)
	assert.Equal(t, "orderer-node.example.com:7050", cfg.Orderer.Address)
	assert.Equal(t, "/path/with spaces/client.key", cfg.Orderer.TLS.ClientKeyPath)
	assert.Equal(t, "orderer-override.example.com", cfg.Orderer.TLS.ServerNameOverride)
}
