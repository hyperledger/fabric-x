/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/tms"
)

var logger = logging.MustGetLogger()

type FabricSmartClient struct {
	node *node.Node
}

func NewFSC(node *node.Node) *FabricSmartClient {
	return &FabricSmartClient{node: node}
}

// Init must be called for fabric-x networks to initialize the token parameters on the ledger.
func (f FabricSmartClient) Init(ctx context.Context) error {
	logger.Info("initializing token parameters")
	dep, err := tms.GetTMSDeployerService(f.node)
	if err != nil {
		return err
	}

	if err := dep.DeployTMSs(); err != nil {
		return err
	}
	return nil
}
