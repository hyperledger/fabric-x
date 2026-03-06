/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x-common/protoutil"
)

// CreateNamespacesTx builds a transaction to create or update a namespace policy.
// Writes to the meta-namespace. Use version -1 for create, >= 0 for update.
func CreateNamespacesTx(nsPolicy *applicationpb.NamespacePolicy, nsID string, nsVersion int) *applicationpb.Tx {
	writeToMetaNs := &applicationpb.TxNamespace{
		NsId: committerpb.MetaNamespaceID,
		// TODO: we need the correct version of the metaNamespaceID
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
