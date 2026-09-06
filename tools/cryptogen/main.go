/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/alecthomas/kingpin/v2"

	"github.com/hyperledger/fabric-x-common/common/metadata"
	"github.com/hyperledger/fabric-x-common/sampleconfig"
	"github.com/hyperledger/fabric-x-common/tools/cryptogen"
)

const programName = "cryptogen"

var (
	commitSHA = metadata.CommitSHA
	version   = metadata.Version
)

// command line flags
var (
	app = kingpin.New("cryptogen", "Utility for generating Hyperledger Fabric key material")

	gen           = app.Command("generate", "Generate key material")
	outputDir     = gen.Flag("output", "The output directory in which to place artifacts").Default("crypto-config").String()
	genConfigFile = gen.Flag("config", "The configuration template to use").File()
	showtemplate  = app.Command("showtemplate", "Show the default configuration template")

	versionCmd    = app.Command("version", "Show version information")
	ext           = app.Command("extend", "Extend existing network")
	inputDir      = ext.Flag("input", "The input directory in which existing network place").Default("crypto-config").String()
	extConfigFile = ext.Flag("config", "The configuration template to use").File()
)

func main() {
	kingpin.Version("0.0.1")
	var err error
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch cmd {
	case gen.FullCommand():
		err = generate()
	case ext.FullCommand():
		err = extend()
	case showtemplate.FullCommand():
		if _, err = fmt.Print(sampleconfig.DefaultCryptoConfig); err != nil {
			err = fmt.Errorf("failed to write template: %w", err)
		}
	case versionCmd.FullCommand():
		if _, err = fmt.Println(getVersionInfo()); err != nil {
			err = fmt.Errorf("failed to write version info: %w", err)
		}
	default:
		panic("programming error")
	}

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error executing command %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func extend() error {
	config, err := getConfig()
	if err != nil {
		return err
	}
	return cryptogen.Extend(*inputDir, config)
}

func generate() error {
	config, err := getConfig()
	if err != nil {
		return err
	}
	return cryptogen.Generate(*outputDir, config)
}

func getConfig() (*cryptogen.Config, error) {
	var configData string
	switch {
	case *genConfigFile != nil:
		data, err := io.ReadAll(*genConfigFile)
		if err != nil {
			return nil, fmt.Errorf("error reading configuration: %w", err)
		}
		configData = string(data)
	case *extConfigFile != nil:
		data, err := io.ReadAll(*extConfigFile)
		if err != nil {
			return nil, fmt.Errorf("error reading configuration: %w", err)
		}
		configData = string(data)
	default:
		configData = sampleconfig.DefaultCryptoConfig
	}
	cfg, err := cryptogen.ParseConfig(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

func getVersionInfo() string {
	return fmt.Sprintf(
		"%s:\n Version: %s\n Commit SHA: %s\n Go version: %s\n OS/Arch: %s",
		programName,
		version,
		commitSHA,
		runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	)
}
