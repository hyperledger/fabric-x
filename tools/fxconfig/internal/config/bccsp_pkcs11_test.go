//go:build pkcs11

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMSPConfigToFactoryOpts_PKCS11(t *testing.T) {
	t.Parallel()

	mspCfg := MSPConfig{
		ConfigPath: "/tmp/msp",
		BCCSP: BCCSPConfig{
			PKCS11: BCCSPPKCS11Config{
				Library:        "/usr/local/lib/libsofthsm2.so",
				Label:          "TestLabel",
				Pin:            "1234",
				Hash:           "SHA2",
				Security:       256,
				SoftwareVerify: true,
			},
		},
	}

	opts := mspCfg.ToFactoryOpts()

	require.Equal(t, "PKCS11", opts.Default)
	require.NotNil(t, opts.PKCS11)
	require.Equal(t, "/usr/local/lib/libsofthsm2.so", opts.PKCS11.Library)
	require.Equal(t, "TestLabel", opts.PKCS11.Label)
	require.Equal(t, "1234", opts.PKCS11.Pin)
	require.Equal(t, "SHA2", opts.PKCS11.Hash)
	require.Equal(t, 256, opts.PKCS11.Security)
	require.True(t, opts.PKCS11.SoftwareVerify)
}

func TestMSPConfigToFactoryOpts_PKCS11_DefaultsWhenLibrarySet(t *testing.T) {
	t.Parallel()

	mspCfg := MSPConfig{
		ConfigPath: "/tmp/msp",
		BCCSP: BCCSPConfig{
			PKCS11: BCCSPPKCS11Config{
				Library: "/usr/local/lib/libsofthsm2.so",
				Label:   "TestLabel",
				Pin:     "1234",
			},
		},
	}

	opts := mspCfg.ToFactoryOpts()

	require.Equal(t, "PKCS11", opts.Default)
	require.NotNil(t, opts.PKCS11)
	require.Equal(t, "SHA2", opts.PKCS11.Hash)
	require.Equal(t, 256, opts.PKCS11.Security)
}

func TestMSPConfigToFactoryOpts_PKCS11_NotActivatedWithoutLibrary(t *testing.T) {
	t.Parallel()

	mspCfg := MSPConfig{
		ConfigPath: "/tmp/msp",
		BCCSP: BCCSPConfig{
			PKCS11: BCCSPPKCS11Config{
				Label: "only-label-without-library",
			},
		},
	}

	opts := mspCfg.ToFactoryOpts()

	require.Equal(t, "SW", opts.Default)
	require.Nil(t, opts.PKCS11)
}
