/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

func TestPolicyDSLChecker_Check(t *testing.T) {
	t.Parallel()

	checker := validation.PolicyDSLChecker{}

	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"valid OR", "OR('Org1MSP.member')", false},
		{"valid AND", "AND('Org1MSP.member', 'Org2MSP.member')", false},
		{"invalid expression", "NOT_A_POLICY", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checker.Check(tt.expr)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOSFileChecker_Exists(t *testing.T) {
	t.Parallel()

	checker := validation.OSFileChecker{}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "key.pem")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0o600))

	// Files whose names legitimately contain ".." used to be rejected by
	// strings.Contains(clean, ".."). These cases guard the fix that scopes
	// the traversal check to ".." as a path component, not a substring.
	dottyFile := filepath.Join(tmpDir, "myproject..backup")
	require.NoError(t, os.WriteFile(dottyFile, []byte("data"), 0o600))
	configDottyFile := filepath.Join(tmpDir, "config..yaml")
	require.NoError(t, os.WriteFile(configDottyFile, []byte("data"), 0o600))
	leadingDotsFile := filepath.Join(tmpDir, "..hidden")
	require.NoError(t, os.WriteFile(leadingDotsFile, []byte("data"), 0o600))

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid file", tmpFile, false},
		{"empty path", "", true},
		{"non-existent", filepath.Join(tmpDir, "missing.pem"), true},
		{"directory given", tmpDir, true},
		{"path traversal", "../../etc/passwd", true},
		{"single dotdot", "..", true},
		{"filename containing dotdot is accepted", dottyFile, false},
		{"config name with dotdot is accepted", configDottyFile, false},
		{"filename with leading dotdot is accepted", leadingDotsFile, false},
		{"internal dotdot collapses to valid path", filepath.Join(tmpDir, "sub", "..", "key.pem"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checker.Exists(tt.path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOSDirectoryChecker_Exists(t *testing.T) {
	t.Parallel()

	checker := validation.OSDirectoryChecker{}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0o600))

	// Directories whose names legitimately contain ".." used to be rejected.
	// These cases guard the fix that scopes the traversal check to ".." as
	// a path component, not a substring.
	dottyDir := filepath.Join(tmpDir, "release..2026-04")
	require.NoError(t, os.MkdirAll(dottyDir, 0o755))
	leadingDotsDir := filepath.Join(tmpDir, "..staging")
	require.NoError(t, os.MkdirAll(leadingDotsDir, 0o755))

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid directory", tmpDir, false},
		{"empty path", "", true},
		{"non-existent", filepath.Join(tmpDir, "missing"), true},
		{"file given", tmpFile, true},
		{"path traversal", "../../etc", true},
		{"single dotdot", "..", true},
		{"directory name containing dotdot is accepted", dottyDir, false},
		{"directory name with leading dotdot is accepted", leadingDotsDir, false},
		{"internal dotdot collapses to valid directory", filepath.Join(tmpDir, "sub", ".."), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checker.Exists(tt.path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
