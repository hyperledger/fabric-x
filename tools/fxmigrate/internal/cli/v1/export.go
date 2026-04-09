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

// NewExportCommand returns the export subcommand.
// It reads a Fabric peer snapshot directory and writes a Fabric-X genesis-data file.
func NewExportCommand() *cobra.Command {
	var (
		snapshotDir string
		outputFile  string
		channel     string
		namespace   string
	)

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
			return runExport(cmd, snapshotDir, channel, namespace, outputFile)
		},
	}

	cmd.Flags().StringVar(&snapshotDir, "snapshot", "", "Path to the Fabric peer snapshot directory (required)")
	cmd.Flags().StringVar(&channel, "channel", "", "Source Fabric channel name (required)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Target Fabric-X namespace name (required)")
	cmd.Flags().StringVar(&outputFile, "output", "genesis.bin", "Output genesis-data file path")

	_ = cmd.MarkFlagRequired("snapshot")
	_ = cmd.MarkFlagRequired("channel")
	_ = cmd.MarkFlagRequired("namespace")

	return cmd
}

func runExport(cmd *cobra.Command, snapshotDir, channel, namespace, outputFile string) error {
	// Phase 1: Verify snapshot integrity
	fmt.Fprintf(cmd.OutOrStdout(), "Phase 1/5: Verifying snapshot integrity...\n")
	meta, err := snapshot.ReadManifest(snapshotDir)
	if err != nil {
		return fmt.Errorf("snapshot integrity check failed: %w", err)
	}
	if meta.ChannelName != channel {
		return fmt.Errorf("snapshot channel %q does not match --channel %q", meta.ChannelName, channel)
	}
	if err := snapshot.VerifyChecksums(snapshotDir, meta); err != nil {
		return fmt.Errorf("snapshot checksum mismatch: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  channel=%s block_height=%d db_type=%s\n",
		meta.ChannelName, meta.LastBlockNumber, meta.StateDBType)

	// Phase 2: Discover namespaces in public state
	fmt.Fprintf(cmd.OutOrStdout(), "Phase 2/5: Discovering namespaces...\n")
	namespaces, err := snapshot.DiscoverNamespaces(snapshotDir)
	if err != nil {
		return fmt.Errorf("namespace discovery failed: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  found %d namespace(s): %v\n", len(namespaces), namespaces)

	// Phase 3: Export state
	fmt.Fprintf(cmd.OutOrStdout(), "Phase 3/5: Exporting state (channel=%s → namespace=%s)...\n", channel, namespace)
	entries, err := snapshot.ExportState(snapshotDir)
	if err != nil {
		return fmt.Errorf("state export failed: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  exported %d key-value pairs\n", len(entries))

	// Phase 4: Write genesis-data file
	fmt.Fprintf(cmd.OutOrStdout(), "Phase 4/5: Writing genesis-data file → %s...\n", outputFile)
	if err := genesis.Write(outputFile, namespace, meta, entries); err != nil {
		return fmt.Errorf("failed to write genesis-data file: %w", err)
	}

	// Phase 5: Generate and print manifest
	fmt.Fprintf(cmd.OutOrStdout(), "Phase 5/5: Computing output checksum...\n")
	checksum, err := genesis.Checksum(outputFile)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nDone.\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  output:   %s\n", outputFile)
	fmt.Fprintf(cmd.OutOrStdout(), "  sha256:   %s\n", checksum)
	fmt.Fprintf(cmd.OutOrStdout(), "  entries:  %d\n", len(entries))
	fmt.Fprintf(cmd.OutOrStdout(), "  channel:  %s → namespace: %s\n", channel, namespace)

	return nil
}
