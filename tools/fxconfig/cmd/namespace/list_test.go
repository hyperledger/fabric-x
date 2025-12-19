/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Run("with no args", func(t *testing.T) {
		t.Parallel()
		rootCmd := setupList(t)
		rootCmd.SetArgs([]string{"list"})
		require.Error(t, rootCmd.Execute())
	})

	t.Run("with endpoint arg", func(t *testing.T) {
		t.Parallel()
		rootCmd := setupList(t)
		rootCmd.SetArgs([]string{"list", "--endpoint=localhost:1234"})
		require.NoError(t, rootCmd.Execute())
	})
}

func setupList(t *testing.T) *cobra.Command {
	t.Helper()

	rootCmd := &cobra.Command{Use: "namespace"}
	rootCmd.AddCommand(newListCommand(fakeList))
	return rootCmd
}

var fakeList = func(_ io.Writer, _, _ string) error {
	// don't do anything
	return nil
}
