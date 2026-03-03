/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUpdateCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newNsUpdateCommand(&CLIContext{App: &testApp{}})

	// Assert
	require.NotNil(t, cmd, "newNsUpdateCommand should return a non-nil command")
	require.Equal(t, "update [name]", cmd.Use, "command use should be 'update [name]'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.RunE, "command should have a RunE function")

	// Verify command-specific required flags
	version := cmd.Flag("version")
	require.NotNil(t, version, "version flag should exist")

	policy := cmd.Flag("policy")
	require.NotNil(t, policy, "policy flag should exist")
}
