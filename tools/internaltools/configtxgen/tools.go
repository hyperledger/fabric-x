package configtxgen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-config/protolator"
	"github.com/hyperledger/fabric-config/protolator/protoext/ordererext"
	"github.com/hyperledger/fabric-config/protolator/protoext/peerext"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/pkg/errors"

	"github.com/hyperledger/fabric-x/tools/common/util"
	"github.com/hyperledger/fabric-x/tools/internaltools/configtxgen/encoder"
	"github.com/hyperledger/fabric-x/tools/internaltools/configtxgen/genesisconfig"
	"github.com/hyperledger/fabric-x/tools/protoutil"
)

var logger = util.MustGetLogger("common.tools.configtxgen")

func GetOutputBlock(config *genesisconfig.Profile, channelID string) (*cb.Block, error) {
	pgen, err := encoder.NewBootstrapper(config)
	if err != nil {
		return nil, errors.WithMessage(err, "could not create bootstrapper")
	}
	logger.Info("Generating genesis block")
	if config.Orderer == nil {
		return nil, errors.New("refusing to generate block which is missing orderer section")
	}
	if config.Consortiums != nil {
		logger.Error("Warning: 'Consortiums' should be nil since system channel is no longer supported in Fabric v3.x")
	} else {
		if config.Application == nil {
			return nil, errors.New("refusing to generate application channel block which is missing application section")
		}
		logger.Info("Creating application channel genesis block")
	}
	genesisBlock := pgen.GenesisBlockForChannel(channelID)
	return genesisBlock, nil
}

func WriteOutputBlock(block *cb.Block, outputBlock string) error {
	err := writeFile(outputBlock, protoutil.MarshalOrPanic(block), 0o640)
	if err != nil {
		return fmt.Errorf("error writing genesis block: %s", err)
	}
	return nil
}

func DoOutputBlock(config *genesisconfig.Profile, channelID string, outputBlock string) error {
	genesisBlock, err := GetOutputBlock(config, channelID)
	if err != nil {
		return err
	}
	logger.Info("Writing genesis block")
	return WriteOutputBlock(genesisBlock, outputBlock)
}

func DoOutputChannelCreateTx(conf, baseProfile *genesisconfig.Profile, channelID string, outputChannelCreateTx string) error {
	logger.Info("Generating new channel configtx")

	var configtx *common.Envelope
	var err error
	if baseProfile == nil {
		configtx, err = encoder.MakeChannelCreationTransaction(channelID, nil, conf)
	} else {
		configtx, err = encoder.MakeChannelCreationTransactionWithSystemChannelContext(channelID, nil, conf, baseProfile)
	}
	if err != nil {
		return err
	}

	logger.Info("Writing new channel tx")
	err = writeFile(outputChannelCreateTx, protoutil.MarshalOrPanic(configtx), 0o640)
	if err != nil {
		return fmt.Errorf("error writing channel create tx: %s", err)
	}
	return nil
}

func ReadBlock(blockPath string) (*cb.Block, error) {
	data, err := os.ReadFile(blockPath)
	if err != nil {
		return nil, fmt.Errorf("could not read block %s", blockPath)
	}
	block, err := protoutil.UnmarshalBlock(data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling to block: %s", err)
	}
	return block, nil
}

func DoInspectBlock(inspectBlock string) error {
	logger.Info("Inspecting block")
	block, err := ReadBlock(inspectBlock)
	if err != nil {
		return err
	}
	err = protolator.DeepMarshalJSON(os.Stdout, block)
	if err != nil {
		return fmt.Errorf("malformed block contents: %s", err)
	}
	return nil
}

func DoInspectChannelCreateTx(inspectChannelCreateTx string) error {
	logger.Info("Inspecting transaction")
	data, err := os.ReadFile(inspectChannelCreateTx)
	if err != nil {
		return fmt.Errorf("could not read channel create tx: %s", err)
	}

	logger.Info("Parsing transaction")
	env, err := protoutil.UnmarshalEnvelope(data)
	if err != nil {
		return fmt.Errorf("Error unmarshalling envelope: %s", err)
	}

	err = protolator.DeepMarshalJSON(os.Stdout, env)
	if err != nil {
		return fmt.Errorf("malformed transaction contents: %s", err)
	}

	return nil
}

func DoPrintOrg(t *genesisconfig.TopLevel, printOrg string) error {
	for _, org := range t.Organizations {
		if org.Name == printOrg {
			if len(org.OrdererEndpoints) > 0 {
				// An Orderer OrgGroup
				channelCapabilities := t.Capabilities["Channel"]
				og, err := encoder.NewOrdererOrgGroup(org, channelCapabilities)
				if err != nil {
					return errors.Wrapf(err, "bad org definition for org %s", org.Name)
				}

				if err := protolator.DeepMarshalJSON(os.Stdout, &ordererext.DynamicOrdererOrgGroup{ConfigGroup: og}); err != nil {
					return errors.Wrapf(err, "malformed org definition for org: %s", org.Name)
				}
				return nil
			}

			// Otherwise assume it is an Application OrgGroup, where the encoder is not strict whether anchor peers exist or not
			ag, err := encoder.NewApplicationOrgGroup(org)
			if err != nil {
				return errors.Wrapf(err, "bad org definition for org %s", org.Name)
			}
			if err := protolator.DeepMarshalJSON(os.Stdout, &peerext.DynamicApplicationOrgGroup{ConfigGroup: ag}); err != nil {
				return errors.Wrapf(err, "malformed org definition for org: %s", org.Name)
			}
			return nil
		}
	}
	return errors.Errorf("organization %s not found", printOrg)
}

func writeFile(filename string, data []byte, perm os.FileMode) error {
	dirPath := filepath.Dir(filename)
	exists, err := dirExists(dirPath)
	if err != nil {
		return err
	}
	if !exists {
		err = os.MkdirAll(dirPath, 0o750)
		if err != nil {
			return err
		}
	}
	return os.WriteFile(filename, data, perm)
}

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
