// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cliio

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

func TestJSONCodec_Encode(t *testing.T) {
	t.Parallel()

	codec := &JSONCodec{}

	t.Run("success with valid transaction", func(t *testing.T) {
		t.Parallel()

		tx := &applicationpb.Tx{}
		txID := "tx-123"

		data, err := codec.Encode(txID, tx)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Verify JSON structure
		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, txID, result["txID"])
		require.NotNil(t, result["tx"])
	})

	t.Run("error with nil transaction", func(t *testing.T) {
		t.Parallel()

		txID := "tx-123"

		data, err := codec.Encode(txID, nil)
		require.Error(t, err)
		require.Nil(t, data)
		require.Contains(t, err.Error(), "tx is nil")
	})

	t.Run("success with empty transaction", func(t *testing.T) {
		t.Parallel()

		tx := &applicationpb.Tx{}
		txID := "tx-456"

		data, err := codec.Encode(txID, tx)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, txID, result["txID"])
	})
}

func TestJSONCodec_Decode(t *testing.T) {
	t.Parallel()

	codec := &JSONCodec{}

	t.Run("success with valid JSON", func(t *testing.T) {
		t.Parallel()

		input := `{
  "tx": {
    "endorsements": [
      {
        "endorsements_with_identity": [
          {
            "endorsement": "cGF5bWVudHM=",
            "identity": {
              "certificate_id": "03ca0f49dc4a7732672611c4ee1ca0362f73b495270eca1dd7a60fede56e4f92",
              "msp_id": "Org1MSP"
            }
          }
        ]
      }
    ],
    "namespaces": [
      {
        "ns_id": "_meta",
        "read_writes": [
          {
            "key": "cGF5bWVudHM=",
            "value": "EigSDBIKCAISAggAEgIIARoLEgkKB09yZzFNU1AaCxIJCgdPcmcyTVNQ"
          }
        ]
      }
    ]
  },
  "txID": "1ca34644c2ac3b7570cc09fee527757d603e282ed03392df805c2e00437ab224"
}`

		txID, tx, err := codec.Decode([]byte(input))
		require.NoError(t, err)
		require.Equal(t, "1ca34644c2ac3b7570cc09fee527757d603e282ed03392df805c2e00437ab224", txID)
		require.NotNil(t, tx)
		require.Len(t, tx.GetNamespaces(), 1)
		require.Equal(t, "_meta", tx.GetNamespaces()[0].GetNsId())
	})

	t.Run("error with invalid JSON", func(t *testing.T) {
		t.Parallel()

		input := `{invalid json}`

		txID, tx, err := codec.Decode([]byte(input))
		require.Error(t, err)
		require.Empty(t, txID)
		require.Nil(t, tx)
	})

	t.Run("missing txID returns empty string without error", func(t *testing.T) {
		t.Parallel()

		input := `{
			"tx": {}
		}`

		txID, tx, err := codec.Decode([]byte(input))
		require.NoError(t, err)
		require.Empty(t, txID)
		require.NotNil(t, tx)
	})

	t.Run("error with missing tx", func(t *testing.T) {
		t.Parallel()

		input := `{
			"txID": "tx-123"
		}`

		txID, tx, err := codec.Decode([]byte(input))
		require.Error(t, err)
		require.Empty(t, txID)
		require.Nil(t, tx)
	})

	t.Run("error with empty input", func(t *testing.T) {
		t.Parallel()

		txID, tx, err := codec.Decode([]byte{})
		require.Error(t, err)
		require.Empty(t, txID)
		require.Nil(t, tx)
	})
}

func TestJSONCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	codec := &JSONCodec{}

	tx := &applicationpb.Tx{
		Namespaces: []*applicationpb.TxNamespace{{
			NsId:      "some_namespace",
			NsVersion: 0,
			BlindWrites: []*applicationpb.Write{{
				Key:   []byte("key"),
				Value: []byte("value"),
			}},
		}},
	}
	originalTxID := "tx-round-trip"

	// Encode
	encoded, err := codec.Encode(originalTxID, tx)
	require.NoError(t, err)

	// Decode
	decodedTxID, decodedTx, err := codec.Decode(encoded)
	require.NoError(t, err)

	// Verify
	require.Equal(t, originalTxID, decodedTxID)
	require.Equal(t, tx.GetNamespaces()[0].GetNsId(), decodedTx.GetNamespaces()[0].GetNsId())
	require.Equal(t, tx.GetNamespaces()[0].GetBlindWrites()[0], decodedTx.GetNamespaces()[0].GetBlindWrites()[0])
}
