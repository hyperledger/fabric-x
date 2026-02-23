/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/cmd/common/comm"
)

func TestListCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		listFunc      listFunc
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing required endpoint flag",
			args:          []string{"list"},
			listFunc:      fakeListSuccess,
			expectError:   true,
			errorContains: "required flag(s)",
		},
		{
			name:        "successful list with endpoint",
			args:        []string{"list", "--endpoint=localhost:1234"},
			listFunc:    fakeListSuccess,
			expectError: false,
		},
		{
			name:        "successful list with endpoint and TLS",
			args:        []string{"list", "--endpoint=orderer.example.com:7050", "--cafile=/opt/tls/ca.crt"},
			listFunc:    fakeListSuccess,
			expectError: false,
		},
		{
			name: "successful list with all TLS options",
			args: []string{
				"list",
				"--endpoint=localhost:7050",
				"--cafile=/opt/tls/ca.crt",
				"--certfile=/opt/tls/client.crt",
				"--keyfile=/opt/tls/client.key",
			},
			listFunc:    fakeListSuccess,
			expectError: false,
		},
		{
			name:          "list function returns error",
			args:          []string{"list", "--endpoint=localhost:1234"},
			listFunc:      fakeListError,
			expectError:   true,
			errorContains: "failed to list namespaces",
		},
		{
			name:        "help flag displays usage",
			args:        []string{"list", "--help"},
			listFunc:    fakeListSuccess,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			rootCmd := setupListCommand(t, tt.listFunc)

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

func TestListCommand_OutputWriter(t *testing.T) {
	t.Parallel()

	// Setup
	var capturedWriter io.Writer
	captureListFunc := func(w io.Writer, _ string, _ comm.Config) error {
		capturedWriter = w
		return nil
	}

	rootCmd := setupListCommand(t, captureListFunc)
	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetArgs([]string{"list", "--endpoint=localhost:1234"})

	// Execute
	err := rootCmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, capturedWriter, "list function should receive a writer")
}

func TestNewListCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newListCommand(fakeListSuccess)

	// Assert
	require.NotNil(t, cmd, "newListCommand should return a non-nil command")
	assert.Equal(t, "list", cmd.Use, "command use should be 'list'")
	assert.NotEmpty(t, cmd.Short, "command should have a short description")
	assert.NotNil(t, cmd.RunE, "command should have a RunE function")

	// Verify required flags
	endpointFlag := cmd.Flag("endpoint")
	require.NotNil(t, endpointFlag, "endpoint flag should exist")

	// Verify optional TLS flags exist
	cafileFlag := cmd.Flag("cafile")
	require.NotNil(t, cafileFlag, "cafile flag should exist")

	certfileFlag := cmd.Flag("certfile")
	require.NotNil(t, certfileFlag, "certfile flag should exist")

	keyfileFlag := cmd.Flag("keyfile")
	require.NotNil(t, keyfileFlag, "keyfile flag should exist")
}

// Test helpers

func setupListCommand(t *testing.T, list listFunc) *cobra.Command {
	t.Helper()

	rootCmd := &cobra.Command{Use: "namespace"}
	rootCmd.AddCommand(newListCommand(list))
	return rootCmd
}

func fakeListSuccess(_ io.Writer, _ string, _ comm.Config) error {
	return nil
}

func fakeListError(_ io.Writer, _ string, _ comm.Config) error {
	return errors.New("failed to list namespaces")
}
