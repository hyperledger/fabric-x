/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// Note that the interesting test cases are covered in the transaction package.

func TestMergeTransactions_TooFew(t *testing.T) {
	t.Parallel()

	a := &AdminApp{}

	_, err := a.MergeTransactions(t.Context(), []*applicationpb.Tx{{}})
	require.Error(t, err)
}

func TestMergeTransactions_Empty(t *testing.T) {
	t.Parallel()

	a := &AdminApp{}

	_, err := a.MergeTransactions(t.Context(), nil)
	require.Error(t, err)
}
