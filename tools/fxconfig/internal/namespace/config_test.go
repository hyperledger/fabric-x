/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nsCfg       NsConfig
		expectError bool
	}{
		{
			name:        "valid namespace ID",
			nsCfg:       NsConfig{NamespaceID: "1"},
			expectError: false,
		},
		{
			name:        "empty namespace ID",
			nsCfg:       NsConfig{NamespaceID: ""},
			expectError: true,
		},
		{
			name:        "invalid namespace ID",
			nsCfg:       NsConfig{NamespaceID: "invalid namespace"},
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
