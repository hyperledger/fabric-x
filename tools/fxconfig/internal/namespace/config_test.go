/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

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

func TestValidateConfig(t *testing.T) {
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
			err := validateConfig(tt.nsCfg)
			if tt.expectError {
				require.Error(t, err, "expected error for %s", tt.name)
			} else {
				require.NoError(t, err, "expected no error for %s", tt.name)
			}
		})
	}
}
