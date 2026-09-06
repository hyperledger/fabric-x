/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectError    bool
	}{
		{
			name:           "version command with no args",
			args:           []string{versionCmd},
			expectedOutput: []string{appName, "Version:", "Go Version:", "OS/Arch:"},
			expectError:    false,
		},
		{
			name:           "version command with help flag",
			args:           []string{versionCmd, "--help"},
			expectedOutput: []string{"Usage:", appName + " " + versionCmd},
			expectError:    false,
		},
		{
			name:           "version command with invalid flag",
			args:           []string{versionCmd, "--invalid"},
			expectedOutput: []string{"unknown flag: --invalid"},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootCmd := &cobra.Command{Use: appName}
			rootCmd.AddCommand(NewVersionCommand())

			var outBuf, errBuf bytes.Buffer
			rootCmd.SetOut(&outBuf)
			rootCmd.SetErr(&errBuf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.expectError {
				require.Error(t, err)
				output := errBuf.String()
				for _, expected := range tt.expectedOutput {
					require.Contains(t, output, expected, "error output should contain expected text")
				}
			} else {
				require.NoError(t, err)
				output := outBuf.String()
				for _, expected := range tt.expectedOutput {
					require.Contains(t, output, expected, "output should contain expected text")
				}
			}
		})
	}
}

func TestVersionCommand_OutputFormat(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: appName}
	rootCmd.AddCommand(NewVersionCommand())

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetArgs([]string{versionCmd})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := outBuf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	require.GreaterOrEqual(t, len(lines), 5, "version output should have at least 5 lines")
	require.Equal(t, appName, lines[0], "first line should be 'fxconfig'")

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		require.Contains(t, line, ":", "line %d should contain a colon separator", i)
	}
}

func TestNewVersionCommand(t *testing.T) {
	t.Parallel()

	cmd := NewVersionCommand()

	require.NotNil(t, cmd, "NewVersionCommand should return a non-nil command")
	require.Equal(t, versionCmd, cmd.Use, "command use should be 'version'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.Run, "command should have a Run function")
}