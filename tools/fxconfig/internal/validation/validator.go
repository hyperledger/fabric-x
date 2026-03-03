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
func NewValidationContext() Context {
	return Context{
		PolicyChecker:    PolicyDSLChecker{},
		FileChecker:      OSFileChecker{},
		DirectoryChecker: OSDirectoryChecker{},
	}
}

// PolicyDSLChecker implements app.PolicyChecker using the Fabric policy DSL parser.
type PolicyDSLChecker struct{}

// Check validates a policy DSL expression.
func (PolicyDSLChecker) Check(e string) error {
	if _, err := policydsl.FromString(e); err != nil {
		return fmt.Errorf("invalid policy expression: %w", err)
	}
	return nil
}

// OSFileChecker implements app.FileChecker using os.Stat.
type OSFileChecker struct{}

// Exists checks if the path exists and is a file.
func (OSFileChecker) Exists(path string) error {
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

// OSDirectoryChecker implements app.DirectoryChecker using os.Stat.
type OSDirectoryChecker struct{}

// Exists checks if the path exists and is a directory.
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
