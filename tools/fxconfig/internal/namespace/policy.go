/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/common/policydsl"
	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x-common/tools/configtxgen"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// createPolicy creates a namespace policy from configuration.
// Supports MSP-based and threshold ECDSA policies.
func createPolicy(cfg config.PolicyConfig) (*applicationpb.NamespacePolicy, error) {
	switch cfg.Type {
	case "msp":
		return createMspPolicy(cfg.MSP.Expression)

	case "threshold":
		return createThresholdPolicy(cfg.Threshold.VerificationKeyPath)

	default:
		return nil, fmt.Errorf("unknown policy type: %s", cfg.Type)
	}
}

// createMspPolicy creates an MSP-based namespace policy from a DSL expression.
func createMspPolicy(policy string) (*applicationpb.NamespacePolicy, error) {
	p, err := policydsl.FromString(policy)
	if err != nil {
		return nil, err
	}

	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_MspRule{
			MspRule: protoutil.MarshalOrPanic(p),
		},
	}

	return nsPolicy, nil
}

// createThresholdPolicy creates a threshold ECDSA namespace policy from PEM-encoded key data.
func createThresholdPolicy(path string) (*applicationpb.NamespacePolicy, error) {
	pkData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	serializedPublicKey, err := getPubKeyFromPemData(pkData)
	if err != nil {
		return nil, err
	}

	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_ThresholdRule{
			ThresholdRule: &applicationpb.ThresholdRule{
				Scheme:    "ECDSA",
				PublicKey: serializedPublicKey,
			},
		},
	}

	return nsPolicy, nil
}

// getPubKeyFromPemData extracts an ECDSA public key from PEM-encoded content.
// It searches through multiple PEM blocks and returns the first valid ECDSA public key found.
func getPubKeyFromPemData(pemContent []byte) ([]byte, error) {
	for {
		block, rest := pem.Decode(pemContent)
		if block == nil {
			break
		}
		pemContent = rest

		key, err := configtxgen.ParseCertificateOrPublicKey(block.Bytes)
		if err != nil {
			continue
		}

		return pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: key,
		}), nil
	}

	return nil, errors.New("no ECDSA public key in pem file")
}
