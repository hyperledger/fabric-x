/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/msppb"
)

const (
	testOrg2 = "Org2MSP"
	testOrg3 = "Org3MSP"
)

// Helper function to create a test transaction with endorsements.
func createTestTx(namespaces []string, endorsements map[string][]string) *applicationpb.Tx {
	tx := &applicationpb.Tx{
		Namespaces:   make([]*applicationpb.TxNamespace, len(namespaces)),
		Endorsements: make([]*applicationpb.Endorsements, len(namespaces)),
	}

	for i, ns := range namespaces {
		tx.Namespaces[i] = &applicationpb.TxNamespace{
			NsId: ns,
		}

		tx.Endorsements[i] = &applicationpb.Endorsements{
			EndorsementsWithIdentity: make([]*applicationpb.EndorsementWithIdentity, 0),
		}

		if mspIDs, ok := endorsements[ns]; ok {
			for _, mspID := range mspIDs {
				identity := &msppb.Identity{
					MspId: mspID,
				}
				tx.Endorsements[i].EndorsementsWithIdentity = append(
					tx.Endorsements[i].EndorsementsWithIdentity,
					&applicationpb.EndorsementWithIdentity{
						Identity:    identity,
						Endorsement: []byte("sig-" + mspID),
					},
				)
			}
		}
	}

	return tx
}

func TestMerge_ErrorCases(t *testing.T) {
	t.Parallel()

	t.Run("less than two transactions", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			txs  []*applicationpb.Tx
		}{
			{
				name: "empty slice",
				txs:  []*applicationpb.Tx{},
			},
			{
				name: "single transaction",
				txs: []*applicationpb.Tx{
					createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}}),
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := Merge(tt.txs)
				require.Error(t, err)
				require.Nil(t, result)
				require.Contains(t, err.Error(), "at least two transactions required")
			})
		}
	})

	t.Run("transaction content mismatch", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx2 := createTestTx([]string{testNs2}, map[string][]string{testNs2: {testOrg2}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "content mismatch")
	})

	t.Run("transaction with empty endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "requires at least one endorsement")
	})

	t.Run("conflicting namespace writes", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg2}})

		tx1.Namespaces[0].ReadWrites = []*applicationpb.ReadWrite{{
			Key:   []byte("asset-1"),
			Value: []byte("value-a"),
		}}
		tx2.Namespaces[0].ReadWrites = []*applicationpb.ReadWrite{{
			Key:   []byte("asset-1"),
			Value: []byte("value-b"),
		}}

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "content mismatch")
	})
}

func TestMerge_NilTransaction(t *testing.T) {
	t.Parallel()

	result, err := Merge([]*applicationpb.Tx{nil, nil})
	require.Error(t, err)
	require.Nil(t, result)
}

func TestMerge_SingleNamespace(t *testing.T) {
	t.Parallel()

	t.Run("two transactions with different endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg2}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Endorsements, 1)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 2)

		require.Equal(t, testOrg1, result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, testOrg2, result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())
	})

	t.Run("three transactions with different endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg2}})
		tx3 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg3}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2, tx3})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Endorsements, 1)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 3)

		require.Equal(t, testOrg1, result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, testOrg2, result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())
		require.Equal(t, testOrg3, result.Endorsements[0].EndorsementsWithIdentity[2].Identity.GetMspId())
	})

	t.Run("deduplication - same MspId in multiple transactions", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Endorsements, 1)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 1)
		require.Equal(t, testOrg1, result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
	})

	t.Run("sorting - unsorted input should be sorted in output", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg3}})
		tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx3 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg2}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2, tx3})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 3)
		require.Equal(t, testOrg1, result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, testOrg2, result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())
		require.Equal(t, testOrg3, result.Endorsements[0].EndorsementsWithIdentity[2].Identity.GetMspId())
	})
}

