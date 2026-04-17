/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/audit"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

// MergeTransactions combines multiple transactions into a single transaction.
// Useful for collecting endorsements from multiple organizations.
func (*AdminApp) MergeTransactions(_ context.Context, txs []*applicationpb.Tx) (*applicationpb.Tx, error) {
	auditLogger := audit.MustGetAuditLogger(nil)

	// Collect tx IDs for audit (tx itself doesn't have an ID field)
	txIDs := make([]string, len(txs))

	auditLogger.TransactionMergeStarted(context.Background(), audit.TransactionMergeStartedEvent{
		EventMeta: audit.NewEventMeta(),
		TxCount:   len(txs),
		TxIDs:     txIDs,
	})

	merged, err := transaction.Merge(txs)
	if err != nil {
		auditLogger.TransactionMerged(context.Background(), audit.TransactionMergedEvent{
			EventMeta:  audit.NewEventMeta(),
			InputTxIDs: txIDs,
			Result:    "failure",
			ErrorMsg:  err.Error(),
		})
		return nil, err
	}

	totalEndorsements := 0
	uniqueEndorsers := make(map[string]bool)
	for _, ns := range merged.GetEndorsements() {
		totalEndorsements += len(ns.GetEndorsementsWithIdentity())
		for _, e := range ns.GetEndorsementsWithIdentity() {
			uniqueEndorsers[e.GetIdentity().GetMspId()] = true
		}
	}

	endorserList := make([]string, 0, len(uniqueEndorsers))
	for k := range uniqueEndorsers {
		endorserList = append(endorserList, k)
	}

	auditLogger.TransactionMerged(context.Background(), audit.TransactionMergedEvent{
		EventMeta:         audit.NewEventMeta(),
		InputTxIDs:        txIDs,
		MergedTxID:        "", // merged tx doesn't have a dedicated ID field
		NamespaceCount:    len(merged.GetNamespaces()),
		TotalEndorsements: totalEndorsements,
		UniqueEndorsers:   endorserList,
		Result:            "success",
	})

	return merged, nil
}
