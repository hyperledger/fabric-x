/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"errors"
	"fmt"
)

// Namespace represents key-value pairs in a namespace.
type Namespace map[string]string

// Snapshot represents a simplified snapshot structure.
type Snapshot struct {
	Channel    string
	Namespaces map[string]Namespace
}

// ValidateSnapshot performs minimal structural validation.
func ValidateSnapshot(s Snapshot) error {
	if s.Channel == "" {
		return errors.New("channel is empty")
	}

	if len(s.Namespaces) == 0 {
		return errors.New("no namespaces found")
	}

	for ns, kv := range s.Namespaces {
		if ns == "" {
			return errors.New("empty namespace name")
		}

		if len(kv) == 0 {
			return fmt.Errorf("namespace %s has no keys", ns)
		}

		for k := range kv {
			if k == "" {
				return fmt.Errorf("empty key in namespace %s", ns)
			}
		}
	}

	return nil
}
