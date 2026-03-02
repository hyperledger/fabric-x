package io

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

type Codec interface {
	Encode(txID string, tx *applicationpb.Tx) ([]byte, error)
	Decode(data []byte) (txID string, tx *applicationpb.Tx, err error)
}

type JsonCodec struct{}

func (jc *JsonCodec) Encode(txID string, tx *applicationpb.Tx) ([]byte, error) {
	var txJSON map[string]interface{}

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

	output := map[string]interface{}{
		"txID": txID,
		"tx":   txJSON,
	}

	return json.MarshalIndent(output, "", "  ")
}

func (jc *JsonCodec) Decode(data []byte) (string, *applicationpb.Tx, error) {
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
