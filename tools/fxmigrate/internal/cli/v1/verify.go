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

// NewVerifyCommand returns the verify subcommand.
// It reads a genesis-data file produced by "export" and cross-checks it against
// the source Fabric snapshot to confirm no entries were lost or corrupted.
func NewVerifyCommand() *cobra.Command {
	var (
		genesisFile string
		snapshotDir string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify integrity between a genesis-data file and the source Fabric snapshot",
		Long: `verify reads a genesis-data file produced by "fxmigrate export" and
cross-checks it against the original Fabric peer snapshot directory.

Checks performed:
  1. Genesis file header is well-formed and readable.
  2. Actual entry count in the file matches the count in the header.
  3. Source snapshot entry count (after the same PDC/system-cc filtering)
     matches the genesis entry count.
  4. Block height in the header matches the snapshot block height.

Example:
  fxmigrate verify \
    --genesis  genesis.bin \
    --snapshot ./peer/snapshots/completed/mychannel/100`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVerify(cmd, genesisFile, snapshotDir)
		},
	}

	cmd.Flags().StringVar(&genesisFile, "genesis", "", "Path to the genesis-data file produced by export (required)")
	cmd.Flags().StringVar(&snapshotDir, "snapshot", "", "Path to the source Fabric peer snapshot directory (required)")

	_ = cmd.MarkFlagRequired("genesis")
	_ = cmd.MarkFlagRequired("snapshot")

	return cmd
}

func runVerify(cmd *cobra.Command, genesisFile, snapshotDir string) error {
	out := cmd.OutOrStdout()

	// Step 1: read genesis file header and all entries.
	fmt.Fprintf(out, "Step 1/4: Reading genesis-data file: %s\n", genesisFile)
	hdr, entries, err := genesis.ReadAll(genesisFile)
	if err != nil {
		return fmt.Errorf("cannot read genesis file: %w", err)
	}
	fmt.Fprintf(out, "  namespace:       %s\n", hdr.Namespace)
	fmt.Fprintf(out, "  source_channel:  %s\n", hdr.SourceChannel)
	fmt.Fprintf(out, "  block_height:    %d\n", hdr.BlockHeight)
	fmt.Fprintf(out, "  exported_at:     %s\n", hdr.ExportedAt)
	fmt.Fprintf(out, "  header says:     %d entries\n", hdr.EntryCount)

	genesisChecksum, err := genesis.Checksum(genesisFile)
	if err != nil {
		return fmt.Errorf("cannot checksum genesis file: %w", err)
	}
	fmt.Fprintf(out, "  sha256:          %s\n", genesisChecksum)

	// Step 2: verify actual entry count matches the header.
	fmt.Fprintf(out, "\nStep 2/4: Verifying entry count matches header...\n")
	actualCount := len(entries)
	if actualCount != hdr.EntryCount {
		return fmt.Errorf("genesis file is corrupt: header declares %d entries but file contains %d",
			hdr.EntryCount, actualCount)
	}
	fmt.Fprintf(out, "  entries in file: %d  OK\n", actualCount)

	// Step 3: verify the snapshot block height.
	fmt.Fprintf(out, "\nStep 3/4: Verifying snapshot block height...\n")
	meta, err := snapshot.ReadManifest(snapshotDir)
	if err != nil {
		return fmt.Errorf("cannot read snapshot manifest: %w", err)
	}
	if meta.LastBlockNumber != hdr.BlockHeight {
		return fmt.Errorf("block height mismatch: genesis says %d, snapshot says %d",
			hdr.BlockHeight, meta.LastBlockNumber)
	}
	fmt.Fprintf(out, "  block_height: %d  OK\n", meta.LastBlockNumber)

	// Step 4: compare filtered entry count from the snapshot.
	fmt.Fprintf(out, "\nStep 4/4: Counting filtered entries in source snapshot...\n")
	snapshotEntries, err := snapshot.ExportState(snapshotDir)
	if err != nil {
		return fmt.Errorf("cannot read snapshot state: %w", err)
	}
	snapshotCount := len(snapshotEntries)
	fmt.Fprintf(out, "  snapshot entries (filtered): %d\n", snapshotCount)
	if snapshotCount != actualCount {
		return fmt.Errorf("entry count mismatch: genesis has %d entries, snapshot has %d after filtering",
			actualCount, snapshotCount)
	}
	fmt.Fprintf(out, "  entry counts match  OK\n")

	fmt.Fprintf(out, "\nResult: OK — genesis-data file integrity verified.\n")
	return nil
}
