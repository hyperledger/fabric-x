/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/common/policydsl"
	"github.com/hyperledger/fabric-x-common/protoutil"
)

// CreateMspPolicy creates an MSP-based namespace policy from a DSL expression.
// Example: "OR('Org1MSP.member', 'Org2MSP.member')" or "AND('Org1MSP.admin', 'Org2MSP.admin')".
func CreateMspPolicy(policy string) (*applicationpb.NamespacePolicy, error) {
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

// CreateThresholdPolicy creates a threshold ECDSA namespace policy from a PEM file.
// The file must contain an ECDSA public key or X.509 certificate with an ECDSA key.
func CreateThresholdPolicy(path string) (*applicationpb.NamespacePolicy, error) {
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

		key, err := parseCertificateOrPublicKey(block.Bytes)
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

func parseCertificateOrPublicKey(blockBytes []byte) ([]byte, error) {
	// Try reading certificate
	cert, err := x509.ParseCertificate(blockBytes)
	var publicKey any
	if err == nil {
		if cert.PublicKey != nil && cert.PublicKeyAlgorithm == x509.ECDSA {
			publicKey = cert.PublicKey
		}
	} else {
		// If fails, try reading public key
		anyPublicKey, err := x509.ParsePKIXPublicKey(blockBytes)
		if err == nil && anyPublicKey != nil {
			publicKey, _ = anyPublicKey.(*ecdsa.PublicKey)

		}
	}

	if publicKey == nil {
		return nil, errors.New("no ECDSA public key in block")
	}

	key, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("marshalling public key from failed: %w", err)
	}
	return key, nil
}
