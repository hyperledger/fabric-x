/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		deployFunc    deployF
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing namespace ID argument",
			args:          []string{"update"},
			deployFunc:    fakeDeploySuccess,
			expectError:   true,
			errorContains: "accepts 1 arg(s), received 0",
		},
		{
			name: "missing required version flag",
			args: []string{
				"update",
				"someNamespaceID",
				"--channel", "mychannel",
				"--orderer", "localhost:1234",
				"--mspConfigPath", "/tmp/msp/",
				"--mspID", "Org1MSP",
				"--policy-ecdsa-threshold", "/tmp/some/path/pk.pem",
			},
			deployFunc:    fakeDeploySuccess,
			expectError:   true,
			errorContains: "required flag(s)",
		},
		{
			name: "missing required channel flag",
			args: []string{
				"update",
				"someNamespaceID",
				"--version", "2",
				"--orderer", "localhost:1234",
				"--mspConfigPath", "/tmp/msp/",
				"--mspID", "Org1MSP",
				"--policy-ecdsa-threshold", "/tmp/some/path/pk.pem",
			},
			deployFunc:    fakeDeploySuccess,
			expectError:   true,
			errorContains: "required flag(s)",
		},
		{
			name: "successful update with all required flags",
			args: []string{
				"update",
				"someNamespaceID",
				"--version", "2",
				"--channel", "mychannel",
				"--orderer", "localhost:1234",
				"--mspConfigPath", "/tmp/msp/",
				"--mspID", "Org1MSP",
				"--policy-ecdsa-threshold", "/tmp/some/path/pk.pem",
			},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
		{
			name: "successful update with TLS flags",
			args: []string{
				"update",
				"testNamespace",
				"--version", "3",
				"--channel", "testchannel",
				"--orderer", "orderer.example.com:7050",
				"--mspConfigPath", "/opt/msp/",
				"--mspID", "TestMSP",
				"--policy-ecdsa-threshold", "/opt/keys/pk.pem",
				"--cafile", "/opt/tls/ca.crt",
			},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
		{
			name: "deploy function returns error",
			args: []string{
				"update",
				"failNamespace",
				"--version", "2",
				"--channel", "mychannel",
				"--orderer", "localhost:1234",
				"--mspConfigPath", "/tmp/msp/",
				"--mspID", "Org1MSP",
				"--policy-ecdsa-threshold", "/tmp/some/path/pk.pem",
			},
			deployFunc:    fakeDeployError,
			expectError:   true,
			errorContains: "deployment failed",
		},
		{
			name:        "help flag displays usage",
			args:        []string{"update", "--help"},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			rootCmd := setupUpdateCommand(t, tt.deployFunc)

			var outBuf, errBuf bytes.Buffer
			rootCmd.SetOut(&outBuf)
			rootCmd.SetErr(&errBuf)
			rootCmd.SetArgs(tt.args)

			// Execute
			err := rootCmd.Execute()

			// Assert
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					output := errBuf.String() + err.Error()
					assert.Contains(t, output, tt.errorContains, "error should contain expected text")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateCommand_VersionValidation(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "valid version 1",
			version:     "1",
			expectError: false,
		},
		{
			name:        "valid version 100",
			version:     "100",
			expectError: false,
		},
		{
			name:        "version zero",
			version:     "0",
			expectError: false,
		},
		{
			name:        "invalid version string",
			version:     "abc",
			expectError: true,
		},
		{
			name:        "empty version",
			version:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			rootCmd := setupUpdateCommand(t, fakeDeploySuccess)
			rootCmd.SetArgs([]string{
				"update",
				"testNS",
				"--version", tt.version,
				"--channel", "mychannel",
				"--orderer", "localhost:1234",
				"--mspConfigPath", "/tmp/msp/",
				"--mspID", "Org1MSP",
				"--policy-ecdsa-threshold", "/tmp/pk.pem",
			})

			// Execute
			err := rootCmd.Execute()

			// Assert
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewUpdateCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newUpdateCommand(fakeDeploySuccess)

	// Assert
	require.NotNil(t, cmd, "newUpdateCommand should return a non-nil command")
	assert.Equal(t, "update NAMESPACE_NAME", cmd.Use, "command use should be 'update NAMESPACE_NAME'")
	assert.NotEmpty(t, cmd.Short, "command should have a short description")
	assert.NotNil(t, cmd.RunE, "command should have a RunE function")

	// Verify required flags are marked as required
	versionFlag := cmd.Flag("version")
	require.NotNil(t, versionFlag, "version flag should exist")

	channelFlag := cmd.Flag("channel")
	require.NotNil(t, channelFlag, "channel flag should exist")

	ordererFlag := cmd.Flag("orderer")
	require.NotNil(t, ordererFlag, "orderer flag should exist")

	mspConfigFlag := cmd.Flag("mspConfigPath")
	require.NotNil(t, mspConfigFlag, "mspConfigPath flag should exist")

	mspIDFlag := cmd.Flag("mspID")
	require.NotNil(t, mspIDFlag, "mspID flag should exist")

	policyFlag := cmd.Flag("policy-ecdsa-threshold")
	require.NotNil(t, policyFlag, "policy-ecdsa-threshold flag should exist")
}

// Test helpers

func setupUpdateCommand(t *testing.T, deploy deployF) *cobra.Command {
	t.Helper()

	rootCmd := &cobra.Command{Use: "namespace"}
	rootCmd.AddCommand(newUpdateCommand(deploy))
	return rootCmd
}
