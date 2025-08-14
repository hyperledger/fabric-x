package configtxgen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric-lib-go/bccsp/factory"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/core/config/configtest"
	"github.com/hyperledger/fabric-x/tools/internaltools/configtxgen/genesisconfig"
)

func TestInspectMissing(t *testing.T) {
	require.EqualError(t, DoInspectBlock("NonSenseBlockFileThatDoesn'tActuallyExist"), "could not read block NonSenseBlockFileThatDoesn'tActuallyExist")
}

func TestInspectBlock(t *testing.T) {
	blockDest := filepath.Join(t.TempDir(), "block")

	config := genesisconfig.Load(genesisconfig.SampleAppChannelInsecureSoloProfile, configtest.GetDevConfigDir())

	require.NoError(t, DoOutputBlock(config, "foo", blockDest), "Good block generation request")
	require.NoError(t, DoInspectBlock(blockDest), "Good block inspection request")
}

func TestInspectBlockErr(t *testing.T) {
	config := genesisconfig.Load(genesisconfig.SampleAppChannelInsecureSoloProfile, configtest.GetDevConfigDir())

	require.EqualError(t, DoOutputBlock(config, "foo", ""), "error writing genesis block: open : no such file or directory")
	require.EqualError(t, DoInspectBlock(""), "could not read block ")
}

func TestMissingOrdererSection(t *testing.T) {
	blockDest := filepath.Join(t.TempDir(), "block")

	config := genesisconfig.Load(genesisconfig.SampleAppChannelInsecureSoloProfile, configtest.GetDevConfigDir())
	config.Orderer = nil

	require.EqualError(t, DoOutputBlock(config, "foo", blockDest), "refusing to generate block which is missing orderer section")
}

func TestApplicationChannelGenesisBlock(t *testing.T) {
	blockDest := filepath.Join(t.TempDir(), "block")

	config := genesisconfig.Load(genesisconfig.SampleAppChannelInsecureSoloProfile, configtest.GetDevConfigDir())

	require.NoError(t, DoOutputBlock(config, "foo", blockDest))
}

func TestApplicationChannelMissingApplicationSection(t *testing.T) {
	blockDest := filepath.Join(t.TempDir(), "block")

	config := genesisconfig.Load(genesisconfig.SampleAppChannelInsecureSoloProfile, configtest.GetDevConfigDir())
	config.Application = nil

	require.EqualError(t, DoOutputBlock(config, "foo", blockDest), "refusing to generate application channel block which is missing application section")
}

func TestMissingConsortiumValue(t *testing.T) {
	configTxDest := filepath.Join(t.TempDir(), "configtx")

	config := genesisconfig.Load(genesisconfig.SampleSingleMSPChannelProfile, configtest.GetDevConfigDir())
	config.Consortium = ""

	require.EqualError(t, DoOutputChannelCreateTx(config, nil, "foo", configTxDest), "config update generation failure: cannot define a new channel with no Consortium value")
}

func TestUnsuccessfulChannelTxFileCreation(t *testing.T) {
	configTxDest := filepath.Join(t.TempDir(), "configtx")

	config := genesisconfig.Load(genesisconfig.SampleSingleMSPChannelProfile, configtest.GetDevConfigDir())
	require.NoError(t, os.WriteFile(configTxDest, []byte{}, 0o440))
	require.EqualError(t, DoOutputChannelCreateTx(config, nil, "foo", configTxDest), fmt.Sprintf("error writing channel create tx: open %s: permission denied", configTxDest))
}

func TestMissingApplicationValue(t *testing.T) {
	configTxDest := filepath.Join(t.TempDir(), "configtx")

	config := genesisconfig.Load(genesisconfig.SampleSingleMSPChannelProfile, configtest.GetDevConfigDir())
	config.Application = nil

	require.EqualError(t, DoOutputChannelCreateTx(config, nil, "foo", configTxDest), "could not generate default config template: channel template configs must contain an application section")
}

