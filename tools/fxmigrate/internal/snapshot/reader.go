/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"bufio"
	"encoding/binary"
	"errors"
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

// rawRecord is one decoded entry from public_state.data before filtering and
// version conversion. namespace is empty when the on-disk record uses the
// "same namespace as previous" marker (ns_len == 0).
type rawRecord struct {
	namespace string
	key       string
	value     []byte
	blockNum  uint64
	txNum     uint64
}

// DiscoverNamespaces scans the public state data file and returns the unique
// set of namespaces present, excluding any that would be filtered out.
func DiscoverNamespaces(snapshotDir string) ([]string, error) {
	entries, err := readStateFile(snapshotDir)
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
	return readStateFile(snapshotDir)
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
func readStateFile(snapshotDir string) ([]StateEntry, error) {
	path := filepath.Join(snapshotDir, publicStateDataFile)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", publicStateDataFile, err)
	}
	defer func() { _ = f.Close() }()

	r := bufio.NewReaderSize(f, 1<<20)

	var (
		entries   []StateEntry
		currentNS string
	)

	for {
		rec, rerr := readRawRecord(r)
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return nil, rerr
		}

		if rec.namespace != "" {
			currentNS = rec.namespace
		}

		if ShouldExcludeNamespace(currentNS) || ShouldExcludeKey(rec.key) || strings.HasPrefix(rec.key, "~") {
			continue
		}

		entries = append(entries, StateEntry{
			Namespace: currentNS,
			Key:       rec.key,
			Value:     rec.value,
			Version:   (rec.blockNum << 32) | (rec.txNum & 0xFFFFFFFF),
		})
	}

	return entries, nil
}

// readRawRecord decodes one entry from the public_state.data stream. It
// returns io.EOF unwrapped when the stream is at a clean record boundary so
// the caller can terminate the loop. Truncated records mid-entry surface as
// wrapped errors.
func readRawRecord(r *bufio.Reader) (*rawRecord, error) {
	nsLen, err := readUint32(r)
	if err != nil {
		return nil, err
	}

	var ns string
	if nsLen > 0 {
		nsBytes, berr := readBytes(r, nsLen)
		if berr != nil {
			return nil, fmt.Errorf("reading namespace: %w", berr)
		}
		ns = string(nsBytes)
	}

	keyBytes, err := readLenPrefixed(r)
	if err != nil {
		return nil, fmt.Errorf("reading key: %w", err)
	}

	valBytes, err := readLenPrefixed(r)
	if err != nil {
		return nil, fmt.Errorf("reading value: %w", err)
	}

	blockNum, err := readUint64(r)
	if err != nil {
		return nil, fmt.Errorf("reading block num: %w", err)
	}
	txNum, err := readUint64(r)
	if err != nil {
		return nil, fmt.Errorf("reading tx num: %w", err)
	}

	return &rawRecord{
		namespace: ns,
		key:       string(keyBytes),
		value:     valBytes,
		blockNum:  blockNum,
		txNum:     txNum,
	}, nil
}

func readLenPrefixed(r io.Reader) ([]byte, error) {
	n, err := readUint32(r)
	if err != nil {
		return nil, err
	}
	return readBytes(r, n)
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
