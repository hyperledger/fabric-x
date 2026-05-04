/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// rawEntry is an unfiltered record written to the fixture file.
type rawEntry struct {
	namespace string // empty string means "reuse previous namespace" (ns_len=0)
	key       string
	value     []byte
	blockNum  uint64
	txNum     uint64
}

// buildStateFile writes a Fabric-format public_state.data fixture to dir.
//
// Fabric format per entry:
//
//	[ns_len uint32][ns bytes]      (ns_len=0 → same namespace as previous)
//	[key_len uint32][key bytes]
//	[val_len uint32][val bytes]
//	[block_num uint64][tx_num uint64]
func buildStateFile(t *testing.T, dir string, entries []rawEntry) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, publicStateDataFile))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	write32 := func(v uint32) {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], v)
		_, err := f.Write(buf[:])
		require.NoError(t, err)
	}
	write64 := func(v uint64) {
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], v)
		_, err := f.Write(buf[:])
		require.NoError(t, err)
	}
	writeBytes := func(b []byte) {
		require.LessOrEqual(t, len(b), math.MaxUint32)
		//nolint:gosec // G115: length is bounds-checked above against math.MaxUint32.
		write32(uint32(len(b)))
		if len(b) > 0 {
			_, err := f.Write(b)
			require.NoError(t, err)
		}
	}

	for _, e := range entries {
		writeBytes([]byte(e.namespace)) // ns_len=0 when namespace==""
		writeBytes([]byte(e.key))
		writeBytes(e.value)
		write64(e.blockNum)
		write64(e.txNum)
	}
}

func TestExportState_BasicEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildStateFile(t, dir, []rawEntry{
		{namespace: "mycc", key: "asset1", value: []byte("val1"), blockNum: 5, txNum: 2},
		{namespace: "mycc", key: "asset2", value: []byte("val2"), blockNum: 10, txNum: 0},
	})

	entries, err := ExportState(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	require.Equal(t, "mycc", entries[0].Namespace)
	require.Equal(t, "asset1", entries[0].Key)
	require.Equal(t, []byte("val1"), entries[0].Value)
	// version = (5 << 32) | 2
	require.Equal(t, uint64(5)<<32|uint64(2), entries[0].Version)

	require.Equal(t, "asset2", entries[1].Key)
	require.Equal(t, uint64(10)<<32|uint64(0), entries[1].Version)
}

func TestExportState_SameNamespaceMarker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Second entry uses ns_len=0 (empty namespace string → reuse previous)
	buildStateFile(t, dir, []rawEntry{
		{namespace: "mycc", key: "k1", value: []byte("v1"), blockNum: 1, txNum: 0},
		{namespace: "", key: "k2", value: []byte("v2"), blockNum: 2, txNum: 0},
	})

	entries, err := ExportState(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "mycc", entries[0].Namespace)
	require.Equal(t, "mycc", entries[1].Namespace)
}

func TestExportState_ExcludesSystemNamespaces(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildStateFile(t, dir, []rawEntry{
		{namespace: "lscc", key: "someCC", value: []byte("x"), blockNum: 1, txNum: 0},
		{namespace: "_lifecycle", key: "def", value: []byte("y"), blockNum: 1, txNum: 1},
		{namespace: "mycc", key: "asset1", value: []byte("z"), blockNum: 2, txNum: 0},
	})

	entries, err := ExportState(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "mycc", entries[0].Namespace)
}

func TestExportState_ExcludesTildeKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildStateFile(t, dir, []rawEntry{
		{namespace: "mycc", key: "~couchdbMeta", value: []byte("meta"), blockNum: 1, txNum: 0},
		{namespace: "mycc", key: "realKey", value: []byte("val"), blockNum: 1, txNum: 1},
	})

	entries, err := ExportState(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "realKey", entries[0].Key)
}

func TestExportState_ExcludesPDCNamespaces(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildStateFile(t, dir, []rawEntry{
		{namespace: "$$h$$mycc", key: "hashKey", value: []byte("h"), blockNum: 1, txNum: 0},
		{namespace: "$$p$$mycc$$coll", key: "privKey", value: []byte("p"), blockNum: 1, txNum: 1},
		{namespace: "mycc", key: "pub", value: []byte("v"), blockNum: 2, txNum: 0},
	})

	entries, err := ExportState(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "mycc", entries[0].Namespace)
}

func TestExportState_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := ExportState(t.TempDir())
	require.Error(t, err)
	require.ErrorContains(t, err, publicStateDataFile)
}

func TestDiscoverNamespaces(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildStateFile(t, dir, []rawEntry{
		{namespace: "mycc", key: "k1", value: []byte("v"), blockNum: 1, txNum: 0},
		{namespace: "token", key: "k2", value: []byte("v"), blockNum: 1, txNum: 1},
		{namespace: "mycc", key: "k3", value: []byte("v"), blockNum: 2, txNum: 0},
		{namespace: "lscc", key: "k4", value: []byte("v"), blockNum: 2, txNum: 1}, // excluded
	})

	namespaces, err := DiscoverNamespaces(dir)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"mycc", "token"}, namespaces)
}

func TestDiscoverNamespaces_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := DiscoverNamespaces(t.TempDir())
	require.Error(t, err)
}
