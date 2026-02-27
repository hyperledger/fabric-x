/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package app provides functionality for namespace lifecycle operations.
// It handles creating, updating, and listing namespaces in Fabric-X.
package app

import (
	"context"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/client"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// DeployNamespace creates a namespace transaction and submits it to the ordering service.
func DeployNamespace(vctx config.ValidationContext, cfg config.Config, nsCfg config.NsConfig) error {
	if err := validate(vctx, cfg, nsCfg); err != nil {
		return err
	}

	tx, err := transaction.CreateNamespaceTransaction(nsCfg)
	if err != nil {
		return err
	}

	// prepare endorsement
	sid, err := msp.GetSignerIdentityFromMSP(cfg.MSP)
	if err != nil {
		return err
	}

	// generate txID
	txID := transaction.GenerateTxID()

	// endorse transaction
	tx, err = transaction.Endorse(sid, txID, tx)
	if err != nil {
		return err
	}

	// submit transaction
	// note that we use the endorser identity to submit the transaction
	oc, err := client.NewOrdererClient(cfg.Orderer, sid)
	if err != nil {
		return err
	}

	return oc.Broadcast(context.TODO(), txID, tx)
}

func validate(ctx config.ValidationContext, cfg config.Config, nsCfg config.NsConfig) error {
	if err := cfg.Orderer.Validate(ctx); err != nil {
		return err
	}

	if err := cfg.MSP.Validate(ctx); err != nil {
		return err
	}

	return nsCfg.Validate(ctx)
}
