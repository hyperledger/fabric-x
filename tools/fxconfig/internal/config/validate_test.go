/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
