/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"encoding/base64"
	"fmt"
	"strings"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

// namespaceListEntry is the display representation of a namespace query result.
type namespaceListEntry struct {
	Name         string `json:"name"         yaml:"name"`
	Version      int    `json:"version"      yaml:"version"`
	Policy       []byte `json:"policy"       yaml:"policy"`
	PolicyString string `json:"policyString" yaml:"policyString"`
}

// namespaceListOutput is a slice of namespaceListEntry with table rendering support.
type namespaceListOutput []namespaceListEntry

// String renders the namespace list as a human-readable table.
func (r namespaceListOutput) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Installed namespaces (%d total):\n", len(r)))
	for i, p := range r {
		sb.WriteString(fmt.Sprintf("%d) %v: version %d policy: %x\n", i, p.Name, p.Version, p.Policy))
	}
	return sb.String()
}

func toListOutput(results []app.NamespaceQueryResult) namespaceListOutput {
	out := make(namespaceListOutput, len(results))
	for i, r := range results {
		out[i] = namespaceListEntry{
			Name:         r.NsID,
			Version:      r.Version,
			Policy:       r.Policy,
			PolicyString: parsePolicy(r.Policy),
		}
	}
	return out
}

// parsePolicy decodes namespace policy bytes into a human-readable string.
// For threshold policies, returns the base64-encoded public key.
// For MSP policies, returns the proto text representation of the signature policy envelope.
// Falls back to a hex-encoded string if decoding fails.
func parsePolicy(b []byte) string {
	var ns applicationpb.NamespacePolicy
	if err := proto.Unmarshal(b, &ns); err != nil {
		return fmt.Sprintf("%x", b)
	}

	switch r := ns.Rule.(type) {
	case *applicationpb.NamespacePolicy_ThresholdRule:
		return base64.StdEncoding.EncodeToString(r.ThresholdRule.GetPublicKey())
	case *applicationpb.NamespacePolicy_MspRule:
		var spe cb.SignaturePolicyEnvelope
		if err := proto.Unmarshal(r.MspRule, &spe); err != nil {
			return fmt.Sprintf("%x", r.MspRule)
		}
		return spe.String()
	default:
		return fmt.Sprintf("%x", b)
	}
}

// newNsListCommand creates a command for listing installed namespaces.
// It connects to the query service and displays namespace names, versions, and policies.
func newNsListCommand(ctx *CLIContext) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed Namespaces",
		Long: `Query and display all installed namespaces with their configurations.

For each namespace, displays:
  • Name (namespace identifier)
  • Version (current version number)
  • Policy (endorsement policy)

Use this command to:
  • Verify namespace deployment
  • Check current version before updates
  • Audit endorsement policies

Output Formats:
  • table  Human-readable text (default)
  • json   Machine-readable JSON array
  • yaml   Machine-readable YAML list

Examples:
  # List all namespaces
  fxconfig namespace list

  # List with custom config
  fxconfig namespace list --config /path/to/config.yaml

  # Machine-readable JSON output for scripting
  fxconfig namespace list --format json

  # YAML output
  fxconfig namespace list --format yaml`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := cliio.Format(format)
			if f != cliio.FormatTable && f != cliio.FormatJSON && f != cliio.FormatYAML {
				return fmt.Errorf("invalid --format %q: must be one of json, yaml, table", format)
			}

			result, err := ctx.App.ListNamespaces(cmd.Context())
			if err != nil {
				return err
			}

			output := toListOutput(result)
			printer := cliio.NewCLIPrinter(cmd.OutOrStdout(), cmd.ErrOrStderr(), f)
			printer.Print(output)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format (json|yaml|table)")

	return cmd
}
