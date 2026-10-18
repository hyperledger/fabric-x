//go:build pkcs11

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"cmp"

	"github.com/hyperledger/fabric-lib-go/bccsp/factory"
	"github.com/hyperledger/fabric-lib-go/bccsp/pkcs11"
)

// applyPKCS11Opts maps fxconfig PKCS#11 settings onto the Fabric factory options.
// When BCCSP.PKCS11.Library is set the factory is switched to the PKCS11 provider.
func applyPKCS11Opts(opts *factory.FactoryOpts, cfg BCCSPPKCS11Config) {
	if cfg.Library == "" {
		return
	}

	opts.Default = "PKCS11"
	opts.PKCS11 = &pkcs11.PKCS11Opts{
		Library:        cfg.Library,
		Label:          cfg.Label,
		Pin:            cfg.Pin,
		Hash:           cmp.Or(cfg.Hash, "SHA2"),
		Security:       cmp.Or(cfg.Security, 256),
		SoftwareVerify: cfg.SoftwareVerify,
		Immutable:      cfg.Immutable,
	}
}
