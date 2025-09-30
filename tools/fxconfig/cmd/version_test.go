/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	t.Run("with no args", func(t *testing.T) {
		t.Parallel()
		rootCmd := &cobra.Command{Use: "fxconfig"}
		rootCmd.AddCommand(NewVersionCommand())
		rootCmd.SetArgs([]string{"version"})
		require.NoError(t, rootCmd.Execute())
	})
}
