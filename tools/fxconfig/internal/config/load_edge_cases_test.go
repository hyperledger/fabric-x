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

// TestLoad_MalformedYAML tests loading a malformed YAML file.
func TestLoad_MalformedYAML(t *testing.T) {
	t.Parallel()

	// Setup - Create malformed YAML file
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should fail with unmarshal error
	require.Error(t, err)
	require.Nil(t, cfg)
}

// TestLoad_FilePermissionError tests loading a file without read permissions.
func TestLoad_FilePermissionError(t *testing.T) {
	t.Parallel()

	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Setup - Create file without read permissions
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: TestMSP
  configPath: /tmp/msp
`
	err := os.WriteFile(configPath, []byte(configContent), 0o000) // No permissions
	require.NoError(t, err)

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should fail with permission error
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "permission denied")
}

// TestLoad_InvalidDurationFormat tests loading config with invalid duration.
func TestLoad_InvalidDurationFormat(t *testing.T) {
	t.Parallel()

	// Setup - Create config with invalid duration
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should fail with unmarshal error
	require.Error(t, err)
	require.Nil(t, cfg)
}

// TestLoad_LargeTimeoutValues tests loading config with very large timeout values.
func TestLoad_LargeTimeoutValues(t *testing.T) {
	t.Parallel()

	// Setup - Create config with large timeouts
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should succeed
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 24*time.Hour, cfg.Orderer.ConnectionTimeout)
	assert.Equal(t, 168*time.Hour, cfg.Queries.ConnectionTimeout)
	assert.Equal(t, 720*time.Hour, cfg.Notifications.ConnectionTimeout)
}

// TestLoad_SpecialCharactersInPaths tests loading config with special characters in paths.
func TestLoad_SpecialCharactersInPaths(t *testing.T) {
	t.Parallel()

	// Setup - Create config with special characters
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should succeed
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "Test-MSP_123", cfg.MSP.LocalMspID)
	assert.Equal(t, "/path/with spaces/and-dashes/msp_config", cfg.MSP.ConfigPath)
	assert.Equal(t, "orderer-node.example.com:7050", cfg.Orderer.Address)
	assert.Equal(t, "/path/with spaces/client.key", cfg.Orderer.TLS.ClientKeyPath)
	assert.Equal(t, "orderer-override.example.com", cfg.Orderer.TLS.ServerNameOverride)
}

// TestLoad_NonExistentConfigFile tests loading a non-existent config file.
func TestLoad_NonExistentConfigFile(t *testing.T) {
	t.Parallel()

	// Execute - Try to load non-existent file
	cfg, err := Load(WithConfigFile("/non/existent/path/config.yaml"))

	// Assert - Should fail
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "error reading config file")
}

// TestLoad_EmptyConfigPath tests loading with empty config path.
func TestLoad_EmptyConfigPath(t *testing.T) {
	// Setup - Set required fields via environment
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "TestMSP")
	t.Setenv("FXCONFIG_MSP_CONFIGPATH", "/tmp/msp")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "localhost:7050")
	t.Setenv("FXCONFIG_QUERIES_ADDRESS", "localhost:7001")
	t.Setenv("FXCONFIG_NOTIFICATIONS_ADDRESS", "localhost:7002")

	// Execute - Load without config file
	cfg, err := Load()

	// Assert - Should succeed using env vars
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "TestMSP", cfg.MSP.LocalMspID)
}

// TestLoad_ConfigFileOverridesDefaults tests that config file overrides defaults.
func TestLoad_ConfigFileOverridesDefaults(t *testing.T) {
	t.Parallel()

	// Setup - Create config file with custom timeout
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should use config file values, not defaults
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 60*time.Second, cfg.Orderer.ConnectionTimeout, "Should override default 30s")
	assert.Equal(t, 45*time.Second, cfg.Queries.ConnectionTimeout, "Should override default 30s")
	assert.Equal(t, 90*time.Second, cfg.Notifications.ConnectionTimeout, "Should override default 30s")
}

// TestLoad_EnvVarsOverrideConfigFile tests that env vars override config file.
func TestLoad_EnvVarsOverrideConfigFile(t *testing.T) {
	// Setup - Create config file
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

	// Setup - Set env vars that should override
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "EnvMSP")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "env-orderer:7050")

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Env vars should override config file
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "EnvMSP", cfg.MSP.LocalMspID, "Env var should override file")
	assert.Equal(t, "env-orderer:7050", cfg.Orderer.Address, "Env var should override file")
	assert.Equal(t, "/file/msp", cfg.MSP.ConfigPath, "Should use file value when no env var")
}

// TestLoad_WithOverrideOption tests that WithOverride option has highest precedence.
func TestLoad_WithOverrideOption(t *testing.T) {
	// Setup - Create config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: FileMSP
  configPath: /file/msp

orderer:
  address: file-orderer:7050

queries:
  address: file-query:7001

notifications:
  address: file-notify:7002
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Setup - Set env var
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "EnvMSP")

	// Execute - Use WithOverride which should have highest precedence
	cfg, err := Load(
		WithConfigFile(configPath),
		WithOverride("msp.localMspID", "OverrideMSP"),
		WithOverride("orderer.address", "override-orderer:7050"),
	)

	// Assert - WithOverride should win
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "OverrideMSP", cfg.MSP.LocalMspID, "WithOverride should have highest precedence")
	assert.Equal(t, "override-orderer:7050", cfg.Orderer.Address, "WithOverride should override file")
}

// TestLoad_NestedConfigStructure tests loading deeply nested config.
func TestLoad_NestedConfigStructure(t *testing.T) {
	t.Parallel()

	// Setup - Create config with all nested TLS fields
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - All nested fields should be loaded
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
