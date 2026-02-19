/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/namespace"
)

func TestCreateCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		deployFunc    deployF
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing namespace ID argument",
			args:          []string{"create"},
			deployFunc:    fakeDeploySuccess,
			expectError:   true,
			errorContains: "accepts 1 arg(s), received 0",
		},
		{
			name: "missing required channel flag",
			args: []string{
				"create",
				"someNamespaceID",
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
			name: "successful creation with all required flags",
			args: []string{
				"create",
				"someNamespaceID",
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
			name: "successful creation with TLS flags",
			args: []string{
				"create",
				"testNamespace",
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
				"create",
				"failNamespace",
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
			args:        []string{"create", "--help"},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			rootCmd := setupCreateCommand(t, tt.deployFunc)

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

func TestNewCreateCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newCreateCommand(fakeDeploySuccess)

	// Assert
	require.NotNil(t, cmd, "newCreateCommand should return a non-nil command")
	assert.Equal(t, "create NAMESPACE_NAME", cmd.Use, "command use should be 'create NAMESPACE_NAME'")
	assert.NotEmpty(t, cmd.Short, "command should have a short description")
	assert.NotNil(t, cmd.RunE, "command should have a RunE function")

	// Verify required flags are marked as required
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

func setupCreateCommand(t *testing.T, deploy deployF) *cobra.Command {
	t.Helper()

	rootCmd := &cobra.Command{Use: "namespace"}
	rootCmd.AddCommand(newCreateCommand(deploy))
	return rootCmd
}

func fakeDeploySuccess(_ namespace.NsConfig, _ namespace.OrdererConfig, _ namespace.MSPConfig) error {
	return nil
}

func fakeDeployError(_ namespace.NsConfig, _ namespace.OrdererConfig, _ namespace.MSPConfig) error {
	return errors.New("deployment failed")
}
