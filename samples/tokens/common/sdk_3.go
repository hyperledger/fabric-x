//go:build !fabricx

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"go.uber.org/dig"
)

func NewSDK(registry services.Registry) *SDK {
	return &SDK{
		tokensdk.NewSDK(registry),
	}
}

type SDK struct {
	*tokensdk.SDK
}

func (p *SDK) Install() error {
	if err := errors.Join(
		p.Container().Provide(fabric.NewGenericDriver, dig.Group("network-drivers")),
		p.Container().Provide(dlog.NewDriver, dig.Group("token-drivers")),
	); err != nil {
		return err
	}
	return p.SDK.Install()
}