func TestMerge_MultipleNamespaces(t *testing.T) {
	t.Parallel()

	t.Run("two namespaces with different endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx(
			[]string{testNs1, testNs2},
			map[string][]string{
				testNs1: {testOrg1},
				testNs2: {testOrg1},
			},
		)
		tx2 := createTestTx(
			[]string{testNs1, testNs2},
			map[string][]string{
				testNs1: {testOrg2},
				testNs2: {testOrg3},
			},
		)

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Endorsements, 2)

		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 2)
		require.Equal(t, testOrg1, result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, testOrg2, result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())

		require.Len(t, result.Endorsements[1].EndorsementsWithIdentity, 2)
		require.Equal(t, testOrg1, result.Endorsements[1].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, testOrg3, result.Endorsements[1].EndorsementsWithIdentity[1].Identity.GetMspId())
	})

	t.Run("three namespaces with mixed endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx(
			[]string{testNs1, testNs2, testNs3},
			map[string][]string{
				testNs1: {testOrg1},
				testNs2: {testOrg2},
				testNs3: {testOrg3},
			},
		)
		tx2 := createTestTx(
			[]string{testNs1, testNs2, testNs3},
			map[string][]string{
				testNs1: {"Org4MSP"},
				testNs2: {"Org5MSP"},
				testNs3: {"Org6MSP"},
			},
		)

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Endorsements, 3)

		for i := range 3 {
			require.Len(t, result.Endorsements[i].EndorsementsWithIdentity, 2)
		}
	})

	t.Run("deduplication across multiple namespaces", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx(
			[]string{testNs1, testNs2},
			map[string][]string{
				testNs1: {testOrg1},
				testNs2: {testOrg1},
			},
		)
		tx2 := createTestTx(
			[]string{testNs1, testNs2},
			map[string][]string{
				testNs1: {testOrg1},
				testNs2: {testOrg1},
			},
		)

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Endorsements, 2)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 1)
		require.Len(t, result.Endorsements[1].EndorsementsWithIdentity, 1)
		require.Equal(t, testOrg1, result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, testOrg1, result.Endorsements[1].EndorsementsWithIdentity[0].Identity.GetMspId())
	})
}

func TestMerge_PreservesTransactionContent(t *testing.T) {
	t.Parallel()

	t.Run("merged transaction preserves namespace data", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
		tx1.Namespaces[0].ReadWrites = []*applicationpb.ReadWrite{
			{Key: []byte("key1"), Value: []byte("value1")},
		}

		tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg2}})
		tx2.Namespaces[0].ReadWrites = []*applicationpb.ReadWrite{
			{Key: []byte("key1"), Value: []byte("value1")},
		}

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Len(t, result.Namespaces, 1)
		require.Equal(t, testNs1, result.Namespaces[0].NsId)
		require.Len(t, result.Namespaces[0].ReadWrites, 1)
		require.Equal(t, []byte("key1"), result.Namespaces[0].ReadWrites[0].Key)

		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 2)
	})
}

func TestMerge_EmptyReadWriteSetIsValid(t *testing.T) {
	t.Parallel()

	tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1}})
	tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg2}})

	result, err := Merge([]*applicationpb.Tx{tx1, tx2})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Namespaces, 1)
	require.Empty(t, result.Namespaces[0].ReadWrites)
	require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 2)
}

func TestMerge_DuplicateEndorsements(t *testing.T) {
	t.Parallel()

	tx1 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg1, testOrg1, testOrg2}})
	tx2 := createTestTx([]string{testNs1}, map[string][]string{testNs1: {testOrg2, testOrg3}})

	result, err := Merge([]*applicationpb.Tx{tx1, tx2})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Endorsements, 1)
	require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 3)
	require.Equal(t, testOrg1, result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
	require.Equal(t, testOrg2, result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())
	require.Equal(t, testOrg3, result.Endorsements[0].EndorsementsWithIdentity[2].Identity.GetMspId())
}