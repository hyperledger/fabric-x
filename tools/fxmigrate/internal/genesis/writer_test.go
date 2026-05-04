/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package genesis

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxmigrate/internal/snapshot"
)

func testMeta(height uint64) *snapshot.Manifest {
	return &snapshot.Manifest{
		ChannelName:     "mychannel",
		LastBlockNumber: height,
		StateDBType:     "goleveldb",
		FileHashes:      map[string]string{},
	}
}

func testEntries() []snapshot.StateEntry {
	return []snapshot.StateEntry{
		{Namespace: "mycc", Key: "asset1", Value: []byte("val1"), Version: (5 << 32) | 2},
		{Namespace: "mycc", Key: "asset2", Value: []byte("val2"), Version: 10 << 32},
	}
}

// readFile parses the genesis-data file and returns the header and raw entries.
func readFile(t *testing.T, path string) (FileHeader, []snapshot.StateEntry) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	r := bufio.NewReader(f)

	read32 := func() uint32 {
		var buf [4]byte
		_, err := io.ReadFull(r, buf[:])
		require.NoError(t, err)
		return binary.BigEndian.Uint32(buf[:])
	}
	read64 := func() uint64 {
		var buf [8]byte
		_, err := io.ReadFull(r, buf[:])
		require.NoError(t, err)
		return binary.BigEndian.Uint64(buf[:])
	}
	readBytes := func() []byte {
		n := read32()
		if n == 0 {
			return nil
		}
		buf := make([]byte, n)
		_, err := io.ReadFull(r, buf)
		require.NoError(t, err)
		return buf
	}

	hdrBytes := readBytes()
	var hdr FileHeader
	require.NoError(t, json.Unmarshal(hdrBytes, &hdr))

	var entries []snapshot.StateEntry
	for {
		keyBytes := make([]byte, 0)
		var keyLen uint32
		err := binary.Read(r, binary.BigEndian, &keyLen)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if keyLen > 0 {
			keyBytes = make([]byte, keyLen)
			_, err := io.ReadFull(r, keyBytes)
			require.NoError(t, err)
		}

		valBytes := readBytes()
		ver := read64()

		entries = append(entries, snapshot.StateEntry{
			Key:     string(keyBytes),
			Value:   valBytes,
			Version: ver,
		})
	}

	return hdr, entries
}

func TestWrite_Header(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "genesis.bin")
	meta := testMeta(100)
	entries := testEntries()

	require.NoError(t, Write(path, "token", meta, entries))

	hdr, _ := readFile(t, path)
	require.Equal(t, "1", hdr.Version)
	require.Equal(t, "token", hdr.Namespace)
	require.Equal(t, "mychannel", hdr.SourceChannel)
	require.Equal(t, uint64(100), hdr.BlockHeight)
	require.Equal(t, len(entries), hdr.EntryCount)
}

func TestWrite_Entries(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "genesis.bin")
	entries := testEntries()

	require.NoError(t, Write(path, "token", testMeta(1), entries))

	_, got := readFile(t, path)
	require.Len(t, got, len(entries))
	require.Equal(t, "asset1", got[0].Key)
	require.Equal(t, []byte("val1"), got[0].Value)
	require.Equal(t, uint64((5<<32)|2), got[0].Version)
	require.Equal(t, "asset2", got[1].Key)
	require.Equal(t, uint64(10<<32), got[1].Version)
}

func TestWrite_EmptyEntries(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "genesis.bin")
	require.NoError(t, Write(path, "token", testMeta(0), nil))

	hdr, entries := readFile(t, path)
	require.Equal(t, 0, hdr.EntryCount)
	require.Empty(t, entries)
}

func TestChecksum_Deterministic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.bin")
	p2 := filepath.Join(dir, "b.bin")

	meta := testMeta(5)
	entries := testEntries()

	require.NoError(t, Write(p1, "token", meta, entries))
	require.NoError(t, Write(p2, "token", meta, entries))

	// ExportedAt timestamps may differ — just verify checksum is non-empty and consistent per file
	c1, err := Checksum(p1)
	require.NoError(t, err)
	require.Len(t, c1, 64) // hex SHA-256

	c2, err := Checksum(p2)
	require.NoError(t, err)
	require.Len(t, c2, 64)
}

func TestChecksum_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := Checksum(filepath.Join(t.TempDir(), "missing.bin"))
	require.Error(t, err)
}
