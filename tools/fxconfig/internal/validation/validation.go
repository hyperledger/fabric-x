/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package validation provides interfaces and implementations for validating
// policy expressions, file paths, and directory paths used in fxconfig operations.
package validation

// Context holds validation interfaces for domain input verification.
type Context struct {
	// PolicyChecker validates Fabric policy DSL expressions.
	PolicyChecker PolicyChecker
	// FileChecker validates file existence and accessibility.
	FileChecker FileChecker
	// DirectoryChecker validates directory existence and accessibility.
	DirectoryChecker DirectoryChecker
}

// PolicyChecker validates Fabric policy DSL expressions.
type PolicyChecker interface {
	// Check validates a policy expression string.
	Check(e string) error
}

// FileChecker validates file paths and existence.
type FileChecker interface {
	// Exists verifies that the path exists and is a regular file.
	Exists(path string) error
}

// DirectoryChecker validates directory paths and existence.
type DirectoryChecker interface {
	// Exists verifies that the path exists and is a directory.
	Exists(path string) error
}
