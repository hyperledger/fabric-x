/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// TestValidatePolicy tests the policy validation.
func TestValidatePolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		policy      string
		expectError bool
		description string
	}{
		{
			name:        "empty policy path",
			policy:      "",
			expectError: true,
			description: "Empty policy path should fail",
		},
		{
			name:        "valid policy path",
			policy:      "threshold:/path/to/policy.pem",
			expectError: false,
			description: "Valid policy path should pass",
		},
		{
			name:        "whitespace-only policy path",
			policy:      "   ",
			expectError: true,
			description: "Whitespace-only policy path should fail",
		},
	}

	vctx := validation.Context{
		PolicyChecker: &FakePolicyChecker{},
		FileChecker:   &FakeFileChecker{},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var pc PolicyConfig
			pc.Set(tt.policy)

			err := pc.Validate(vctx)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

type FakePolicyChecker struct{}

func (FakePolicyChecker) Check(_ string) error {
	// always exists
	return nil
}

type FakeFileChecker struct{}

func (FakeFileChecker) Exists(_ string) error {
	// always exists
	return nil
}

type FakeDirectoryChecker struct{}

func (FakeDirectoryChecker) Exists(_ string) error {
	// always exists
	return nil
}

func fakeValidationContext() validation.Context {
	return validation.Context{
		PolicyChecker:    FakePolicyChecker{},
		FileChecker:      FakeFileChecker{},
		DirectoryChecker: FakeDirectoryChecker{},
	}
}
