/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protoutil_test

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/protoutil"
)

func TestNewConfigGroup(t *testing.T) {
	require.Equal(t,
		&common.ConfigGroup{
			Groups:   make(map[string]*common.ConfigGroup),
			Values:   make(map[string]*common.ConfigValue),
			Policies: make(map[string]*common.ConfigPolicy),
		},
		protoutil.NewConfigGroup(),
	)
}
