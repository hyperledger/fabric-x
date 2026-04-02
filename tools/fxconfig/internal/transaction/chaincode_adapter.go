/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// ChaincodeWrite represents a chaincode-style write operation.
type ChaincodeWrite struct {
	Key     string
	Value   []byte
	Version *uint64
}

// ChaincodeNamespaceRWSet represents chaincode read-write output for one namespace.
type ChaincodeNamespaceRWSet struct {
	Namespace string
	Version   uint64
	Writes    []ChaincodeWrite
}

// CreateTxNamespaceFromChaincodeRWSet converts one chaincode namespace rwset into a Fabric-X TxNamespace.
func CreateTxNamespaceFromChaincodeRWSet(rwset ChaincodeNamespaceRWSet) (*applicationpb.TxNamespace, error) {
	if rwset.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	ns := &applicationpb.TxNamespace{
		NsId:       rwset.Namespace,
		NsVersion:  rwset.Version,
		ReadWrites: make([]*applicationpb.ReadWrite, 0, len(rwset.Writes)),
	}

	for _, w := range rwset.Writes {
		if w.Key == "" {
			return nil, fmt.Errorf("write key is required")
		}

		rw := &applicationpb.ReadWrite{
			Key:   []byte(w.Key),
			Value: w.Value,
		}
		if w.Version != nil {
			rw.Version = applicationpb.NewVersion(*w.Version)
		}

		ns.ReadWrites = append(ns.ReadWrites, rw)
	}

	return ns, nil
}

// CreateTxFromChaincodeRWSets converts multiple chaincode namespace rwsets into a Fabric-X transaction.
func CreateTxFromChaincodeRWSets(rwsets []ChaincodeNamespaceRWSet) (*applicationpb.Tx, error) {
	if len(rwsets) == 0 {
		return nil, fmt.Errorf("at least one namespace rwset is required")
	}

	namespaces := make([]*applicationpb.TxNamespace, 0, len(rwsets))
	for _, rwset := range rwsets {
		ns, err := CreateTxNamespaceFromChaincodeRWSet(rwset)
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns)
	}

	return &applicationpb.Tx{Namespaces: namespaces}, nil
}
