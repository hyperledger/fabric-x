/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCommand(t *testing.T) {
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
			args:          []string{"namespace", "update"},
			deployFunc:    fakeDeploySuccess,
			expectError:   true,
			errorContains: "accepts 1 arg(s), received 0",
		},
		{
			name: "successful update with all flags",
			args: []string{
				"namespace", "update",
				"1",
				"--version", "2",
				"--policy", "threshold:/tmp/some/path/pk.pem",
			},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
		{
			name: "deploy function returns error",
			args: []string{
				"namespace", "update",
				"2",
				"--version", "2",
				"--policy", "threshold:/tmp/some/path/pk.pem",
			},
			deployFunc:    fakeDeployError,
			expectError:   true,
			errorContains: "deployment failed",
		},
		{
			name:        "help flag displays usage",
			args:        []string{"namespace", "update", "--help"},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup with mock config
			rootCmd := setupUpdateCommandWithConfig(t, tt.deployFunc)

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
	t.Parallel()

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
			rootCmd := setupUpdateCommandWithConfig(t, fakeDeploySuccess)
			rootCmd.SetArgs([]string{
				"namespace", "update",
				"1",
				"--version", tt.version,
				"--policy", "threshold:/tmp/pk.pem",
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

	// Verify command-specific required flags
	versionFlag := cmd.Flag("version")
	require.NotNil(t, versionFlag, "version flag should exist")

	policyFlag := cmd.Flag("policy")
	require.NotNil(t, policyFlag, "policy flag should exist")
}

// Test helpers

func setupUpdateCommandWithConfig(t *testing.T, deploy deployF) *cobra.Command {
	t.Helper()

	// Create proper command hierarchy: root -> namespace -> update
	updateCmd := newUpdateCommand(deploy)
	return setupNamespaceCommandWithConfig(t, updateCmd)
}
