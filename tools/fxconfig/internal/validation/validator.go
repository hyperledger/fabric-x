/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric-x-common/common/policydsl"
)

// NewValidationContext creates a validation context with OS-based validators.
// Returns a Context configured with PolicyDSLChecker, OSFileChecker, and OSDirectoryChecker.
func NewValidationContext() Context {
	return Context{
		PolicyChecker:    PolicyDSLChecker{},
		FileChecker:      OSFileChecker{},
		DirectoryChecker: OSDirectoryChecker{},
	}
}

// PolicyDSLChecker validates Fabric policy DSL expressions using the policydsl parser.
type PolicyDSLChecker struct{}

// Check validates a policy DSL expression string.
// Returns an error if the expression cannot be parsed.
func (PolicyDSLChecker) Check(e string) error {
	if _, err := policydsl.FromString(e); err != nil {
		return fmt.Errorf("invalid policy expression: %w", err)
	}
	return nil
}

// OSFileChecker validates file paths using os.Stat.
// Prevents path traversal attacks and ensures paths reference regular files.
type OSFileChecker struct{}

// Exists verifies that the path exists and is a regular file.
// Returns an error if the path doesn't exist, is a directory, or contains path traversal.
func (OSFileChecker) Exists(path string) error {
	if path == "" {
		return errors.New("path must not be empty")
	}

	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return errors.New("path traversal not allowed")
	}

	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but got directory: %s", path)
	}
	return nil
}

// OSDirectoryChecker validates directory paths using os.Stat.
// Prevents path traversal attacks and ensures paths reference directories.
type OSDirectoryChecker struct{}

// Exists verifies that the path exists and is a directory.
// Returns an error if the path is empty, doesn't exist, is a file, or contains path traversal.
func (OSDirectoryChecker) Exists(path string) error {
	if path == "" {
		return errors.New("path must not be empty")
	}

	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return errors.New("path traversal not allowed")
	}

	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", path)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}
