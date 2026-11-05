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

// CreateNamespacesTx builds a transaction to create or update a namespace policy in the meta-namespace.
//
// nsVersion is the version of the namespace policy being replaced: -1 for create, >= 0 for update.
// metaNsVersion is the version of the latest committed channel config; the committer rejects a
// meta-namespace write with ABORTED_MVCC_CONFLICT unless its NsVersion matches that version.
func CreateNamespacesTx(
	nsPolicy *applicationpb.NamespacePolicy, nsID string, nsVersion int, metaNsVersion uint64,
) *applicationpb.Tx {
	writeToMetaNs := &applicationpb.TxNamespace{
		NsId:       committerpb.MetaNamespaceID,
		NsVersion:  metaNsVersion,
		ReadWrites: make([]*applicationpb.ReadWrite, 0, 1),
	}

	policyBytes := protoutil.MarshalOrPanic(nsPolicy)
	rw := &applicationpb.ReadWrite{
		Key:   []byte(nsID),
		Value: policyBytes,
	}

	// note that we only set the version if we update a namespace policy
	if nsVersion >= 0 {
		rw.Version = new(uint64(nsVersion))
	}

	writeToMetaNs.ReadWrites = append(writeToMetaNs.ReadWrites, rw)

	tx := &applicationpb.Tx{
		Namespaces: []*applicationpb.TxNamespace{
			writeToMetaNs,
		},
	}

	return tx
}
