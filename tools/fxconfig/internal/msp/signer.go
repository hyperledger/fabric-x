/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package msp provides MSP (Membership Service Provider) signing identity management.
package msp

import (
	"fmt"

	"github.com/hyperledger/fabric-lib-go/bccsp/factory"

	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// GetSignerIdentityFromMSP returns the default signing identity from MSP configuration.
//
//nolint:ireturn
func GetSignerIdentityFromMSP(cfg config.MSPConfig) (msp.SigningIdentity, error) {
	thisMSP, err := setupMSP(cfg)
	if err != nil {
		return nil, fmt.Errorf("msp setup error: %w", err)
	}

	sid, err := thisMSP.GetDefaultSigningIdentity()
	if err != nil {
		return nil, fmt.Errorf("get signer identity error: %w", err)
	}

	return sid, nil
}

// setupMSP creates an MSP instance with file-based BCCSP keystore from the given configuration.
//
//nolint:ireturn
func setupMSP(mspCfg config.MSPConfig) (msp.MSP, error) {
	bccspOpts := mspCfg.ToFactoryOpts()

	conf, err := msp.GetLocalMspConfig(mspCfg.ConfigPath, bccspOpts, mspCfg.LocalMspID)
	if err != nil {
		return nil, fmt.Errorf("error getting local msp config from %v: %w", mspCfg.ConfigPath, err)
	}

	cp, err := factory.GetBCCSPFromOpts(bccspOpts)
	if err != nil {
		return nil, err
	}

	mspOpts := &msp.BCCSPNewOpts{
		NewBaseOpts: msp.NewBaseOpts{
			Version: msp.MSPv1_0,
		},
	}

	thisMSP, err := msp.New(mspOpts, cp)
	if err != nil {
		return nil, err
	}

	err = thisMSP.Setup(conf)
	if err != nil {
		return nil, err
	}

	return thisMSP, nil
}
