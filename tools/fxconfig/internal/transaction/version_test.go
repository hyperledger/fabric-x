/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"testing"

	"github.com/stretchr/testify/require"
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

			err := ValidateVersion(tt.version)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}
