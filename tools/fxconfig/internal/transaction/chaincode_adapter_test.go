/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTxNamespaceFromChaincodeRWSet(t *testing.T) {
	t.Parallel()

	t.Run("maps chaincode rwset into tx namespace", func(t *testing.T) {
		t.Parallel()

		v := uint64(3)
		rwset := ChaincodeNamespaceRWSet{
			Namespace: "asset",
			Version:   2,
			Writes: []ChaincodeWrite{
				{Key: "a1", Value: []byte("v1")},
				{Key: "a2", Value: []byte("v2"), Version: &v},
			},
		}

		ns, err := CreateTxNamespaceFromChaincodeRWSet(rwset)
		require.NoError(t, err)
		require.Equal(t, "asset", ns.NsId)
		require.Equal(t, uint64(2), ns.NsVersion)
		require.Len(t, ns.ReadWrites, 2)
		require.Equal(t, []byte("a1"), ns.ReadWrites[0].Key)
		require.Equal(t, []byte("v1"), ns.ReadWrites[0].Value)
		require.Nil(t, ns.ReadWrites[0].Version)
		require.NotNil(t, ns.ReadWrites[1].Version)
		require.Equal(t, uint64(3), *ns.ReadWrites[1].Version)
	})

	t.Run("fails on empty namespace", func(t *testing.T) {
		t.Parallel()

		_, err := CreateTxNamespaceFromChaincodeRWSet(ChaincodeNamespaceRWSet{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "namespace is required")
	})

	t.Run("fails on empty key", func(t *testing.T) {
		t.Parallel()

		_, err := CreateTxNamespaceFromChaincodeRWSet(ChaincodeNamespaceRWSet{
			Namespace: "asset",
			Writes: []ChaincodeWrite{
				{Key: "", Value: []byte("v")},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "write key is required")
	})
}

func TestCreateTxFromChaincodeRWSets(t *testing.T) {
	t.Parallel()

	t.Run("maps multiple namespaces", func(t *testing.T) {
		t.Parallel()

		tx, err := CreateTxFromChaincodeRWSets([]ChaincodeNamespaceRWSet{
			{Namespace: "asset", Writes: []ChaincodeWrite{{Key: "k1", Value: []byte("v1")}}},
			{Namespace: "payment", Writes: []ChaincodeWrite{{Key: "k2", Value: []byte("v2")}}},
		})
		require.NoError(t, err)
		require.Len(t, tx.Namespaces, 2)
		require.Equal(t, "asset", tx.Namespaces[0].NsId)
		require.Equal(t, "payment", tx.Namespaces[1].NsId)
	})

	t.Run("fails on empty list", func(t *testing.T) {
		t.Parallel()

		_, err := CreateTxFromChaincodeRWSets(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one namespace rwset is required")
	})
}
