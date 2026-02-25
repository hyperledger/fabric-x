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
	// Setup - Set required fields via environment variables
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "TestMSP")
	t.Setenv("FXCONFIG_MSP_CONFIGPATH", "/tmp/msp")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "localhost:7050")
	t.Setenv("FXCONFIG_QUERIES_ADDRESS", "localhost:7001")
	t.Setenv("FXCONFIG_NOTIFICATIONS_ADDRESS", "localhost:7002")

	// Execute
	cfg, err := Load()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.Orderer.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, cfg.Queries.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, cfg.Notifications.ConnectionTimeout)
}

func TestLoad_WithConfigFile(t *testing.T) {
	t.Parallel()

	// Setup - Create temporary config file
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert
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
	// Setup - Set environment variables
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "EnvMSP")
	t.Setenv("FXCONFIG_MSP_CONFIGPATH", "/env/msp")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "env-orderer:7050")
	t.Setenv("FXCONFIG_ORDERER_CONNECTIONTIMEOUT", "15s")
	t.Setenv("FXCONFIG_QUERIES_ADDRESS", "env-query:7001")

	// Execute
	cfg, err := Load()

	// Assert
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

	// Execute
	cfg, err := Load(
		WithOverride("msp.localMspID", "OverrideMSP"),
		WithOverride("orderer.address", "override-orderer:7050"),
		WithOverride("orderer.connectionTimeout", 20*time.Second),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "OverrideMSP", cfg.MSP.LocalMspID)
	assert.Equal(t, "override-orderer:7050", cfg.Orderer.Address)
	assert.Equal(t, 20*time.Second, cfg.Orderer.ConnectionTimeout)
}

func TestLoad_ConfigHierarchy(t *testing.T) {
	// Setup - Create config file
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

	// Setup - Set environment variable (should override file)
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "EnvMSP")

	// Execute - With override (should override both file and env)
	cfg, err := Load(
		WithConfigFile(configPath),
		WithOverride("msp.localMspID", "OverrideMSP"),
	)

	// Assert - Override should win
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "OverrideMSP", cfg.MSP.LocalMspID, "Override should take precedence")
	assert.Equal(t, "/file/msp", cfg.MSP.ConfigPath, "File value should be used")
	assert.Equal(t, "file-orderer:7050", cfg.Orderer.Address, "File value should be used")
}

func TestLoad_ProjectConfig(t *testing.T) { //nolint:paralleltest
	// Note: This test is not parallel because it changes working directory

	// Setup - Create temporary directory structure
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

	// Change to temp directory
	t.Chdir(tmpDir)

	// Execute
	cfg, err := Load()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "ProjectMSP", cfg.MSP.LocalMspID)
	assert.Equal(t, "/project/msp", cfg.MSP.ConfigPath)
}

func TestLoad_InvalidYAML(t *testing.T) {
	t.Parallel()

	// Setup - Create invalid YAML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	invalidContent := `
msp:
  localMspID: TestMSP
  invalid yaml content [[[
`
	err := os.WriteFile(configPath, []byte(invalidContent), 0o600)
	require.NoError(t, err)

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should return error for invalid YAML
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_EmptyConfig(t *testing.T) {
	// Setup - Create empty config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)

	// Setup - Set required fields via environment variables
	t.Setenv("FXCONFIG_MSP_LOCALMSPID", "TestMSP")
	t.Setenv("FXCONFIG_MSP_CONFIGPATH", "/tmp/msp")
	t.Setenv("FXCONFIG_ORDERER_ADDRESS", "localhost:7050")
	t.Setenv("FXCONFIG_QUERIES_ADDRESS", "localhost:7001")
	t.Setenv("FXCONFIG_NOTIFICATIONS_ADDRESS", "localhost:7002")

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should succeed with defaults
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.Orderer.ConnectionTimeout)
}

func TestLoad_PartialConfig(t *testing.T) {
	t.Parallel()

	// Setup - Create partial config file with all required fields
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert - Should use defaults for missing values
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

	// Setup
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

	// Execute
	cfg, err := Load(WithConfigFile(configPath))

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "/path/to/client.key", cfg.Orderer.TLS.ClientKeyPath)
	assert.Equal(t, "/path/to/client.crt", cfg.Orderer.TLS.ClientCertPath)
	assert.Len(t, cfg.Orderer.TLS.RootCertPaths, 1)
	assert.Equal(t, "/path/to/ca.crt", cfg.Orderer.TLS.RootCertPaths[0])
	assert.Equal(t, "orderer.override.com", cfg.Orderer.TLS.ServerNameOverride)
}
