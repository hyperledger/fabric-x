/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"errors"
	"strings"
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
	Channel                            string
	NamespaceID                        string
	Version                            int
	ThresholdPolicyVerificationKeyPath string
}

func validateConfig(nsCfg NsConfig) error {
	return errors.Join(
		validateChannel(nsCfg),
		policy.ValidateNamespaceID(nsCfg.NamespaceID),
		validateVersion(nsCfg),
		mustHavePolicy(nsCfg),
	)
}

func validateChannel(nsCfg NsConfig) error {
	if isEmpty(nsCfg.Channel) {
		return errors.New("channel name must be specified")
	}
	return nil
}

func validateVersion(nsCfg NsConfig) error {
	if nsCfg.Version < -1 {
		return errors.New("invalid version: must be -1 (create) or >= 0 (update)")
	}
	return nil
}

func mustHavePolicy(nsCfg NsConfig) error {
	if isEmpty(nsCfg.ThresholdPolicyVerificationKeyPath) {
		return errors.New("policy verification key path must be specified")
	}
	return nil
}

func isEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}