func TestInspectMissingConfigTx(t *testing.T) {
	require.EqualError(t, DoInspectChannelCreateTx("ChannelCreateTxFileWhichDoesn'tReallyExist"), "could not read channel create tx: open ChannelCreateTxFileWhichDoesn'tReallyExist: no such file or directory")
}

func TestInspectConfigTx(t *testing.T) {
	configTxDest := filepath.Join(t.TempDir(), "configtx")

	config := genesisconfig.Load(genesisconfig.SampleSingleMSPChannelProfile, configtest.GetDevConfigDir())

	require.NoError(t, DoOutputChannelCreateTx(config, nil, "foo", configTxDest), "Good outputChannelCreateTx generation request")
	require.NoError(t, DoInspectChannelCreateTx(configTxDest), "Good configtx inspection request")
}

func TestPrintOrg(t *testing.T) {
	require.NoError(t, factory.InitFactories(nil))
	config := genesisconfig.LoadTopLevel(configtest.GetDevConfigDir())

	require.NoError(t, DoPrintOrg(config, genesisconfig.SampleOrgName), "Good org to print")

	err := DoPrintOrg(config, genesisconfig.SampleOrgName+".wrong")
	require.Error(t, err, "Bad org name")
	require.Regexp(t, "organization [^ ]* not found", err.Error())

	config.Organizations[0] = &genesisconfig.Organization{Name: "FakeOrg", ID: "FakeOrg"}
	err = DoPrintOrg(config, "FakeOrg")
	require.Error(t, err, "Fake org")
	require.Regexp(t, "bad org definition", err.Error())
}

func createBftOrdererConfig() *genesisconfig.Profile {
	// Load the BFT config from the sample, and use some TLS CA Cert as crypto material
	config := genesisconfig.Load(genesisconfig.SampleAppChannelSmartBftProfile, configtest.GetDevConfigDir())
	addTlsCertToConsenters(config)
	return config
}

func addTlsCertToConsenters(config *genesisconfig.Profile) {
	tlsCertPath := filepath.Join(configtest.GetDevConfigDir(), "msp", "tlscacerts", "tlsroot.pem")
	for _, consenter := range config.Orderer.ConsenterMapping {
		consenter.Identity = tlsCertPath
		consenter.ClientTLSCert = tlsCertPath
		consenter.ServerTLSCert = tlsCertPath
	}
}

func TestBftOrdererTypeWithoutV3CapabilitiesShouldRaiseAnError(t *testing.T) {
	// ### Arrange
	blockDest := filepath.Join(t.TempDir(), "block")
	config := createBftOrdererConfig()
	config.Capabilities["V3_0"] = false

	// ### Act & Assert
	require.EqualError(
		t,
		DoOutputBlock(config, "testChannelId", blockDest),
		"could not create bootstrapper: could not create channel group: "+
			"could not create orderer group: "+
			"orderer type BFT must be used with V3_0 channel capability: map[V3_0:false]",
	)
}

func TestBftOrdererTypeWithV3CapabilitiesShouldNotRaiseAnError(t *testing.T) {
	// ### Arrange
	blockDest := filepath.Join(t.TempDir(), "block")
	config := createBftOrdererConfig()
	config.Capabilities["V3_0"] = true

	// ### Act & Assert
	require.NoError(t, DoOutputBlock(config, "testChannelId", blockDest))
}

func TestFabricXGenesisBlock(t *testing.T) {
	blockDest := filepath.Join(t.TempDir(), "block")

	config := genesisconfig.Load(genesisconfig.SampleFabricX, configtest.GetDevConfigDir())
	addTlsCertToConsenters(config)
	keyPath := filepath.Join(configtest.GetDevConfigDir(), "msp", "signcerts", "peer.pem")
	config.Application.MetaNamespaceVerificationKeyPath = keyPath
	armaPath := filepath.Join(configtest.GetDevConfigDir(), "arma_shared_config.pbbin")
	config.Orderer.Arma.Path = armaPath

	require.NoError(t, DoOutputBlock(config, "foo", blockDest))
}
