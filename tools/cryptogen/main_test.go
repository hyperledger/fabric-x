/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExecutionErrorRoutedToStderr exercises the top-level error handler in
// main() by re-invoking the test binary as a subprocess with an unparseable
// config file. It guards two POSIX-conformance properties:
//  1. Errors are written to stderr (not stdout) so callers piping stdout do
//     not silently capture error text alongside any future structured output.
//  2. The exit code is exactly 1 (not -1, which wraps to 255 on POSIX and is
//     conventionally reserved for shell signals / out-of-range statuses).
func TestExecutionErrorRoutedToStderr(t *testing.T) {
	if os.Getenv("CRYPTOGEN_RUN_MAIN") == "1" {
		argv := strings.Fields(os.Getenv("CRYPTOGEN_ARGS"))
		os.Args = append([]string{"cryptogen"}, argv...)
		main()
		return
	}

	tmpDir := t.TempDir()
	badCfg := filepath.Join(tmpDir, "bad.yaml")
	require.NoError(t, os.WriteFile(badCfg, []byte("{unterminated: "), 0o600))

	cmd := exec.Command(os.Args[0], "-test.run=^TestExecutionErrorRoutedToStderr$")
	cmd.Env = append(os.Environ(),
		"CRYPTOGEN_RUN_MAIN=1",
		"CRYPTOGEN_ARGS=generate --config "+badCfg,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "expected non-zero exit")
	require.Equal(t, 1, exitErr.ExitCode(), "exit code must be 1, not -1/255")
	require.NotContains(t, stdout.String(), "error executing command",
		"error text must not appear on stdout")
	require.Contains(t, stderr.String(), "error executing command",
		"error text must appear on stderr")
}

func TestGetVersionInfo(t *testing.T) {
	t.Parallel()
	expected := fmt.Sprintf(
		"%s:\n Version: %s\n Commit SHA: %s\n Go version: %s\n OS/Arch: %s",
		programName,
		version,
		"development build",
		runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	)
	require.Equal(t, expected, getVersionInfo())

	testSHA := "abcdefg"
	commitSHA = testSHA
	expected = fmt.Sprintf(
		"%s:\n Version: %s\n Commit SHA: %s\n Go version: %s\n OS/Arch: %s",
		programName,
		version,
		testSHA,
		runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	)
	require.Equal(t, expected, getVersionInfo())
}
