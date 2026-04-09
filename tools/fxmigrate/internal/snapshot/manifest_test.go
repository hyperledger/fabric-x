/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	meta := Manifest{
		ChannelName:     "mychannel",
		LastBlockNumber: 42,
		StateDBType:     "goleveldb",
		FileHashes:      map[string]string{},
	}
	data, err := json.Marshal(meta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, signableMetadataFile), data, 0o600))

	got, err := ReadManifest(dir)
	require.NoError(t, err)
	require.Equal(t, "mychannel", got.ChannelName)
	require.Equal(t, uint64(42), got.LastBlockNumber)
	require.Equal(t, "goleveldb", got.StateDBType)
}

func TestReadManifest_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := ReadManifest(t.TempDir())
	require.Error(t, err)
}

func TestReadManifest_EmptyChannelName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	meta := Manifest{LastBlockNumber: 1, StateDBType: "goleveldb", FileHashes: map[string]string{}}
	data, err := json.Marshal(meta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, signableMetadataFile), data, 0o600))

	_, err = ReadManifest(dir)
	require.ErrorContains(t, err, "channel_name is empty")
}
