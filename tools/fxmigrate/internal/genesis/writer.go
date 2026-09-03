/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package genesis handles writing the Fabric-X genesis-data file that the
// committer's --init-from-snapshot bootstrap mode ingests.
package genesis

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hyperledger/fabric-x/tools/fxmigrate/internal/snapshot"
)

// FileHeader is written at the start of every genesis-data file so the
// committer can reject files intended for a different namespace or produced
// by a different tool version.
type FileHeader struct {
	Version       string `json:"version"`
	Namespace     string `json:"namespace"`
	SourceChannel string `json:"source_channel"`
	BlockHeight   uint64 `json:"block_height"`
	ExportedAt    string `json:"exported_at"`
	EntryCount    int    `json:"entry_count"`
}

// Write creates the genesis-data file at path.
//
// File format (sequential, no framing overhead):
//
//	[header_len uint32][header JSON bytes]
//	for each entry:
//	  [key_len uint32][key bytes]
//	  [value_len uint32][value bytes]
//	  [version uint64]
func Write(path, namespace string, meta *snapshot.Manifest, entries []snapshot.StateEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create output file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 1<<20)

	// Write header
	hdr := FileHeader{
		Version:       "1",
		Namespace:     namespace,
		SourceChannel: meta.ChannelName,
		BlockHeight:   meta.LastBlockNumber,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		EntryCount:    len(entries),
	}
	hdrBytes, err := json.Marshal(hdr)
	if err != nil {
		return fmt.Errorf("cannot marshal header: %w", err)
	}
	if err := writeUint32(w, uint32(len(hdrBytes))); err != nil {
		return err
	}
	if _, err := w.Write(hdrBytes); err != nil {
		return err
	}

	// Write entries
	for _, e := range entries {
		if err := writeBytes(w, []byte(e.Key)); err != nil {
			return fmt.Errorf("writing key: %w", err)
		}
		if err := writeBytes(w, e.Value); err != nil {
			return fmt.Errorf("writing value: %w", err)
		}
		if err := writeUint64(w, e.Version); err != nil {
			return fmt.Errorf("writing version: %w", err)
		}
	}

	return w.Flush()
}

// Checksum returns the hex-encoded SHA-256 hash of the file at path.
func Checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeUint32(w io.Writer, v uint32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeUint64(w io.Writer, v uint64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeBytes(w io.Writer, b []byte) error {
	if err := writeUint32(w, uint32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}
