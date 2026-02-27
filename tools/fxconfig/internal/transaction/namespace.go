/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// CreateNamespaceTransaction creates a namespace transaction.
func CreateNamespaceTransaction(nsCfg config.NsConfig) (*applicationpb.Tx, error) {
	// create endorsement policy
	nsPolicy, err := createPolicy(nsCfg.Policy)
	if err != nil {
		return nil, err
	}

	// create transaction
	tx := createNamespacesTx(nsPolicy, nsCfg.NamespaceID, nsCfg.Version)

	return tx, nil
}

// createNamespacesTx constructs a transaction for creating or updating a namespace.
// The transaction writes to the meta-namespace with the namespace policy.
// Version -1 indicates a create operation; >= 0 indicates an update.
func createNamespacesTx(nsPolicy *applicationpb.NamespacePolicy, nsID string, nsVersion int) *applicationpb.Tx {
	writeToMetaNs := &applicationpb.TxNamespace{
		NsId: committerpb.MetaNamespaceID,
		// TODO we need the correct version of the metaNamespaceID
		NsVersion:  0,
		ReadWrites: make([]*applicationpb.ReadWrite, 0, 1),
	}

	policyBytes := protoutil.MarshalOrPanic(nsPolicy)
	rw := &applicationpb.ReadWrite{
		Key:   []byte(nsID),
		Value: policyBytes,
	}

	// note that we only set the version if we update a namespace policy
	if nsVersion >= 0 {
		rw.Version = applicationpb.NewVersion(uint64(nsVersion))
	}

	writeToMetaNs.ReadWrites = append(writeToMetaNs.ReadWrites, rw)

	tx := &applicationpb.Tx{
		Namespaces: []*applicationpb.TxNamespace{
			writeToMetaNs,
		},
	}

	return tx
}
