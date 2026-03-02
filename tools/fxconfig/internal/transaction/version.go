/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import "errors"

// ValidateVersion ensures the version is valid.
// Version -1 indicates a create operation, >= 0 indicates an update.
func ValidateVersion(v int) error {
	if v < -1 {
		return errors.New("invalid version: must be -1 (create) or >= 0 (update)")
	}
	return nil
}
