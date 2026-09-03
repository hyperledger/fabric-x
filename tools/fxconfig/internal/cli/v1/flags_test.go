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

func TestNamespaceDeployFlags_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		flags   namespaceDeployFlags
		wantErr string
	}{
		{
			name: "valid: none",
			flags: namespaceDeployFlags{
				endorse: false,
				submit:  false,
				wait:    false,
			},
		},
		{
			name: "valid: endorse only",
			flags: namespaceDeployFlags{
				endorse: true,
				submit:  false,
				wait:    false,
			},
		},
		{
			name: "valid: endorse and submit",
			flags: namespaceDeployFlags{
				endorse: true,
				submit:  true,
				wait:    false,
			},
		},
		{
			name: "valid: all flags",
			flags: namespaceDeployFlags{
				endorse: true,
				submit:  true,
				wait:    true,
			},
		},
		{
			name: "invalid: submit without endorse",
			flags: namespaceDeployFlags{
				endorse: false,
				submit:  true,
				wait:    false,
			},
			wantErr: "the --submit flag requires --endorse",
		},
		{
			name: "invalid: wait without submit",
			flags: namespaceDeployFlags{
				endorse: true,
				submit:  false,
				wait:    true,
			},
			wantErr: "the --wait flag requires --submit",
		},
		{
			name: "invalid: wait without submit and endorse",
			flags: namespaceDeployFlags{
				endorse: false,
				submit:  false,
				wait:    true,
			},
			wantErr: "the --wait flag requires --submit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.flags.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
