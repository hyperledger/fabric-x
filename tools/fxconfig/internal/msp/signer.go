/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package msp provides MSP (Membership Service Provider) signing identity management.
package msp

import (
	"fmt"
	"path"

	"github.com/hyperledger/fabric-lib-go/bccsp/sw"

	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// SignerProvider manages MSP signing identity retrieval with validation support.
type SignerProvider struct {
	ValidationContext validation.Context
	Cfg               config.MSPConfig
}

// Validate checks the MSP configuration validity.
func (f *SignerProvider) Validate() error {
	return f.Cfg.Validate(f.ValidationContext)
}

// Get retrieves the signing identity from the configured MSP.
func (f *SignerProvider) Get() (msp.SigningIdentity, error) {
	return GetSignerIdentityFromMSP(f.Cfg)
}

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
	conf, err := msp.GetLocalMspConfig(mspCfg.ConfigPath, nil, mspCfg.LocalMspID)
	if err != nil {
		return nil, fmt.Errorf("error getting local msp config from %v: %w", mspCfg.ConfigPath, err)
	}

	// TODO get proper BCCSP connfiguration via config

	dir := path.Join(mspCfg.ConfigPath, "keystore")
	ks, err := sw.NewFileBasedKeyStore(nil, dir, true)
	if err != nil {
		return nil, err
	}

	cp, err := sw.NewDefaultSecurityLevelWithKeystore(ks)
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

// TODO we keep this for later when we come back for the MSP-based endorsement implementation.
// func getSignerID(signer msp.SigningIdentity) (*msppb.Identity, error) {
//  if signer == nil {
//		return nil, errors.Get("nil signer")
//	}
//
//	signerCert, err := signer.GetCertificatePEM()
//	if err != nil {
//		return nil, err
//	}
//	return msppb.NewIdentity(signer.GetIdentifier().Mspid, signerCert), nil
// }
