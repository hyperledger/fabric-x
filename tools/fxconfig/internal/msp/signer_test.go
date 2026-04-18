/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// testdataMSPDir returns the path to the local MSP testdata directory.
// Go sets the working directory to the package directory during tests.
func testdataMSPDir() string {
	return "testdata/msp"
}

func TestGetSignerIdentityFromMSP(t *testing.T) {
	t.Parallel()

	t.Run("success with valid MSP directory", func(t *testing.T) {
		t.Parallel()

		cfg := config.MSPConfig{
			LocalMspID: "Org1MSP",
			ConfigPath: testdataMSPDir(),
		}

		sid, err := GetSignerIdentityFromMSP(cfg)

		require.NoError(t, err)
		require.NotNil(t, sid)
		require.Equal(t, "Org1MSP", sid.GetMSPIdentifier())
	})

	t.Run("error with empty config path", func(t *testing.T) {
		t.Parallel()

		cfg := config.MSPConfig{
			LocalMspID: "Org1MSP",
			ConfigPath: "",
		}

		_, err := GetSignerIdentityFromMSP(cfg)

		require.Error(t, err)
		require.Contains(t, err.Error(), "msp setup error")
	})

	t.Run("error with non-existent config path", func(t *testing.T) {
		t.Parallel()

		cfg := config.MSPConfig{
			LocalMspID: "Org1MSP",
			ConfigPath: "/does/not/exist",
		}

		_, err := GetSignerIdentityFromMSP(cfg)

		require.Error(t, err)
		require.Contains(t, err.Error(), "msp setup error")
	})

	t.Run("signer can sign data", func(t *testing.T) {
		t.Parallel()

		cfg := config.MSPConfig{
			LocalMspID: "Org1MSP",
			ConfigPath: testdataMSPDir(),
		}

		sid, err := GetSignerIdentityFromMSP(cfg)
		require.NoError(t, err)

		sig, err := sid.Sign([]byte("test message"))
		require.NoError(t, err)
		require.NotEmpty(t, sig)
	})

	t.Run("success with explicit bccsp config", func(t *testing.T) {
		t.Parallel()

		cfg := config.MSPConfig{
			LocalMspID: "Org1MSP",
			ConfigPath: testdataMSPDir(),
			BCCSP: config.BCCSPConfig{
				Default: "SW",
				SW: config.BCCSPSWConfig{
					Security: 256,
					Hash:     "SHA2",
					FileKeyStore: config.BCCSPFileKeyStoreConfig{
						KeyStorePath: filepath.Join(testdataMSPDir(), "keystore"),
					},
				},
			},
		}

		sid, err := GetSignerIdentityFromMSP(cfg)

		require.NoError(t, err)
		require.NotNil(t, sid)
		require.Equal(t, "Org1MSP", sid.GetMSPIdentifier())
	})

	t.Run("success with default bccsp config", func(t *testing.T) {
		t.Parallel()

		cfg := config.MSPConfig{
			LocalMspID: "Org1MSP",
			ConfigPath: testdataMSPDir(),
		}

		sid, err := GetSignerIdentityFromMSP(cfg)

		require.NoError(t, err)
		require.NotNil(t, sid)
		require.Equal(t, "Org1MSP", sid.GetMSPIdentifier())

		sig, err := sid.Sign([]byte("test with defaults"))
		require.NoError(t, err)
		require.NotEmpty(t, sig)
	})

	t.Run("signer can serialize identity", func(t *testing.T) {
		t.Parallel()

		cfg := config.MSPConfig{
			LocalMspID: "Org1MSP",
			ConfigPath: testdataMSPDir(),
		}

		sid, err := GetSignerIdentityFromMSP(cfg)
		require.NoError(t, err)

		serialized, err := sid.Serialize()
		require.NoError(t, err)
		require.NotEmpty(t, serialized)
	})
}
