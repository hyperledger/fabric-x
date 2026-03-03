// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cliio

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// Codec handles encoding and decoding of transactions for CLI I/O.
type Codec interface {
	Encode(txID string, tx *applicationpb.Tx) ([]byte, error)
	Decode(data []byte) (txID string, tx *applicationpb.Tx, err error)
}

// JSONCodec implements Codec using JSON format with protobuf marshaling.
type JSONCodec struct{}

// Encode converts a transaction to JSON format with transaction ID.
func (*JSONCodec) Encode(txID string, tx *applicationpb.Tx) ([]byte, error) {
	var txJSON map[string]any

	if tx == nil {
		return nil, errors.New("tx is nil")
	}

	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}

	b, err := marshaler.Marshal(tx)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &txJSON); err != nil {
		return nil, err
	}

	output := map[string]any{
		"txID": txID,
		"tx":   txJSON,
	}

	return json.MarshalIndent(output, "", "  ")
}

// Decode parses JSON data into transaction ID and transaction.
func (*JSONCodec) Decode(data []byte) (string, *applicationpb.Tx, error) {
	carrier := struct {
		TxID string          `json:"txID"`
		Tx   json.RawMessage `json:"tx"`
	}{}

	err := json.Unmarshal(data, &carrier)
	if err != nil {
		return "", nil, err
	}

	var tx applicationpb.Tx
	err = protojson.Unmarshal(carrier.Tx, &tx)
	if err != nil {
		return "", nil, err
	}

	txID := carrier.TxID
	return txID, &tx, nil
}
