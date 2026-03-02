/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewListCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newNsListCommand(&CLIContext{App: &testApp{}})

	// Assert
	require.NotNil(t, cmd, "newNsListCommand should return a non-nil command")
	require.Equal(t, "list", cmd.Use, "command use should be 'list'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.RunE, "command should have a RunE function")
}
