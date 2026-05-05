/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// Merge combines multiple endorsed transactions into a single transaction.
// It validates that all transactions have identical namespace content, then merges
// their endorsements while deduplicating by MSP ID. Requires at least 2 transactions.
// The merged endorsements are sorted alphabetically by MSP ID.
func Merge(txs []*applicationpb.Tx) (*applicationpb.Tx, error) {
	if len(txs) < 2 {
		return nil, errors.New("at least two transactions required for merge")
	}

	if err := validateTransactionsForMerge(txs); err != nil {
		return nil, err
	}

	merged := proto.CloneOf(txs[0])
	if merged == nil {
		return nil, errors.New("failed to clone base transaction")
	}
	merged.Endorsements = mergeEndorsements(txs)
	sortEndorsementsByMspID(merged)

	return merged, nil
}

func sortEndorsementsByMspID(merged *applicationpb.Tx) {
	for k := range merged.GetEndorsements() {
		// we sort endorsements by the MspID of the endorser
		sort.Slice(merged.Endorsements[k].EndorsementsWithIdentity, func(i, j int) bool {
			return strings.Compare(
				merged.Endorsements[k].EndorsementsWithIdentity[i].Identity.MspId,
				merged.Endorsements[k].EndorsementsWithIdentity[j].Identity.MspId,
			) < 0
		})
	}
}

func validateTransactionsForMerge(txs []*applicationpb.Tx) error {
	baseTx := txs[0]
	baseNsCount := len(baseTx.GetNamespaces())

	for i, tx := range txs {
		if len(tx.GetNamespaces()) != baseNsCount {
			return fmt.Errorf("transaction %d: namespace count mismatch", i)
		}

		if len(tx.GetEndorsements()) != baseNsCount {
			return fmt.Errorf(
				"transaction %d: endorsements count (%d) does not match namespaces count (%d)",
				i,
				len(tx.GetEndorsements()),
				baseNsCount,
			)
		}

		for nsIdx := range baseTx.GetNamespaces() {
			if !proto.Equal(baseTx.GetNamespaces()[nsIdx], txs[i].GetNamespaces()[nsIdx]) {
				return fmt.Errorf("transaction %d: namespace %d content mismatch", i, nsIdx)
			}
		}

		for nsIdx, ns := range tx.GetEndorsements() {
			if len(ns.GetEndorsementsWithIdentity()) == 0 {
				return fmt.Errorf("transaction %d: namespace %d requires at least one endorsement", i, nsIdx)
			}
		}
	}

	return nil
}

func mergeEndorsements(txs []*applicationpb.Tx) []*applicationpb.Endorsements {
	numNamespaces := len(txs[0].GetNamespaces())
	merged := make([]*applicationpb.Endorsements, numNamespaces)
	seen := make([]map[string]struct{}, numNamespaces)

	// Initialize merged and seen for each namespace
	for i := range numNamespaces {
		merged[i] = &applicationpb.Endorsements{
			EndorsementsWithIdentity: make([]*applicationpb.EndorsementWithIdentity, 0),
		}
		seen[i] = make(map[string]struct{})
	}

	for _, tx := range txs {
		for nsIdx, ns := range tx.GetEndorsements() {
			for _, e := range ns.GetEndorsementsWithIdentity() {
				key := e.GetIdentity().GetMspId()
				if _, exists := seen[nsIdx][key]; exists {
					// we have seen
					continue
				}

				seen[nsIdx][key] = struct{}{}
				merged[nsIdx].EndorsementsWithIdentity = append(merged[nsIdx].EndorsementsWithIdentity, e)
			}
		}
	}

	return merged
}
