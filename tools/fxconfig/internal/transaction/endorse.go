/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/msppb"
	"github.com/hyperledger/fabric-x-common/msp"
)

// Endorse signs a transaction with the provided identity for all namespaces.
// Returns a cloned transaction with added endorsements. Currently uses threshold ECDSA;
// MSP-based endorsement support is planned.
func Endorse(signer msp.SigningIdentity, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}

	tx = proto.CloneOf(tx)

	// check that tx does not yet carry any endorsements
	if tx.Endorsements == nil {
		tx.Endorsements = make([]*applicationpb.Endorsements, len(tx.GetNamespaces()))
	}

	// get signer identity to be attached to the endorsement
	signerIdentity, err := identity(signer)
	if err != nil {
		return nil, err
	}

	// create signature for each namespace in transaction
	for nsIdx := range tx.GetNamespaces() {
		// Note that a default msp signer hash the msg before signing.
		// For that reason we use the TxNamespace message as ASN1 encoded msg

		msg, err := tx.Namespaces[nsIdx].ASN1Marshal(txID)
		if err != nil {
			return nil, fmt.Errorf("failed asn1 marshal tx: %w", err)
		}

		sig, err := signer.Sign(msg)
		if err != nil {
			return nil, fmt.Errorf("failed signing tx: %w", err)
		}

		// store signature as endorsementWithIdentity
		eid := &applicationpb.EndorsementWithIdentity{
			Endorsement: sig,
			Identity:    signerIdentity,
		}

		// check if there is already an endorsement for this namespace, so we can append the new endorsement
		// if not we create an empty endorser set
		if tx.Endorsements[nsIdx] == nil {
			tx.Endorsements[nsIdx] = &applicationpb.Endorsements{
				EndorsementsWithIdentity: []*applicationpb.EndorsementWithIdentity{},
			}
		}

		tx.Endorsements[nsIdx].EndorsementsWithIdentity = append(tx.Endorsements[nsIdx].EndorsementsWithIdentity, eid)
	}

	return tx, nil
}

func identity(signer msp.SigningIdentity) (*msppb.Identity, error) {
	// signer identity with certificate attached
	s, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	// signer identity with hash of certificate attached
	// s, err := signer.SerializeWithIDOfCert()
	// if err != nil {
	// 	 return nil, err
	// }

	var sid msppb.Identity
	err = proto.Unmarshal(s, &sid)
	if err != nil {
		return nil, err
	}

	return &sid, nil
}

// GenerateTxID generates a unique transaction ID using SHA-256 hash of a random nonce.
func GenerateTxID() (string, error) {
	nonce, err := readNonce(nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate tx ID: %w", err)
	}
	hasher := sha256.New()
	hasher.Write(nonce)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// readNonce reads a byte array of the given size from the source.
// If source is nil, crypto/rand.Reader is used.
func readNonce(source io.Reader) ([]byte, error) {
	if source == nil {
		source = rand.Reader
	}

	size := 24
	value := make([]byte, size)
	n, err := source.Read(value)
	if err != nil {
		return nil, fmt.Errorf("error while creating nonce: %w", err)
	}
	if n != size {
		return nil, fmt.Errorf("cannot read enough bytes for nonce actual: %d wanted: %d", n, size)
	}

	return value, nil
}
