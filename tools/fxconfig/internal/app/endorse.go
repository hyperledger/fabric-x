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

// EndorseTransaction receives a transaction as input and endorses it.
func (d *AdminApp) EndorseTransaction(
	ctx context.Context,
	txID string,
	tx *applicationpb.Tx,
) (*applicationpb.Tx, error) {
	return d.endorseTransaction(ctx, txID, tx)
}

// endorseTransaction signs the transaction with the configured MSP identity.
func (d *AdminApp) endorseTransaction(
	_ context.Context,
	txID string,
	tx *applicationpb.Tx,
) (*applicationpb.Tx, error) {
	auditLogger := audit.MustGetAuditLogger(nil)

	// get signing identity
	sid, err := d.MspProvider.Get()
	if err != nil {
		return nil, err
	}

	auditLogger.TransactionEndorsementStarted(context.Background(), audit.TransactionEndorsementStartedEvent{
		EventMeta: audit.NewEventMeta(),
		TxID:      txID,
	})

	// Endorse transaction
	tx, err = transaction.Endorse(sid, txID, tx)
	if err != nil {
		signerID := ""
		if sid.GetIdentifier() != nil {
			signerID = sid.GetIdentifier().Id
		}
		auditLogger.TransactionEndorsed(context.Background(), audit.TransactionEndorsedEvent{
			EventMeta:      audit.NewEventMeta(),
			TxID:           txID,
			SignerID:       signerID,
			SignerType:     "msp",
			NamespaceCount: len(tx.GetNamespaces()),
			Result:         "failure",
			ErrorMsg:       err.Error(),
		})
		return nil, err
	}

	signatureCount := 0
	for _, ns := range tx.GetEndorsements() {
		signatureCount += len(ns.GetEndorsementsWithIdentity())
	}

	signerID := ""
	if sid.GetIdentifier() != nil {
		signerID = sid.GetIdentifier().Id
	}

	auditLogger.TransactionEndorsed(context.Background(), audit.TransactionEndorsedEvent{
		EventMeta:      audit.NewEventMeta(),
		TxID:           txID,
		SignerID:       signerID,
		SignerType:     "msp",
		NamespaceCount: len(tx.GetNamespaces()),
		SignatureCount: signatureCount,
		Result:         "success",
	})

	return tx, nil
}
