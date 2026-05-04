/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxmigrate/internal/genesis"
	"github.com/hyperledger/fabric-x/tools/fxmigrate/internal/snapshot"
)

// exportOptions groups the CLI flags consumed by runExport so the function
// signature stays under the revive argument-limit.
type exportOptions struct {
	snapshotDir string
	channel     string
	namespace   string
	outputFile  string
}

// NewExportCommand returns the export subcommand.
// It reads a Fabric peer snapshot directory and writes a Fabric-X genesis-data file.
func NewExportCommand() *cobra.Command {
	var opts exportOptions

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a Fabric peer snapshot into a Fabric-X genesis-data file",
		Long: `export reads a Hyperledger Fabric peer snapshot directory and produces
a verifiable genesis-data file compatible with the Fabric-X committer's
--init-from-snapshot bootstrap mode.

The tool applies the following transformations:
  - Strips all private-data-collection (PDC) hash keys ($$h$$ pattern)
  - Strips PDC private state keys ($$p$$ pattern)
  - Strips implicit org collection keys
  - Strips system chaincode namespaces (lscc, _lifecycle, etc.)
  - Converts Fabric block-version pairs to Fabric-X scalar BIGINT versions
  - Maps the source channel to the specified Fabric-X namespace

Example:
  fxmigrate export \
    --snapshot ./peer/snapshots/completed/mychannel/100 \
    --channel  mychannel \
    --namespace token \
    --output   genesis.bin`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runExport(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.snapshotDir, "snapshot", "", "Path to the Fabric peer snapshot directory (required)")
	cmd.Flags().StringVar(&opts.channel, "channel", "", "Source Fabric channel name (required)")
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "Target Fabric-X namespace name (required)")
	cmd.Flags().StringVar(&opts.outputFile, "output", "genesis.bin", "Output genesis-data file path")

	_ = cmd.MarkFlagRequired("snapshot")
	_ = cmd.MarkFlagRequired("channel")
	_ = cmd.MarkFlagRequired("namespace")

	return cmd
}

func runExport(cmd *cobra.Command, opts exportOptions) error {
	out := cmd.OutOrStdout()

	// Phase 1: Verify snapshot integrity
	_, _ = fmt.Fprintln(out, "Phase 1/5: Verifying snapshot integrity...")
	meta, err := snapshot.ReadManifest(opts.snapshotDir)
	if err != nil {
		return fmt.Errorf("snapshot integrity check failed: %w", err)
	}
	if meta.ChannelName != opts.channel {
		return fmt.Errorf("snapshot channel %q does not match --channel %q", meta.ChannelName, opts.channel)
	}
	if cerr := snapshot.VerifyChecksums(opts.snapshotDir, meta); cerr != nil {
		return fmt.Errorf("snapshot checksum mismatch: %w", cerr)
	}
	_, _ = fmt.Fprintf(out, "  channel=%s block_height=%d db_type=%s\n",
		meta.ChannelName, meta.LastBlockNumber, meta.StateDBType)

	// Phase 2: Discover namespaces in public state
	_, _ = fmt.Fprintln(out, "Phase 2/5: Discovering namespaces...")
	namespaces, err := snapshot.DiscoverNamespaces(opts.snapshotDir)
	if err != nil {
		return fmt.Errorf("namespace discovery failed: %w", err)
	}
	_, _ = fmt.Fprintf(out, "  found %d namespace(s): %v\n", len(namespaces), namespaces)

	// Phase 3: Export state
	_, _ = fmt.Fprintf(out, "Phase 3/5: Exporting state (channel=%s → namespace=%s)...\n", opts.channel, opts.namespace)
	entries, err := snapshot.ExportState(opts.snapshotDir)
	if err != nil {
		return fmt.Errorf("state export failed: %w", err)
	}
	_, _ = fmt.Fprintf(out, "  exported %d key-value pairs\n", len(entries))

	// Phase 4: Write genesis-data file
	_, _ = fmt.Fprintf(out, "Phase 4/5: Writing genesis-data file → %s...\n", opts.outputFile)
	if werr := genesis.Write(opts.outputFile, opts.namespace, meta, entries); werr != nil {
		return fmt.Errorf("failed to write genesis-data file: %w", werr)
	}

	// Phase 5: Generate and print manifest
	_, _ = fmt.Fprintln(out, "Phase 5/5: Computing output checksum...")
	checksum, err := genesis.Checksum(opts.outputFile)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	_, _ = fmt.Fprintln(out, "\nDone.")
	_, _ = fmt.Fprintf(out, "  output:   %s\n", opts.outputFile)
	_, _ = fmt.Fprintf(out, "  sha256:   %s\n", checksum)
	_, _ = fmt.Fprintf(out, "  entries:  %d\n", len(entries))
	_, _ = fmt.Fprintf(out, "  channel:  %s → namespace: %s\n", opts.channel, opts.namespace)

	return nil
}
