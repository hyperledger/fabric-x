/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import "errors"

// ValidateVersion checks if version is valid: -1 for create, >= 0 for update.
func ValidateVersion(v int) error {
	if v < -1 {
		return errors.New("invalid version: must be -1 (create) or >= 0 (update)")
	}
	return nil
}
