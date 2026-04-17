/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package msp provides MSP (Membership Service Provider) signing identity management.
package msp

import (
	"context"
	"fmt"
	"path"

	"github.com/hyperledger/fabric-lib-go/bccsp/sw"

	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/audit"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// GetSignerIdentityFromMSP returns the default signing identity from MSP configuration.
//
//nolint:ireturn
func GetSignerIdentityFromMSP(cfg config.MSPConfig) (msp.SigningIdentity, error) {
	auditLogger := audit.MustGetAuditLogger(nil)

	auditLogger.IdentityLoadStarted(context.Background(), audit.IdentityLoadStartedEvent{
		EventMeta:  audit.NewEventMeta(),
		MspID:      cfg.LocalMspID,
		ConfigPath: cfg.ConfigPath,
	})

	thisMSP, err := setupMSP(cfg)
	if err != nil {
		auditLogger.IdentityLoaded(context.Background(), audit.IdentityLoadedEvent{
			EventMeta:  audit.NewEventMeta(),
			MspID:      cfg.LocalMspID,
			ConfigPath: cfg.ConfigPath,
			Result:     "failure",
			ErrorMsg:   err.Error(),
		})
		return nil, fmt.Errorf("msp setup error: %w", err)
	}

	sid, err := thisMSP.GetDefaultSigningIdentity()
	if err != nil {
		auditLogger.IdentityLoaded(context.Background(), audit.IdentityLoadedEvent{
			EventMeta:  audit.NewEventMeta(),
			MspID:      cfg.LocalMspID,
			ConfigPath: cfg.ConfigPath,
			Result:     "failure",
			ErrorMsg:   err.Error(),
		})
		return nil, fmt.Errorf("get signer identity error: %w", err)
	}

	// Extract cert subject for audit
	certSubject := extractCertSubject(sid)

	auditLogger.IdentityLoaded(context.Background(), audit.IdentityLoadedEvent{
		EventMeta:   audit.NewEventMeta(),
		MspID:       cfg.LocalMspID,
		ConfigPath:  cfg.ConfigPath,
		CertSubject: certSubject,
		Result:      "success",
	})

	return sid, nil
}

func extractCertSubject(sid msp.SigningIdentity) string {
	// Try to extract certificate subject from signing identity
	// This is a best-effort attempt
	return "" // Placeholder - would need to deserialize and parse cert
}

// setupMSP creates an MSP instance with file-based BCCSP keystore from the given configuration.
//
//nolint:ireturn
func setupMSP(mspCfg config.MSPConfig) (msp.MSP, error) {
	conf, err := msp.GetLocalMspConfig(mspCfg.ConfigPath, nil, mspCfg.LocalMspID)
	if err != nil {
		return nil, fmt.Errorf("error getting local msp config from %v: %w", mspCfg.ConfigPath, err)
	}

	// TODO: get proper BCCSP connfiguration via config

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
