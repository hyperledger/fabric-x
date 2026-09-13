/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package snapshot provides utilities for reading and validating Hyperledger Fabric
// peer snapshot directories.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const signableMetadataFile = "_snapshot_signable_metadata.json"

// Manifest holds the contents of _snapshot_signable_metadata.json.
// Fabric writes this file at snapshot time so the receiver can verify
// the snapshot files haven't been tampered with.
type Manifest struct {
	ChannelName       string            `json:"channel_name"`
	LastBlockNumber   uint64            `json:"last_block_number"`
	LastBlockHash     string            `json:"last_block_hash"`
	PreviousBlockHash string            `json:"previous_block_hash"`
	StateDBType       string            `json:"state_db_type"`
	FileHashes        map[string]string `json:"file_hashes"`
}

// ReadManifest parses the signable metadata file from a snapshot directory.
func ReadManifest(snapshotDir string) (*Manifest, error) {
	path := filepath.Join(snapshotDir, signableMetadataFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", signableMetadataFile, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", signableMetadataFile, err)
	}

	if m.ChannelName == "" {
		return nil, fmt.Errorf("%s: channel_name is empty", signableMetadataFile)
	}

	return &m, nil
}

// VerifyChecksums recomputes SHA-256 hashes for every file listed in the manifest
// and returns an error if any hash doesn't match.
func VerifyChecksums(snapshotDir string, m *Manifest) error {
	for filename, expectedHash := range m.FileHashes {
		path := filepath.Join(snapshotDir, filename)
		got, err := sha256File(path)
		if err != nil {
			return fmt.Errorf("cannot hash %s: %w", filename, err)
		}
		// Fabric stores hashes as hex strings, sometimes prefixed with "sha256:"
		want := expectedHash
		if len(want) > 7 && want[:7] == "sha256:" {
			want = want[7:]
		}
		if got != want {
			return fmt.Errorf("checksum mismatch for %s: got %s want %s", filename, got, want)
		}
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
