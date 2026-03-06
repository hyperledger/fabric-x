// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cliio

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestIOFlags_Bind(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{
		Use: "test",
	}

	flags := &IOFlags{}
	flags.Bind(cmd)

	// Verify flags are registered
	inputFlag := cmd.Flags().Lookup("input")
	require.NotNil(t, inputFlag)
	require.Empty(t, inputFlag.DefValue)

	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	require.Empty(t, outputFlag.DefValue)
}

func TestResolveInput_FromFile(t *testing.T) {
	t.Parallel()

	t.Run("success reading from file", func(t *testing.T) {
		t.Parallel()

		// Create temporary file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "input.txt")
		testData := []byte("test input data")
		err := os.WriteFile(tmpFile, testData, 0o600)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		data, err := ResolveInput(cmd, tmpFile)
		require.NoError(t, err)
		require.Equal(t, testData, data)
	})

	t.Run("error with non-existent file", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{}
		data, err := ResolveInput(cmd, "/non/existent/file.txt")
		require.Error(t, err)
		require.Nil(t, data)
	})

	t.Run("error with path traversal", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{}
		data, err := ResolveInput(cmd, "../../../etc/passwd")
		require.Error(t, err)
		require.Nil(t, data)
		require.Contains(t, err.Error(), "path traversal not allowed")
	})

	t.Run("error with file exceeding size limit", func(t *testing.T) {
		t.Parallel()

		// Create large temporary file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "large.txt")
		largeData := make([]byte, defaultMaxInputSize+1)
		err := os.WriteFile(tmpFile, largeData, 0o600)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		data, err := ResolveInput(cmd, tmpFile)
		require.Error(t, err)
		require.Nil(t, data)
		require.Contains(t, err.Error(), "exceeds maximum allowed size")
	})
}

func TestResolveInput_NoInput(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	data, err := ResolveInput(cmd, "")
	require.Error(t, err)
	require.Nil(t, data)
	require.Contains(t, err.Error(), "no input provided")
}

func TestWriteOutput_ToFile(t *testing.T) {
	t.Parallel()

	t.Run("success writing to file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "output.txt")
		testData := []byte("test output data")

		cmd := &cobra.Command{}
		err := WriteOutput(cmd, tmpFile, testData)
		require.NoError(t, err)

		// Verify file contents
		written, err := os.ReadFile(tmpFile)
		require.NoError(t, err)
		require.Equal(t, testData, written)

		// Verify file permissions
		info, err := os.Stat(tmpFile)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("error with invalid path", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{}
		err := WriteOutput(cmd, "/invalid/path/output.txt", []byte("data"))
		require.Error(t, err)
	})
}

func TestWriteOutput_ToStdout(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	testData := []byte("test output data")
	err := WriteOutput(cmd, "", testData)
	require.NoError(t, err)
	require.Equal(t, testData, buf.Bytes())
}

func TestReadWithLimit(t *testing.T) {
	t.Parallel()

	t.Run("success reading within limit", func(t *testing.T) {
		t.Parallel()

		input := strings.NewReader("test data")
		data, err := ReadWithLimit(input, 1024)
		require.NoError(t, err)
		require.Equal(t, []byte("test data"), data)
	})

	t.Run("error exceeding limit", func(t *testing.T) {
		t.Parallel()

		largeData := strings.Repeat("x", 1001)
		input := strings.NewReader(largeData)
		data, err := ReadWithLimit(input, 1000)
		require.Error(t, err)
		require.Nil(t, data)
		require.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("success reading exactly at limit", func(t *testing.T) {
		t.Parallel()

		exactData := strings.Repeat("x", 100)
		input := strings.NewReader(exactData)
		data, err := ReadWithLimit(input, 100)
		require.NoError(t, err)
		require.Len(t, data, 100)
	})

	t.Run("success reading empty input", func(t *testing.T) {
		t.Parallel()

		input := strings.NewReader("")
		data, err := ReadWithLimit(input, 1024)
		require.NoError(t, err)
		require.Empty(t, data)
	})
}
