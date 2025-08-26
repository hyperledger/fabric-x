/*
Copyright IBM Corp. 2017 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/core/config/configtest"
	"github.com/hyperledger/fabric-x-common/internaltools/configtxgen/genesisconfig"
)

func TestConfigTxFlags(t *testing.T) {
	configTxDest := filepath.Join(t.TempDir(), "configtx")

	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	configtest.SetDevFabricConfigPath(t)
	devConfigDir := configtest.GetDevConfigDir()

	os.Args = []string{
		"cmd",
		"-channelID=testchannelid",
		"-outputCreateChannelTx=" + configTxDest,
		"-profile=" + genesisconfig.SampleSingleMSPChannelProfile,
		"-configPath=" + devConfigDir,
		"-inspectChannelCreateTx=" + configTxDest,
		"-asOrg=" + genesisconfig.SampleOrgName,
	}

	main()

	_, err := os.Stat(configTxDest)
	require.NoError(t, err, "Configtx file is written successfully")
}

func TestBlockFlags(t *testing.T) {
	blockDest := filepath.Join(t.TempDir(), "block")
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()
	os.Args = []string{
		"cmd",
		"-channelID=testchannelid",
		"-profile=" + genesisconfig.SampleSingleMSPSoloProfile,
		"-outputBlock=" + blockDest,
		"-inspectBlock=" + blockDest,
	}
	configtest.SetDevFabricConfigPath(t)

	main()

	_, err := os.Stat(blockDest)
	require.NoError(t, err, "Block file is written successfully")
}
