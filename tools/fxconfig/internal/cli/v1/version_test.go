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
			args:           []string{"version"},
			expectedOutput: []string{"fxconfig", "Version:", "Go Version:", "OS/Arch:"},
			expectError:    false,
		},
		{
			name:           "version command with help flag",
			args:           []string{"version", "--help"},
			expectedOutput: []string{"Usage:", "fxconfig version"},
			expectError:    false,
		},
		{
			name:           "version command with invalid flag",
			args:           []string{"version", "--invalid"},
			expectedOutput: []string{"unknown flag: --invalid"},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			rootCmd := &cobra.Command{Use: "fxconfig"}
			rootCmd.AddCommand(NewVersionCommand())

			var outBuf, errBuf bytes.Buffer
			rootCmd.SetOut(&outBuf)
			rootCmd.SetErr(&errBuf)
			rootCmd.SetArgs(tt.args)

			// Execute
			err := rootCmd.Execute()

			// Assert
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

	// Setup
	rootCmd := &cobra.Command{Use: "fxconfig"}
	rootCmd.AddCommand(NewVersionCommand())

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetArgs([]string{"version"})

	// Execute
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Assert output format
	output := outBuf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	require.GreaterOrEqual(t, len(lines), 5, "version output should have at least 5 lines")
	require.Equal(t, "fxconfig", lines[0], "first line should be 'fxconfig'")

	// Verify subsequent lines have the expected format (key: value)
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

	// Execute
	cmd := NewVersionCommand()

	// Assert
	require.NotNil(t, cmd, "NewVersionCommand should return a non-nil command")
	require.Equal(t, "version", cmd.Use, "command use should be 'version'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.Run, "command should have a Run function")
}
