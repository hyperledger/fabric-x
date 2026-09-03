/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package genesis

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxmigrate/internal/snapshot"
)

func makeManifest(channel string, blockHeight uint64) *snapshot.Manifest {
	return &snapshot.Manifest{
		ChannelName:     channel,
		LastBlockNumber: blockHeight,
		StateDBType:     "goleveldb",
	}
}

func TestReadHeader_RoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "genesis.bin")
	meta := makeManifest("mychannel", 42)
	entries := []snapshot.StateEntry{
		{Key: "k1", Value: []byte("v1"), Version: 1},
		{Key: "k2", Value: []byte("v2"), Version: 2},
	}

	require.NoError(t, Write(path, "token", meta, entries))

	hdr, err := ReadHeader(path)
	require.NoError(t, err)
	require.Equal(t, "1", hdr.Version)
	require.Equal(t, "token", hdr.Namespace)
	require.Equal(t, "mychannel", hdr.SourceChannel)
	require.Equal(t, uint64(42), hdr.BlockHeight)
	require.Equal(t, 2, hdr.EntryCount)
}

func TestReadAll_RoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "genesis.bin")
	meta := makeManifest("ch1", 100)
	want := []snapshot.StateEntry{
		{Key: "alpha", Value: []byte("aaa"), Version: 10},
		{Key: "beta", Value: []byte("bbb"), Version: 20},
		{Key: "gamma", Value: []byte("ccc"), Version: 30},
	}

	require.NoError(t, Write(path, "ns1", meta, want))

	hdr, got, err := ReadAll(path)
	require.NoError(t, err)
	require.Equal(t, 3, hdr.EntryCount)
	require.Len(t, got, 3)

	for i, e := range got {
		require.Equal(t, want[i].Key, e.Key)
		require.Equal(t, want[i].Value, e.Value)
		require.Equal(t, want[i].Version, e.Version)
	}
}

func TestReadAll_Empty(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "empty.bin")
	meta := makeManifest("ch", 0)

	require.NoError(t, Write(path, "ns", meta, nil))

	hdr, entries, err := ReadAll(path)
	require.NoError(t, err)
	require.Equal(t, 0, hdr.EntryCount)
	require.Empty(t, entries)
}

func TestReadHeader_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := ReadHeader(filepath.Join(t.TempDir(), "nonexistent.bin"))
	require.Error(t, err)
}

func TestReadAll_MissingFile(t *testing.T) {
	t.Parallel()

	_, _, err := ReadAll(filepath.Join(t.TempDir(), "nonexistent.bin"))
	require.Error(t, err)
}

func TestReadAll_BinaryValues(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "binary.bin")
	meta := makeManifest("ch", 5)
	want := []snapshot.StateEntry{
		{Key: "bin", Value: []byte{0x00, 0x01, 0xFF, 0xFE}, Version: 99},
	}

	require.NoError(t, Write(path, "ns", meta, want))

	_, got, err := ReadAll(path)
	require.NoError(t, err)
	require.Equal(t, want[0].Value, got[0].Value)
}
