/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hyperledger/fabric-x-common/common/policydsl"
)

// ValidationContext holds validators for configuration validation.
type ValidationContext struct {
	PolicyChecker    PolicyChecker
	FileChecker      FileChecker
	DirectoryChecker DirectoryChecker
}

// NewValidationContext creates a ValidationContext with OS-based validators.
func NewValidationContext() ValidationContext {
	return ValidationContext{
		PolicyChecker:    PolicyDSLChecker{},
		FileChecker:      OSFileChecker{},
		DirectoryChecker: OSDirectoryCheckerChecker{},
	}
}

// PolicyChecker validates policy expressions.
type PolicyChecker interface {
	Check(e string) error
}

// PolicyDSLChecker validates Fabric policy DSL expressions.
type PolicyDSLChecker struct{}

// Check validates a policy DSL expression.
func (PolicyDSLChecker) Check(e string) error {
	if _, err := policydsl.FromString(e); err != nil {
		return fmt.Errorf("invalid policy expression: %w", err)
	}
	return nil
}

// FileChecker validates file paths.
type FileChecker interface {
	Exists(path string) error
}

// OSFileChecker validates file paths using os.Stat.
type OSFileChecker struct{}

// Exists checks if the path exists and is a file.
func (OSFileChecker) Exists(path string) error {
	info, err := os.Stat(path)
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

// DirectoryChecker validates directory paths.
type DirectoryChecker interface {
	Exists(path string) error
}

// OSDirectoryCheckerChecker validates directory paths using os.Stat.
type OSDirectoryCheckerChecker struct{}

// Exists checks if the path exists and is a directory.
func (OSDirectoryCheckerChecker) Exists(path string) error {
	info, err := os.Stat(path)
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

// validateVersion ensures the version is valid.
// Version -1 indicates a create operation, >= 0 indicates an update.
func validateVersion(v int) error {
	if v < -1 {
		return errors.New("invalid version: must be -1 (create) or >= 0 (update)")
	}
	return nil
}

// errorIfEmpty returns an error if the string is empty or whitespace-only.
func errorIfEmpty(s, errMsg string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New(errMsg)
	}
	return nil
}

// errorIfZeroDuration returns an error if the duration is zero.
func errorIfZeroDuration(d time.Duration, errMsg string) error {
	if d == time.Duration(0) {
		return errors.New(errMsg)
	}
	return nil
}

func validateFilePath(ctx ValidationContext, p string) error {
	if p == "" {
		return errors.New("path must not be empty")
	}

	if ctx.FileChecker == nil {
		return errors.New("file checker not available")
	}

	return ctx.FileChecker.Exists(p)
}

func validateDirectoryPath(ctx ValidationContext, p string) error {
	if p == "" {
		return errors.New("path must not be empty")
	}

	if ctx.DirectoryChecker == nil {
		return errors.New("directory checker not available")
	}

	return ctx.DirectoryChecker.Exists(p)
}
