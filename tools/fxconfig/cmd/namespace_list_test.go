/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
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
			name:        "successful list",
			args:        []string{"namespace", "list"},
			listFunc:    fakeListSuccess,
			expectError: false,
		},
		{
			name:          "list function returns error",
			args:          []string{"namespace", "list"},
			listFunc:      fakeListError,
			expectError:   true,
			errorContains: "failed to list namespaces",
		},
		{
			name:        "help flag displays usage",
			args:        []string{"namespace", "list", "--help"},
			listFunc:    fakeListSuccess,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup with mock config
			rootCmd := setupListCommandWithConfig(t, tt.listFunc)

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
	captureListFunc := func(w io.Writer, _ config.QueriesConfig) error {
		capturedWriter = w
		return nil
	}

	rootCmd := setupListCommandWithConfig(t, captureListFunc)
	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetArgs([]string{"namespace", "list"})

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
}

// Test helpers

func setupListCommandWithConfig(t *testing.T, list listFunc) *cobra.Command {
	t.Helper()

	listCmd := newListCommand(list)
	return setupNamespaceCommandWithConfig(t, listCmd)
}

func fakeListSuccess(_ io.Writer, _ config.QueriesConfig) error {
	return nil
}

func fakeListError(_ io.Writer, _ config.QueriesConfig) error {
	return errors.New("failed to list namespaces")
}
