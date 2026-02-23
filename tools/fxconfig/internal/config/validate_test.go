/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	validChannelName                        = "mychannel"
	validNamespaceID                        = "1"
	validVersion                            = 0
	validThresholdPolicyVerificationKeyPath = "/path/to/key"
)

// TestValidateVersion tests the validateVersion function.
func TestValidateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		version     int
		expectError bool
		description string
	}{
		{
			name:        "version -1 (create)",
			version:     -1,
			expectError: false,
			description: "Version -1 should be valid for create operations",
		},
		{
			name:        "version -2 (invalid)",
			version:     -2,
			expectError: true,
			description: "Version -2 should be invalid",
		},
		{
			name:        "version -999 (invalid)",
			version:     -999,
			expectError: true,
			description: "Large negative version should be invalid",
		},
		{
			name:        "version 0 (update)",
			version:     0,
			expectError: false,
			description: "Version 0 should be valid for updates",
		},
		{
			name:        "version 1 (update)",
			version:     1,
			expectError: false,
			description: "Version 1 should be valid for updates",
		},
		{
			name:        "version 999999 (large positive)",
			version:     999999,
			expectError: false,
			description: "Large positive version should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nsCfg := NsConfig{
				Version: tt.version,
			}

			err := validateVersion(nsCfg)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// TestMustHavePolicy tests the mustHavePolicy function.
func TestMustHavePolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		policyPath  string
		expectError bool
		description string
	}{
		{
			name:        "empty policy path",
			policyPath:  "",
			expectError: true,
			description: "Empty policy path should fail",
		},
		{
			name:        "valid policy path",
			policyPath:  "/path/to/policy.pem",
			expectError: false,
			description: "Valid policy path should pass",
		},
		{
			name:        "whitespace-only policy path",
			policyPath:  "   ",
			expectError: true,
			description: "Whitespace-only policy path should fail",
		},
		{
			name:        "policy path with spaces",
			policyPath:  "/path/with spaces/policy.pem",
			expectError: false,
			description: "Policy path with spaces should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nsCfg := NsConfig{
				ThresholdPolicyVerificationKeyPath: tt.policyPath,
			}

			err := mustHavePolicy(nsCfg)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// TestErrorIfEmpty tests the errorIfEmpty helper function.
func TestErrorIfEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "non-empty string",
			input:       "test",
			expectError: false,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: true,
		},
		{
			name:        "single space",
			input:       " ",
			expectError: true,
		},
		{
			name:        "tab character",
			input:       "\t",
			expectError: true,
		},
		{
			name:        "newline character",
			input:       "\n",
			expectError: true,
		},
		{
			name:        "mixed whitespace",
			input:       " \t\n ",
			expectError: true,
		},
		{
			name:        "string with leading/trailing spaces",
			input:       "  test  ",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := errorIfEmpty(tt.input, "test error message")
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateNsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nsCfg       NsConfig
		expectError bool
	}{
		{
			name: "valid config",
			nsCfg: NsConfig{
				Channel:                            validChannelName,
				NamespaceID:                        validNamespaceID,
				Version:                            validVersion,
				ThresholdPolicyVerificationKeyPath: validThresholdPolicyVerificationKeyPath,
			},
			expectError: false,
		},
		{
			name: "empty namespace ID",
			nsCfg: NsConfig{
				Channel:                            validChannelName,
				NamespaceID:                        "",
				Version:                            validVersion,
				ThresholdPolicyVerificationKeyPath: validThresholdPolicyVerificationKeyPath,
			},
			expectError: true,
		},
		{
			name: "invalid namespace ID",
			nsCfg: NsConfig{
				Channel:                            validChannelName,
				NamespaceID:                        "invalid namespace",
				Version:                            validVersion,
				ThresholdPolicyVerificationKeyPath: validThresholdPolicyVerificationKeyPath,
			},
			expectError: true,
		},
		{
			name: "invalid version",
			nsCfg: NsConfig{
				Channel:                            validChannelName,
				NamespaceID:                        validNamespaceID,
				Version:                            -2,
				ThresholdPolicyVerificationKeyPath: validThresholdPolicyVerificationKeyPath,
			},
			expectError: true,
		},
		{
			name: "empty threshold policy verification key path",
			nsCfg: NsConfig{
				Channel:                            validChannelName,
				NamespaceID:                        validNamespaceID,
				Version:                            validVersion,
				ThresholdPolicyVerificationKeyPath: "",
			},
			expectError: true,
		},
		{
			name: "invalid threshold policy verification key path",
			nsCfg: NsConfig{
				Channel:                            validChannelName,
				NamespaceID:                        validNamespaceID,
				Version:                            validVersion,
				ThresholdPolicyVerificationKeyPath: " ",
			},
			expectError: true,
		},
		{
			name: "invalid channel name",
			nsCfg: NsConfig{
				Channel:                            "",
				NamespaceID:                        validNamespaceID,
				Version:                            validVersion,
				ThresholdPolicyVerificationKeyPath: validThresholdPolicyVerificationKeyPath,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateNsConfig(tt.nsCfg)
			if tt.expectError {
				require.Error(t, err, "expected error for %s", tt.name)
			} else {
				require.NoError(t, err, "expected no error for %s", tt.name)
			}
		})
	}
}
