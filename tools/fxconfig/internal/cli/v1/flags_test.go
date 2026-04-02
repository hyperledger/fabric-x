/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestOutputFlag_Bind(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	var f outputFlag
	f.bind(cmd)

	flag := cmd.Flags().Lookup("output")
	require.NotNil(t, flag)
	require.Empty(t, flag.DefValue)
}

func TestPolicyFlag_Bind(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	var f policyFlag
	f.bind(cmd)

	flag := cmd.Flags().Lookup("policy")
	require.NotNil(t, flag)
	require.Empty(t, flag.DefValue)

	// policy is required
	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	require.Contains(t, err.Error(), "policy")
}

func TestVersionFlag_Bind(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	var f versionFlag
	f.bind(cmd)

	flag := cmd.Flags().Lookup("version")
	require.NotNil(t, flag)
	require.Equal(t, "0", flag.DefValue)

	// version is required
	err := cmd.ValidateRequiredFlags()
	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestNamespaceDeployFlags_Bind(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	var f namespaceDeployFlags
	f.bind(cmd)

	for _, name := range []string{"endorse", "submit", "wait"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			flag := cmd.Flags().Lookup(name)
			require.NotNil(t, flag)
			require.Equal(t, "false", flag.DefValue)
		})
	}
}

func TestWaitFlag_Bind(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	var f waitFlag
	f.bind(cmd)

	flag := cmd.Flags().Lookup("wait")
	require.NotNil(t, flag)
	require.Equal(t, "false", flag.DefValue)
}
