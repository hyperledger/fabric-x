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
	"github.com/hyperledger/fabric-x-common/msp"
)

// Endorse signs the transaction with the provided identity.
// It creates endorsements for each namespace in the transaction.
// Currently, uses threshold ECDSA signatures; MSP-based endorsement will be added later.
func Endorse(signer msp.SigningIdentity, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}

	tx = proto.CloneOf(tx)

	// check that tx does not yet carry any endorsements
	if tx.Endorsements == nil {
		tx.Endorsements = make([]*applicationpb.Endorsements, len(tx.GetNamespaces()))
	}

	// TODO for MSP-based endorsements we need either singerID or the hashed singerID to be attached on the Endorsement.
	// get signer signerCert
	// signerID, err := getSignerID(signer)
	// if err != nil {
	//	 return nil, err
	// }

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
			// TODO MSP-based endorsements will attach either the signerID or just a hash.
			// Identity:    signerID,
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

// GenerateTxID produces a TxID.
func GenerateTxID() string {
	nonce := readNonce(nil)
	hasher := sha256.New()
	hasher.Write(nonce)
	return hex.EncodeToString(hasher.Sum(nil))
}

// readNonce reads a byte array of the given size from the source.
// It panics if the read fails, or cannot read the requested size.
// "crypto/rand" and "math/rand" never fail and always returns the correct length.
func readNonce(source io.Reader) []byte {
	if source == nil {
		source = rand.Reader
	}

	size := 24
	value := make([]byte, 24)
	n, err := source.Read(value)
	if err != nil {
		panic("ouch")
	}
	if n != size {
		panic("ouch ouch")
	}

	return value
}
