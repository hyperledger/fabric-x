/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package msp provides MSP (Membership Service Provider) signing identity management.
package msp

import (
	"fmt"
	"path"

	"github.com/hyperledger/fabric-lib-go/bccsp"
	"github.com/hyperledger/fabric-lib-go/bccsp/pkcs11"
	"github.com/hyperledger/fabric-lib-go/bccsp/sw"

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

// setupMSP creates an MSP instance.
// Selects PKCS#11 BCCSP when mspCfg.BCCSP.PKCS11.Library is set; otherwise falls back to
// the default file-based software BCCSP that reads keys from <ConfigPath>/keystore.
//
//nolint:ireturn
func setupMSP(mspCfg config.MSPConfig) (msp.MSP, error) {
	conf, err := msp.GetLocalMspConfig(mspCfg.ConfigPath, nil, mspCfg.LocalMspID)
	if err != nil {
		return nil, fmt.Errorf("error getting local msp config from %v: %w", mspCfg.ConfigPath, err)
	}

	cp, err := buildBCCSP(mspCfg)
	if err != nil {
		return nil, fmt.Errorf("error building BCCSP: %w", err)
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

	if err := thisMSP.Setup(conf); err != nil {
		return nil, err
	}

	return thisMSP, nil
}

// buildBCCSP constructs a BCCSP provider based on the MSP configuration.
// PKCS11 mode activates when mspCfg.BCCSP.PKCS11.Library is non-empty.
//
//nolint:ireturn
func buildBCCSP(mspCfg config.MSPConfig) (bccsp.BCCSP, error) {
	p := mspCfg.BCCSP.PKCS11
	if p.Library != "" {
		return newPKCS11BCCSP(p)
	}
	return newFileBasedBCCSP(mspCfg.ConfigPath)
}

//nolint:ireturn
func newFileBasedBCCSP(mspConfigPath string) (bccsp.BCCSP, error) {
	dir := path.Join(mspConfigPath, "keystore")
	ks, err := sw.NewFileBasedKeyStore(nil, dir, true)
	if err != nil {
		return nil, err
	}
	return sw.NewDefaultSecurityLevelWithKeystore(ks)
}

//nolint:ireturn
func newPKCS11BCCSP(cfg config.PKCS11Config) (bccsp.BCCSP, error) {
	hash := cfg.Hash
	if hash == "" {
		hash = "SHA2"
	}
	security := cfg.Security
	if security == 0 {
		security = 256
	}

	opts := pkcs11.PKCS11Opts{
		Security:       security,
		Hash:           hash,
		Library:        cfg.Library,
		Label:          cfg.Label,
		Pin:            cfg.Pin,
		SoftwareVerify: cfg.SoftwareVerify,
		Immutable:      cfg.Immutable,
	}
	return pkcs11.New(opts, &dummyKeyStore{})
}

// dummyKeyStore is a no-op keystore used with PKCS11.
// PKCS11 manages keys inside the HSM/KMS; no file-based storage is needed.
type dummyKeyStore struct{}

func (ks *dummyKeyStore) ReadOnly() bool { return true }

func (ks *dummyKeyStore) GetKey(ski []byte) (bccsp.Key, error) {
	return nil, fmt.Errorf("not implemented - keys are managed by PKCS11")
}

func (ks *dummyKeyStore) StoreKey(k bccsp.Key) error {
	return fmt.Errorf("not implemented - keys are managed by PKCS11")
}
