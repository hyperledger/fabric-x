//go:build fabricx

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	common "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fxdlog"
)

func NewSDK(registry services.Registry) *SDK {
	// return &SDK{SDK: fts.NewFrom(fsc.NewFrom(fdlog.NewSDK(registry)))}
	return &SDK{SDK: fxdlog.NewSDK(registry)}
}

type SDK struct {
	common.SDK
}

func (p *SDK) Install() error {
	return p.SDK.Install()
}
