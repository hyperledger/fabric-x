/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTxRootCommand(t *testing.T) {
	t.Parallel()

	ctx := &CLIContext{App: &testApp{}}
	cmd := NewTxRootCommand(ctx)

	require.NotNil(t, cmd)
	require.Equal(t, "tx", cmd.Use)
	require.NotEmpty(t, cmd.Short)

	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}
	require.True(t, subCmds["endorse"])
	require.True(t, subCmds["merge"])
	require.True(t, subCmds["submit"])
}
