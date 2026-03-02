package v1

import "github.com/spf13/cobra"

type InputFlag string

func (f *InputFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "input", "", "Input file (optional, defaults to stdin)")
}

type OutputFlag string

func (f *OutputFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "output", "", "Output file (optional, defaults to stdout)")
}

type PolicyFlag string

func (f *PolicyFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar((*string)(f), "policy", "", "The endorsement policy")
	_ = cmd.MarkFlagRequired("policy")
}

type VersionFlag int

func (f *VersionFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().IntVar((*int)(f), "version", 0, "The current namespace version")
	_ = cmd.MarkFlagRequired("version")
}

type NamespaceDeployFlags struct {
	endorse bool
	submit  bool
	wait    bool
}

func (f *NamespaceDeployFlags) Bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.endorse, "endorse", false, "Endorse transaction")
	cmd.Flags().BoolVar(&f.submit, "submit", false, "Submit transaction")
	cmd.Flags().BoolVar(&f.wait, "wait", false, "Wait for transaction finalizing")
}

type WaitFlag bool

func (f *WaitFlag) Bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar((*bool)(f), "wait", false, "Wait for transaction finalizing")
}
