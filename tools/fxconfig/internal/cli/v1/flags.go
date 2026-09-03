// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"

	"github.com/spf13/cobra"
)

// outputFlag represents an output file path flag.
type outputFlag string

func (f *outputFlag) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "output", "",
		"Output file path (if not specified, writes to stdout)")
}

// policyFlag represents an endorsement policy flag.
type policyFlag string

func (f *policyFlag) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "policy", "",
		"Endorsement policy (e.g., \"OR('Org1MSP.member')\" or \"AND('Org1MSP.member', 'Org2MSP.member')\")")
	_ = cmd.MarkFlagRequired("policy")
}

// versionFlag represents a namespace version number flag.
type versionFlag int

func (f *versionFlag) bind(cmd *cobra.Command) {
	cmd.Flags().IntVar((*int)(f), "version", 0,
		"Current namespace version (required for updates to prevent conflicts)")
	_ = cmd.MarkFlagRequired("version")
}

// namespaceDeployFlags groups flags for namespace deployment operations.
type namespaceDeployFlags struct {
	endorse bool
	submit  bool
	wait    bool
}

func (f *namespaceDeployFlags) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.endorse, "endorse", false,
		"Endorse transaction with local MSP before saving/submitting")
	cmd.Flags().BoolVar(&f.submit, "submit", false,
		"Submit transaction to ordering service (requires --endorse)")
	cmd.Flags().BoolVar(&f.wait, "wait", false,
		"Wait for transaction finalization (implies --submit)")
}

// Validate checks that the flag combinations are valid.
func (f *namespaceDeployFlags) Validate() error {
	if f.submit && !f.endorse {
		return fmt.Errorf("the --submit flag requires --endorse")
	}
	if f.wait && !f.submit {
		return fmt.Errorf("the --wait flag requires --submit")
	}
	return nil
}

// waitFlag represents a flag to wait for transaction finalization.
type waitFlag bool

func (f *waitFlag) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar((*bool)(f), "wait", false,
		"Wait for transaction to be finalized and return status code")
}
