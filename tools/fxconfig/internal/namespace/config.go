/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"time"

	"github.com/hyperledger/fabric-x-common/cmd/common/comm"

	"github.com/hyperledger/fabric-x-committer/service/verifier/policy"
)

// DefaultTimeout for gRPC connections.
const DefaultTimeout = 3 * time.Second

// OrdererConfig is a helper struct to deal with orderer-related arguments.
type OrdererConfig struct {
	OrderingEndpoint string
	Config           comm.Config
}

// MSPConfig is a helper struct to deal with MSP-related arguments.
type MSPConfig struct {
	MSPConfigPath string
	MSPID         string
}

// NsConfig is a helper struct to deal with namespace related arguments.
type NsConfig struct {
	Channel             string
	NamespaceID         string
	Version             int
	VerificationKeyPath string
}

func validateConfig(nsCfg NsConfig) error {
	return policy.ValidateNamespaceID(nsCfg.NamespaceID)
}
