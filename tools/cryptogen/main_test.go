/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestGetConfig_InvalidFile(t *testing.T) {
	t.Parallel()

	// Create temp invalid config file
	f, err := os.CreateTemp(t.TempDir(), "invalid-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Write invalid content
	_, _ = f.WriteString("invalid: [unclosed")
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("failed to close temp file: %v", closeErr)
	}

	// Save original value and restore after test
	originalGenConfigFile := genConfigFile
	defer func() { genConfigFile = originalGenConfigFile }()

	// Simulate CLI flag
	fHandle, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer func() {
		_ = fHandle.Close()
	}()
	genConfigFile = &fHandle

	_, err = getConfig()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse config")
}
