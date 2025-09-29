/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"testing"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/namespace"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	t.Run("with no args", func(t *testing.T) {
		t.Parallel()
		rootCmd := setupCreate(t)
		rootCmd.SetArgs([]string{"create"})
		require.Error(t, rootCmd.Execute())
	})

	t.Run("with correct args", func(t *testing.T) {
		t.Parallel()
		rootCmd := setupCreate(t)
		rootCmd.SetArgs([]string{
			"create",
			"someNamespaceID",
			"--channel", "mychannel",
			"--orderer", "localhost:1234",
			"--mspConfigPath", "/tmp/msp/",
			"--mspID", "Org1MSP",
			"--pk", "/tmp/some/path/pk.pem",
		})
		require.NoError(t, rootCmd.Execute())
	})
}

func setupCreate(t *testing.T) *cobra.Command {
	t.Helper()

	rootCmd := &cobra.Command{Use: "namespace"}
	rootCmd.AddCommand(newCreateCommand(fakeDeploy))
	return rootCmd
}

func fakeDeploy(_ namespace.NsConfig, _ namespace.OrdererConfig, _ namespace.MSPConfig) error {
	// don't do anything
	return nil
}
