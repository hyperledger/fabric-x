/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

// Context holds port interfaces used to validate domain inputs.
// Concrete implementations (OS-based checkers) are provided by the infrastructure
// layer (internal/config) and injected at the composition root.
type Context struct {
	PolicyChecker    PolicyChecker
	FileChecker      FileChecker
	DirectoryChecker DirectoryChecker
}

// PolicyChecker validates policy expressions.
type PolicyChecker interface {
	Check(e string) error
}

// FileChecker validates file paths.
type FileChecker interface {
	Exists(path string) error
}

// DirectoryChecker validates directory paths.
type DirectoryChecker interface {
	Exists(path string) error
}
