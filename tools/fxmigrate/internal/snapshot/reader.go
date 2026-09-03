/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	publicStateDataFile = "public_state.data"
)

// StateEntry holds a single key-value pair from the public state, along
// with its namespace and the converted Fabric-X version.
type StateEntry struct {
	Namespace string
	Key       string
	Value     []byte
	// Version is converted from Fabric's (BlockNum, TxNum) pair to a
	// Fabric-X scalar: (blockNum << 32) | txNum
	Version uint64
}

// DiscoverNamespaces scans the public state data file and returns the unique
// set of namespaces present, excluding any that would be filtered out.
func DiscoverNamespaces(snapshotDir string) ([]string, error) {
	entries, err := readStateFile(snapshotDir, true)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, e := range entries {
		seen[e.Namespace] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for ns := range seen {
		out = append(out, ns)
	}
	return out, nil
}

// ExportState reads the public state data file and returns all entries that
// pass the namespace and key filters.
func ExportState(snapshotDir string) ([]StateEntry, error) {
	return readStateFile(snapshotDir, false)
}

// readStateFile reads the Fabric snapshot public_state.data file.
//
// Fabric writes this file using a simple length-delimited binary format:
//
//	[namespace_len uint32][namespace bytes]
//	[key_len uint32][key bytes]
//	[value_len uint32][value bytes]
//	[block_num uint64][tx_num uint64]   <- version
//
// A zero-length namespace signals a namespace boundary marker (same namespace
// as the previous entry continues). We handle both forms.
//
// If discoverOnly is true the function returns entries with empty Value/Version
// for speed — we only need namespace names.
func readStateFile(snapshotDir string, discoverOnly bool) ([]StateEntry, error) {
	path := filepath.Join(snapshotDir, publicStateDataFile)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", publicStateDataFile, err)
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 1<<20) // 1 MiB read buffer

	var (
		entries      []StateEntry
		currentNS    string
	)

	for {
		// Read namespace length
		nsLen, err := readUint32(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading namespace length: %w", err)
		}

		if nsLen > 0 {
			nsBytes, err := readBytes(r, nsLen)
			if err != nil {
				return nil, fmt.Errorf("reading namespace: %w", err)
			}
			currentNS = string(nsBytes)
		}
		// nsLen == 0 means same namespace as previous entry

		// Read key
		keyLen, err := readUint32(r)
		if err != nil {
			return nil, fmt.Errorf("reading key length: %w", err)
		}
		keyBytes, err := readBytes(r, keyLen)
		if err != nil {
			return nil, fmt.Errorf("reading key: %w", err)
		}
		key := string(keyBytes)

		// Read value
		valLen, err := readUint32(r)
		if err != nil {
			return nil, fmt.Errorf("reading value length: %w", err)
		}
		valBytes, err := readBytes(r, valLen)
		if err != nil {
			return nil, fmt.Errorf("reading value: %w", err)
		}

		// Read version: (blockNum uint64, txNum uint64)
		blockNum, err := readUint64(r)
		if err != nil {
			return nil, fmt.Errorf("reading block num: %w", err)
		}
		txNum, err := readUint64(r)
		if err != nil {
			return nil, fmt.Errorf("reading tx num: %w", err)
		}

		// Apply filters
		if ShouldExcludeNamespace(currentNS) {
			continue
		}
		if ShouldExcludeKey(key) {
			continue
		}
		// Strip CouchDB internal metadata keys
		if strings.HasPrefix(key, "~") {
			continue
		}

		entry := StateEntry{
			Namespace: currentNS,
			Key:       key,
		}
		if !discoverOnly {
			entry.Value = valBytes
			// Convert (blockNum, txNum) → scalar Fabric-X version
			entry.Version = (blockNum << 32) | (txNum & 0xFFFFFFFF)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func readUint32(r io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf[:]), nil
}

func readUint64(r io.Reader) (uint64, error) {
	var buf [8]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf[:]), nil
}

func readBytes(r io.Reader, n uint32) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}
