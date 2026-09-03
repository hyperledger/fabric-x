//go:build !pkcs11

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger/fabric-lib-go/bccsp/factory"
)

// applyPKCS11Opts is a no-op when fxconfig is built without the pkcs11 tag.
// PKCS#11 support is opt-in and requires building with -tags pkcs11.
func applyPKCS11Opts(_ *factory.FactoryOpts, _ BCCSPPKCS11Config) {}
