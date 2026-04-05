/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		expectedOutput []string
	}{
		{
			name:           "version command output",
			expectedOutput: []string{"fxconfig", "Version:", "Go version:", "OS/Arch:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var outBuf bytes.Buffer
			ctx := &CLIContext{
				Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
			}

			cmd := NewVersionCommand(ctx)
			cmd.Run(cmd, nil)

			output := outBuf.String()
			for _, expected := range tt.expectedOutput {
				require.Contains(t, output, expected)
			}
		})
	}
}

func TestVersionCommand_OutputFormat(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewVersionCommand(ctx)
	cmd.Run(cmd, nil)

	output := outBuf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	require.GreaterOrEqual(t, len(lines), 5, "version output should have at least 5 lines")
	require.Equal(t, "fxconfig", strings.TrimSpace(lines[0]), "first line should be 'fxconfig'")

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

	ctx := &CLIContext{
		Printer: cliio.NewCLIPrinter(&bytes.Buffer{}, &bytes.Buffer{}, cliio.FormatTable),
	}
	cmd := NewVersionCommand(ctx)

	require.NotNil(t, cmd)
	require.Equal(t, "version", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.Run)
}
