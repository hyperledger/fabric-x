package v1

import "github.com/spf13/cobra"

// InputFlag represents an input file path flag.
type InputFlag string

func (f *InputFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "input", "",
		"Input file path (if not specified, reads from stdin)")
}

// OutputFlag represents an output file path flag.
type OutputFlag string

func (f *OutputFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "output", "",
		"Output file path (if not specified, writes to stdout)")
}

// PolicyFlag represents an endorsement policy flag.
type PolicyFlag string

func (f *PolicyFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "policy", "",
		"Endorsement policy (e.g., \"OR('Org1MSP.member')\" or \"AND('Org1MSP.member', 'Org2MSP.member')\")")
	_ = cmd.MarkFlagRequired("policy")
}

// VersionFlag represents a namespace version number flag.
type VersionFlag int

func (f *VersionFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().IntVar((*int)(f), "version", 0,
		"Current namespace version (required for updates to prevent conflicts)")
	_ = cmd.MarkFlagRequired("version")
}

// NamespaceDeployFlags groups flags for namespace deployment operations.
type NamespaceDeployFlags struct {
	endorse bool
	submit  bool
	wait    bool
}

func (f *NamespaceDeployFlags) Bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.endorse, "endorse", false,
		"Endorse transaction with local MSP before saving/submitting")
	cmd.Flags().BoolVar(&f.submit, "submit", false,
		"Submit transaction to ordering service (requires --endorse)")
	cmd.Flags().BoolVar(&f.wait, "wait", false,
		"Wait for transaction finalization (implies --submit)")
}

// WaitFlag represents a flag to wait for transaction finalization.
type WaitFlag bool

func (f *WaitFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar((*bool)(f), "wait", false,
		"Wait for transaction to be finalized and return status code")
}
