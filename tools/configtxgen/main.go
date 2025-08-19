/*
Copyright IBM Corp. 2017 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hyperledger/fabric-lib-go/bccsp/factory"

	"github.com/hyperledger/fabric-x-common/common/util"
	"github.com/hyperledger/fabric-x-common/internaltools/configtxgen"
	"github.com/hyperledger/fabric-x-common/internaltools/configtxgen/genesisconfig"
	"github.com/hyperledger/fabric-x-common/internaltools/configtxgen/metadata"
)

var logger = util.MustGetLogger("common.tools.configtxgen")

func main() {
	var outputBlock, outputChannelCreateTx, channelCreateTxBaseProfile, profile, configPath, channelID, inspectBlock, inspectChannelCreateTx, asOrg, printOrg string

	flag.StringVar(&outputBlock, "outputBlock", "", "The path to write the genesis block to (if set)")
	flag.StringVar(&channelID, "channelID", "", "The channel ID to use in the configtx")
	flag.StringVar(&outputChannelCreateTx, "outputCreateChannelTx", "", "[DEPRECATED] The path to write a channel creation configtx to (if set)")
	flag.StringVar(&channelCreateTxBaseProfile, "channelCreateTxBaseProfile", "", "[DEPRECATED] Specifies a profile to consider as the orderer system channel current state to allow modification of non-application parameters during channel create tx generation. Only valid in conjunction with 'outputCreateChannelTx'.")
	flag.StringVar(&profile, "profile", "", "The profile from configtx.yaml to use for generation.")
	flag.StringVar(&configPath, "configPath", "", "The path containing the configuration to use (if set)")
	flag.StringVar(&inspectBlock, "inspectBlock", "", "Prints the configuration contained in the block at the specified path")
	flag.StringVar(&inspectChannelCreateTx, "inspectChannelCreateTx", "", "[DEPRECATED] Prints the configuration contained in the transaction at the specified path")
	flag.StringVar(&asOrg, "asOrg", "", "Performs the config generation as a particular organization (by name), only including values in the write set that org (likely) has privilege to set")
	flag.StringVar(&printOrg, "printOrg", "", "Prints the definition of an organization as JSON. (useful for adding an org to a channel manually)")

	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if channelID == "" && (outputBlock != "" || outputChannelCreateTx != "") {
		logger.Fatalf("Missing channelID, please specify it with '-channelID'")
	}

	// show version
	if *version {
		printVersion()
		os.Exit(0)
	}

	// don't need to panic when running via command line
	defer func() {
		if err := recover(); err != nil {
			if strings.Contains(fmt.Sprint(err), "Error reading configuration: Unsupported Config Type") {
				logger.Error("Could not find configtx.yaml. " +
					"Please make sure that FABRIC_CFG_PATH or -configPath is set to a path " +
					"which contains configtx.yaml")
				os.Exit(1)
			}
			if strings.Contains(fmt.Sprint(err), "Could not find profile") {
				logger.Error(fmt.Sprint(err) + ". " +
					"Please make sure that FABRIC_CFG_PATH or -configPath is set to a path " +
					"which contains configtx.yaml with the specified profile")
				os.Exit(1)
			}
			logger.Panic(err)
		}
	}()

	logger.Info("Loading configuration")
	err := factory.InitFactories(nil)
	if err != nil {
		logger.Fatalf("Error on initFactories: %s", err)
	}
	var profileConfig *genesisconfig.Profile
	if outputBlock != "" || outputChannelCreateTx != "" {
		if profile == "" {
			logger.Fatalf("The '-profile' is required when '-outputBlock', '-outputChannelCreateTx' is specified")
		}

		if configPath != "" {
			profileConfig = genesisconfig.Load(profile, configPath)
		} else {
			profileConfig = genesisconfig.Load(profile)
		}
	}

	var baseProfile *genesisconfig.Profile
	if channelCreateTxBaseProfile != "" {
		if outputChannelCreateTx == "" {
			logger.Warning("Specified 'channelCreateTxBaseProfile', but did not specify 'outputChannelCreateTx', 'channelCreateTxBaseProfile' will not affect output.")
		}
		if configPath != "" {
			baseProfile = genesisconfig.Load(channelCreateTxBaseProfile, configPath)
		} else {
			baseProfile = genesisconfig.Load(channelCreateTxBaseProfile)
		}
	}

	if outputBlock != "" {
		if err := configtxgen.DoOutputBlock(profileConfig, channelID, outputBlock); err != nil {
			logger.Fatalf("Error on outputBlock: %s", err)
		}
	}

	if outputChannelCreateTx != "" {
		if err := configtxgen.DoOutputChannelCreateTx(profileConfig, baseProfile, channelID, outputChannelCreateTx); err != nil {
			logger.Fatalf("Error on outputChannelCreateTx: %s", err)
		}
	}

	if inspectBlock != "" {
		if err := configtxgen.DoInspectBlock(inspectBlock); err != nil {
			logger.Fatalf("Error on inspectBlock: %s", err)
		}
	}

	if inspectChannelCreateTx != "" {
		if err := configtxgen.DoInspectChannelCreateTx(inspectChannelCreateTx); err != nil {
			logger.Fatalf("Error on inspectChannelCreateTx: %s", err)
		}
	}

	if printOrg != "" {
		var topLevelConfig *genesisconfig.TopLevel
		if configPath != "" {
			topLevelConfig = genesisconfig.LoadTopLevel(configPath)
		} else {
			topLevelConfig = genesisconfig.LoadTopLevel()
		}

		if err := configtxgen.DoPrintOrg(topLevelConfig, printOrg); err != nil {
			logger.Fatalf("Error on printOrg: %s", err)
		}
	}
}

func printVersion() {
	fmt.Println(metadata.GetVersionInfo())
}
