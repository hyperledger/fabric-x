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
	"fmt"
	"io"
	"os"

	"github.com/hyperledger/fabric-x/tools/fxmigrate/internal/snapshot"
)

// ReadHeader opens the genesis-data file at path and returns its header.
// The file is closed after reading the header.
func ReadHeader(path string) (*FileHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open genesis file: %w", err)
	}
	defer func() { _ = f.Close() }()

	r := bufio.NewReader(f)
	return readHeader(r)
}

// ReadAll opens the genesis-data file at path, parses the header and all
// entries, and returns them. For large files use a streaming approach instead.
func ReadAll(path string) (*FileHeader, []snapshot.StateEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open genesis file: %w", err)
	}
	defer func() { _ = f.Close() }()

	r := bufio.NewReaderSize(f, 1<<20)

	hdr, err := readHeader(r)
	if err != nil {
		return nil, nil, err
	}

	entries := make([]snapshot.StateEntry, 0, hdr.EntryCount)
	for {
		key, kerr := readBytes(r)
		if errors.Is(kerr, io.EOF) {
			break
		}
		if kerr != nil {
			return nil, nil, fmt.Errorf("reading key: %w", kerr)
		}

		value, verr := readBytes(r)
		if verr != nil {
			return nil, nil, fmt.Errorf("reading value: %w", verr)
		}

		version, xerr := readUint64(r)
		if xerr != nil {
			return nil, nil, fmt.Errorf("reading version: %w", xerr)
		}

		entries = append(entries, snapshot.StateEntry{
			Key:     string(key),
			Value:   value,
			Version: version,
		})
	}

	return hdr, entries, nil
}

func readHeader(r *bufio.Reader) (*FileHeader, error) {
	hdrLen, err := readUint32(r)
	if err != nil {
		return nil, fmt.Errorf("reading header length: %w", err)
	}

	hdrBytes := make([]byte, hdrLen)
	if _, err := io.ReadFull(r, hdrBytes); err != nil {
		return nil, fmt.Errorf("reading header bytes: %w", err)
	}

	var hdr FileHeader
	if err := json.Unmarshal(hdrBytes, &hdr); err != nil {
		return nil, fmt.Errorf("parsing header: %w", err)
	}

	return &hdr, nil
}

func readUint32(r io.Reader) (uint32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf[:]), nil
}

func readUint64(r io.Reader) (uint64, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf[:]), nil
}

func readBytes(r io.Reader) ([]byte, error) {
	length, err := readUint32(r)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
