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
				// Create identity using the msppb package
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
					createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}}),
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

		tx1 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})
		tx2 := createTestTx([]string{"ns2"}, map[string][]string{"ns2": {"Org2MSP"}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "content mismatch")
	})

	t.Run("transaction with empty endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})
		tx2 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "requires at least one endorsement")
	})
}

func TestMerge_SingleNamespace(t *testing.T) {
	t.Parallel()

	t.Run("two transactions with different endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})
		tx2 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org2MSP"}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have 2 endorsements
		require.Len(t, result.Endorsements, 1)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 2)

		// Should be sorted by MspId
		require.Equal(t, "Org1MSP", result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, "Org2MSP", result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())
	})

	t.Run("three transactions with different endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})
		tx2 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org2MSP"}})
		tx3 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org3MSP"}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2, tx3})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have 3 endorsements
		require.Len(t, result.Endorsements, 1)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 3)

		// Should be sorted by MspId
		require.Equal(t, "Org1MSP", result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, "Org2MSP", result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())
		require.Equal(t, "Org3MSP", result.Endorsements[0].EndorsementsWithIdentity[2].Identity.GetMspId())
	})

	t.Run("deduplication - same MspId in multiple transactions", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})
		tx2 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have only 1 endorsement (deduplicated)
		require.Len(t, result.Endorsements, 1)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 1)
		require.Equal(t, "Org1MSP", result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
	})

	t.Run("sorting - unsorted input should be sorted in output", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org3MSP"}})
		tx2 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})
		tx3 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org2MSP"}})

		result, err := Merge([]*applicationpb.Tx{tx1, tx2, tx3})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should be sorted alphabetically by MspId
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 3)
		require.Equal(t, "Org1MSP", result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, "Org2MSP", result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())
		require.Equal(t, "Org3MSP", result.Endorsements[0].EndorsementsWithIdentity[2].Identity.GetMspId())
	})
}

func TestMerge_MultipleNamespaces(t *testing.T) {
	t.Parallel()

	t.Run("two namespaces with different endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx(
			[]string{"ns1", "ns2"},
			map[string][]string{
				"ns1": {"Org1MSP"},
				"ns2": {"Org1MSP"},
			},
		)
		tx2 := createTestTx(
			[]string{"ns1", "ns2"},
			map[string][]string{
				"ns1": {"Org2MSP"},
				"ns2": {"Org3MSP"},
			},
		)

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have 2 namespaces
		require.Len(t, result.Endorsements, 2)

		// ns1 should have 2 endorsements (Org1MSP, Org2MSP)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 2)
		require.Equal(t, "Org1MSP", result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, "Org2MSP", result.Endorsements[0].EndorsementsWithIdentity[1].Identity.GetMspId())

		// ns2 should have 2 endorsements (Org1MSP, Org3MSP)
		require.Len(t, result.Endorsements[1].EndorsementsWithIdentity, 2)
		require.Equal(t, "Org1MSP", result.Endorsements[1].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, "Org3MSP", result.Endorsements[1].EndorsementsWithIdentity[1].Identity.GetMspId())
	})

	t.Run("three namespaces with mixed endorsements", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx(
			[]string{"ns1", "ns2", "ns3"},
			map[string][]string{
				"ns1": {"Org1MSP"},
				"ns2": {"Org2MSP"},
				"ns3": {"Org3MSP"},
			},
		)
		tx2 := createTestTx(
			[]string{"ns1", "ns2", "ns3"},
			map[string][]string{
				"ns1": {"Org4MSP"},
				"ns2": {"Org5MSP"},
				"ns3": {"Org6MSP"},
			},
		)

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have 3 namespaces
		require.Len(t, result.Endorsements, 3)

		// Each namespace should have 2 endorsements
		for i := range 3 {
			require.Len(t, result.Endorsements[i].EndorsementsWithIdentity, 2)
		}
	})

	t.Run("deduplication across multiple namespaces", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx(
			[]string{"ns1", "ns2"},
			map[string][]string{
				"ns1": {"Org1MSP"},
				"ns2": {"Org1MSP"},
			},
		)
		tx2 := createTestTx(
			[]string{"ns1", "ns2"},
			map[string][]string{
				"ns1": {"Org1MSP"},
				"ns2": {"Org1MSP"},
			},
		)

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Each namespace should have only 1 endorsement (deduplicated)
		require.Len(t, result.Endorsements, 2)
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 1)
		require.Len(t, result.Endorsements[1].EndorsementsWithIdentity, 1)
		require.Equal(t, "Org1MSP", result.Endorsements[0].EndorsementsWithIdentity[0].Identity.GetMspId())
		require.Equal(t, "Org1MSP", result.Endorsements[1].EndorsementsWithIdentity[0].Identity.GetMspId())
	})
}

func TestMerge_PreservesTransactionContent(t *testing.T) {
	t.Parallel()

	t.Run("merged transaction preserves namespace data", func(t *testing.T) {
		t.Parallel()

		tx1 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org1MSP"}})
		tx1.Namespaces[0].ReadWrites = []*applicationpb.ReadWrite{
			{Key: []byte("key1"), Value: []byte("value1")},
		}

		tx2 := createTestTx([]string{"ns1"}, map[string][]string{"ns1": {"Org2MSP"}})
		tx2.Namespaces[0].ReadWrites = []*applicationpb.ReadWrite{
			{Key: []byte("key1"), Value: []byte("value1")},
		}

		result, err := Merge([]*applicationpb.Tx{tx1, tx2})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Namespace data should be preserved
		require.Len(t, result.Namespaces, 1)
		require.Equal(t, "ns1", result.Namespaces[0].NsId)
		require.Len(t, result.Namespaces[0].ReadWrites, 1)
		require.Equal(t, []byte("key1"), result.Namespaces[0].ReadWrites[0].Key)

		// But endorsements should be merged
		require.Len(t, result.Endorsements[0].EndorsementsWithIdentity, 2)
	})
}
