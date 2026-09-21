/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x-common/protoutil/identity"
	"github.com/hyperledger/fabric-x-common/tools/configtxgen"
	"github.com/hyperledger/fabric-x-orderer/common/tools/armageddon"
	"github.com/hyperledger/fabric-x-orderer/common/types"
	ordererutils "github.com/hyperledger/fabric-x-orderer/common/utils"
	ordererconfig "github.com/hyperledger/fabric-x-orderer/config"
	"github.com/hyperledger/fabric-x-orderer/config/generate"
	testutils "github.com/hyperledger/fabric-x-orderer/test/utils"
	"github.com/hyperledger/fabric-x-orderer/testutil"
	"github.com/hyperledger/fabric-x-orderer/testutil/client"
	"github.com/hyperledger/fabric-x-orderer/testutil/configutil"
	"github.com/hyperledger/fabric-x-orderer/testutil/signutil"
	"github.com/hyperledger/fabric-x-orderer/testutil/tx"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.yaml.in/yaml/v2"
)

const (
	defaultOrdererImage           = "hyperledger/arma-4p1s:main"
	defaultCommitterTestNodeImage = "hyperledger/committer-test-node:main"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// TestReconfigAppOrg is an end-to-end integration test that:
//  1. Starts a full orderer network and a committer sidecar via Docker Compose.
//  2. Sends an initial batch of transactions and verifies they are committed.
//  3. Submits a channel reconfiguration that adds a new peer organisation (peer-org-1).
//  4. Waits for all orderer nodes to restart with the new config sequence.
//  5. Verifies that the committer delivers the config block and all subsequent data blocks.
func TestReconfigAppOrg(t *testing.T) {
	artifactsDir, compose, baseCtx := setupTestEnv(t)

	configPath := filepath.Join(".", "ordererconfig", "arma_config.yaml")
	numOfParties, _ := networkStats(t, configPath)

	parties := make([]types.PartyID, 0, numOfParties)
	for i := 1; i <= numOfParties; i++ {
		parties = append(parties, types.PartyID(i))
	}

	f, err := os.Open(filepath.Join(artifactsDir, "config", "peer-org-0", "user_config.yaml"))
	require.NoError(t, err)
	defer f.Close()

	puc, err := armageddon.ReadUserConfig(&f)
	require.NoError(t, err)

	var totalTxNumber int
	totalTxNumber = sendInitialTxs(t, puc, totalTxNumber)

	sidecarPort := "4001"
	blockQueryClient := newCommitterBlockQueryClient(t, artifactsDir, sidecarPort, puc)

	pullRequestSigner, err := signutil.CreateSignerForUser(puc.MSPDir)
	require.NoError(t, err)

	pullOpts := newPullOpts(puc, parties, pullRequestSigner)

	testutils.PullFromAssemblers(t, pullOpts(totalTxNumber, nil))
	requireCommitterEnvelopes(t, blockQueryClient, totalTxNumber)

	configSeq, totalTxNumber := submitPeerOrgReconfig(t, artifactsDir, compose, baseCtx, pullOpts, totalTxNumber)

	verifyPostReconfigBlocks(t, blockQueryClient, totalTxNumber, configSeq)
}

// setupTestEnv creates a temporary artifacts directory, generates orderer and committer
// crypto/config material, starts the Docker Compose stack (arma + committer services),
// and waits until every party health endpoint and the committer sidecar are ready.
// It registers a t.Cleanup that tears down the stack and removes all temporary paths.
func setupTestEnv(t *testing.T) (string, tc.ComposeStack, context.Context) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)

	symlinkTarget := "/tmp/arma-all-in-one"
	artifactsDir := filepath.Join(tmpDir, "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0o755))
	createSymlink(t, artifactsDir, symlinkTarget)

	storageDir := filepath.Join(tmpDir, "storage")
	require.NoError(t, os.RemoveAll(storageDir), "Failed to clean storage dir")
	require.NoError(t, os.MkdirAll(storageDir, 0o755), "Failed to create storage dir")

	configPath := filepath.Join(".", "ordererconfig", "arma_config.yaml")
	numOfParties, numOfShards := networkStats(t, configPath)
	t.Logf("Running test with %d parties and %d shards", numOfParties, numOfShards)

	armageddon.NewCLI().Run([]string{"generate", "--config", configPath, "--output", symlinkTarget, "--sampleConfigPath", "./ordererconfig/configtx.yaml"})
	setupArtifacts(t, artifactsDir, symlinkTarget, numOfParties)

	baseCtx := context.Background()
	newDockerCompose := includeServices(t, []string{"arma", "committer"}, "./docker-compose.yaml")
	compose, err := tc.NewDockerCompose(newDockerCompose)
	require.NoError(t, err, "Failed to create Docker Compose instance")

	// Tear down any stale stack from a previous run before starting fresh.
	teardownStaleStack := func() {
		_ = compose.Down(baseCtx, tc.RemoveOrphans(true), tc.RemoveImagesLocal)
	}
	teardownStaleStack()
	t.Cleanup(func() {
		_ = compose.Down(baseCtx, tc.RemoveOrphans(true), tc.RemoveImagesLocal)
		_ = os.Remove(newDockerCompose)
		_ = os.Remove(symlinkTarget)
		_ = os.RemoveAll(tmpDir)
	})

	partyStrategies := make([]wait.Strategy, 0, numOfParties)
	for i := range numOfParties {
		base := 8022 + i*100
		partyStrategies = append(partyStrategies, waitForPartyHealthy(base, base+3))
	}

	err = compose.WithEnv(map[string]string{
		"ORDERER_IMAGE":   envOrDefault("ORDERER_IMAGE", defaultOrdererImage),
		"COMMITTER_IMAGE": envOrDefault("COMMITTER_IMAGE", defaultCommitterTestNodeImage),
		"ARTIFACTS_DIR":   artifactsDir,
		"STORAGE_DIR":     storageDir,
	}).
		WaitForService("arma", wait.ForAll(partyStrategies...)).
		WaitForService("committer", wait.ForLog("sidecar connected to coordinator at localhost:9001")).
		Up(baseCtx, tc.Wait(true))
	require.NoError(t, err, "Failed to start Docker Compose services")

	return artifactsDir, compose, baseCtx
}

// sendInitialTxs sends 10 signed data transactions to the orderer using a rate limiter
// and returns the updated running total of transactions sent.
func sendInitialTxs(t *testing.T, puc *armageddon.UserConfig, totalTxNumber int) int {
	t.Helper()

	fillInterval := 10 * time.Millisecond
	fillFrequency := 1000 / int(fillInterval.Milliseconds())
	rate := 500
	capacity := rate / fillFrequency
	rl, err := armageddon.NewRateLimiter(rate, fillInterval, capacity)
	require.NoError(t, err)

	broadcastClient := client.NewBroadcastTxClient(puc, 10*time.Second)
	signer, certBytes, err := signutil.LoadCryptoMaterialForSigner(puc.MSPDir)
	require.NoError(t, err)

	for i := range 10 {
		status := rl.GetToken()
		require.True(t, status, "failed to send tx %d", i+1)
		env := tx.PrepareSignedEnvelopeWithCertificateID(i, 64, []byte("sessionNumber"), signer, certBytes, "peer-org-0")
		err = broadcastClient.SendTx(env)
		require.NoError(t, err)
		totalTxNumber++
	}

	return totalTxNumber
}

// newPullOpts returns a factory function that builds a BlockPullerOptions value for pulling
// blocks from all assembler parties. The caller supplies the expected transaction count and
// an optional block handler; all other fields are fixed from puc, parties, and signer.
func newPullOpts(puc *armageddon.UserConfig, parties []types.PartyID, signer identity.SignerSerializer) func(int, testutils.TestBlockHandler) *testutils.BlockPullerOptions {
	statusUnknown := common.Status_UNKNOWN
	return func(txCount int, handler testutils.TestBlockHandler) *testutils.BlockPullerOptions {
		return &testutils.BlockPullerOptions{
			UserConfig:   puc,
			Status:       &statusUnknown,
			Parties:      parties,
			StartBlock:   0,
			EndBlock:     math.MaxUint64,
			Transactions: txCount,
			ErrString:    "cancelled pull from assembler: %d",
			Signer:       signer,
			Timeout:      120,
			BlockHandler: handler,
		}
	}
}

// newCommitterBlockQueryClient constructs a BlockQueryClient that connects to the committer
// sidecar at the given port, authenticating with the peer-org-0 TLS CA certificate.
func newCommitterBlockQueryClient(t *testing.T, artifactsDir, sidecarPort string, puc *armageddon.UserConfig) *BlockQueryClient {
	t.Helper()
	tlsCACertPath := filepath.Join(artifactsDir, "crypto/peerOrganizations/peer-org-0/msp/tlscacerts/tlspeer-org-0-CA-cert.pem")
	tlsCACert, err := ordererutils.ReadPem(tlsCACertPath)
	require.NoError(t, err)
	return NewBlockQueryClient(t, fmt.Sprintf("127.0.0.1:%s", sidecarPort), puc.TLSPrivateKey, puc.TLSCertificate, [][]byte{tlsCACert})
}

// requireCommitterEnvelopes pulls all blocks from the committer sidecar and asserts that the
// total number of envelopes equals totalTxNumber+1 (the +1 accounts for the genesis block).
func requireCommitterEnvelopes(t *testing.T, blockQueryClient *BlockQueryClient, totalTxNumber int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var envelopes []*common.Envelope
	for block := range blockQueryClient.PullBlocks(t, ctx) {
		t.Logf("Pulled block: %v", block.GetHeader().GetNumber())
		t.Logf("Block contains %d envelopes.\n", len(block.GetData().GetData()))
		for _, envBytes := range block.GetData().GetData() {
			envelope, err := protoutil.UnmarshalEnvelope(envBytes)
			require.NoError(t, err, "failed to unmarshal envelope from block %d", block.GetHeader().GetNumber())
			envelopes = append(envelopes, envelope)
		}
	}
	require.Equal(t, totalTxNumber+1, len(envelopes), "number of envelopes does not match number of transactions sent")
}

// submitPeerOrgReconfig extends the network config with peer-org-1, builds and submits a
// signed channel config-update transaction that adds the new peer organisation, then waits
// for all orderer nodes to relaunch with the new config sequence. It also pulls blocks from
// the assemblers to confirm the config block was committed and writes it to a temp file.
// Returns the config sequence number and the updated total transaction count.
func submitPeerOrgReconfig(t *testing.T, artifactsDir string, compose tc.ComposeStack, ctx context.Context, pullOpts func(int, testutils.TestBlockHandler) *testutils.BlockPullerOptions, totalTxNumber int) (configSeq uint64, updatedTxNumber int) {
	t.Helper()

	configPath := filepath.Join(".", "ordererconfig", "arma_config.yaml")
	configFileContent, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var network generate.Network
	require.NoError(t, yaml.Unmarshal(configFileContent, &network))

	// Extend the network configuration to add peer-org-1 and generate its crypto materials.
	network.Peers = append(network.Peers, "peer-org-1")
	testutil.ExtendConfigAndCrypto(&network, artifactsDir, true)

	peerOrg1MSP := filepath.Join(artifactsDir, "crypto", "peerOrganizations", "peer-org-1", "msp")
	caCerts, err := os.ReadFile(filepath.Join(peerOrg1MSP, "cacerts", "peer-org-1-CA-cert.pem"))
	require.NoError(t, err)
	tlsCaCerts, err := os.ReadFile(filepath.Join(peerOrg1MSP, "tlscacerts", "tlspeer-org-1-CA-cert.pem"))
	require.NoError(t, err)
	adminCerts, err := os.ReadFile(filepath.Join(peerOrg1MSP, "admincerts", "Admin@peer-org-1-cert.pem"))
	require.NoError(t, err)

	knowncertsDir := filepath.Join(peerOrg1MSP, "knowncerts")
	knownCertPaths, err := ordererutils.PemFilesFromDir(knowncertsDir)
	require.NoError(t, err)
	var knownCerts [][]byte
	for _, p := range knownCertPaths {
		cert, err := os.ReadFile(filepath.Join(knowncertsDir, p))
		require.NoError(t, err)
		knownCerts = append(knownCerts, cert)
	}

	configBlockPath := filepath.Join(artifactsDir, "bootstrap", "bootstrap.block")
	builder := configutil.NewConfigUpdateBuilder(t, artifactsDir, configBlockPath)
	builder.AddNewPeer(t, &configutil.PeerConfig{
		Name:       "peer-org-1",
		CACerts:    [][]byte{caCerts},
		TLSCACerts: [][]byte{tlsCaCerts},
		AdminCerts: [][]byte{adminCerts},
		KnownCerts: knownCerts,
	})

	uc, err := testutil.GetUserConfig(artifactsDir, types.PartyID(1))
	require.NoError(t, err)
	broadcastClient := client.NewBroadcastTxClient(uc, 10*time.Second)
	defer broadcastClient.Stop()

	env := configutil.CreateConfigTXSignedByOrgs(t, artifactsDir, configutil.ChannelGroupName.Application, []string{"peer-org-0"}, 1, builder.ConfigUpdatePBData(t))
	require.NotNil(t, env)
	require.NoError(t, broadcastClient.SendTx(env))
	totalTxNumber++
	configSeq = 1

	armaContainer, err := compose.ServiceContainer(ctx, "arma")
	require.NoError(t, err, "Failed to get arma service container")
	waitForNetworkRelaunch(t, ctx, armaContainer, network, configSeq, 30*time.Second, 2*time.Second)

	// Pull from assemblers to confirm all transactions (including the config tx) are committed,
	// and export the config block to a temp file for later verification.
	t.Logf("Get the config block %d from an assembler ledger and write it to a temp location", configSeq)
	configBlockStoreDir := t.TempDir()
	configBlockFile := filepath.Join(configBlockStoreDir, fmt.Sprintf("config_%d.block", configSeq))
	testutils.PullFromAssemblers(t, pullOpts(totalTxNumber, &exportConfigBlockToFile{configSeq: configSeq, path: configBlockFile}))
	t.Logf("Verify the config block %d file was created", configSeq)
	require.FileExists(t, configBlockFile, "Config block file should exist after pulling from assembler")
	t.Logf("Config block %d successfully written to: %s", configSeq, configBlockFile)

	return configSeq, totalTxNumber
}

// verifyPostReconfigBlocks pulls blocks from the committer sidecar after the reconfig, asserts
// that the config block for configSeq is present and committed, and checks the total envelope count.
func verifyPostReconfigBlocks(t *testing.T, blockQueryClient *BlockQueryClient, totalTxNumber int, configSeq uint64) {
	t.Helper()

	queryCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var envelopes []*common.Envelope
	var lastConfigBlock *common.Block
	for block := range blockQueryClient.PullBlocks(t, queryCtx) {
		t.Logf("Pulled block: %v", block.GetHeader().GetNumber())
		t.Logf("Block contains %d envelopes.\n", len(block.GetData().GetData()))
		if protoutil.IsConfigBlock(block) {
			if extractConfigEnvelopeSequence(t, block) == configSeq {
				lastConfigBlock = block
			}
		}
		for _, envelopeBytes := range block.GetData().GetData() {
			envelope, err := protoutil.UnmarshalEnvelope(envelopeBytes)
			require.NoError(t, err, "failed to unmarshal envelope from block %d", block.GetHeader().GetNumber())
			envelopes = append(envelopes, envelope)
		}
	}

	require.NotNil(t, lastConfigBlock, "last config block should not be nil")
	verifyConfigBlockCommitted(t, lastConfigBlock)
	require.Equal(t, totalTxNumber+1, len(envelopes), "number of envelopes does not match number of transactions sent")
}

// verifyConfigBlockCommitted asserts that the single envelope inside configBlock carries a
// COMMITTED transaction-filter status, confirming the config transaction was accepted.
func verifyConfigBlockCommitted(t *testing.T, configBlock *common.Block) {
	t.Helper()

	require.NotEmpty(t, configBlock.GetMetadata())
	require.Greater(t, len(configBlock.GetMetadata().GetMetadata()), int(common.BlockMetadataIndex_TRANSACTIONS_FILTER))
	txFilter := configBlock.GetMetadata().GetMetadata()[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	require.Len(t, configBlock.GetData().GetData(), 1)
	status := committerpb.Status(txFilter[0])
	statusName := committerpb.Status_name[int32(status)]
	t.Logf("Transaction status: %s (%d)", statusName, status)
	require.Equal(t, committerpb.Status_COMMITTED, status)
}

type exportConfigBlockToFile struct {
	configSeq uint64
	path      string
	writeLock sync.Mutex
}

// HandleBlock inspects each pulled block and, when the block is a config block whose
// sequence matches ec.configSeq, serialises it to ec.path under a write lock.
func (ec *exportConfigBlockToFile) HandleBlock(t *testing.T, block *common.Block) error {
	if protoutil.IsConfigBlock(block) {
		sequence := extractConfigEnvelopeSequence(t, block)
		if sequence == ec.configSeq {
			configBlock := &common.Block{Header: block.GetHeader(), Data: block.GetData(), Metadata: block.GetMetadata()}
			ec.writeLock.Lock()
			defer ec.writeLock.Unlock()
			err := configtxgen.WriteOutputBlock(configBlock, ec.path)
			require.NoError(t, err)
		}
	}
	return nil
}

// extractConfigEnvelopeSequence extracts and returns the config sequence number from the
// first envelope of a config block.
func extractConfigEnvelopeSequence(t *testing.T, block *common.Block) uint64 {
	t.Helper()
	env, err := protoutil.ExtractEnvelope(block, 0)
	require.NoError(t, err)
	payload, err := protoutil.UnmarshalPayload(env.Payload)
	require.NoError(t, err)
	configEnv, err := protoutil.UnmarshalConfigEnvelope(payload.Data)
	require.NoError(t, err)
	return configEnv.GetConfig().Sequence
}

// waitForPartyHealthy returns a composite wait strategy that polls /healthz on every port
// in the inclusive range [start, end], used to confirm all components of a party are ready.
func waitForPartyHealthy(start, end int) wait.Strategy {
	strategies := make([]wait.Strategy, 0, end-start+1)
	for port := start; port <= end; port++ {
		strategies = append(strategies, wait.ForHTTP("/healthz").WithPort(fmt.Sprintf("%d", port)))
	}
	return wait.ForAll(strategies...)
}

// waitForNetworkRelaunch polls the container logs until every Router, Consensus, Assembler,
// and Batcher component of every party has logged its "started with new config sequence"
// message, or until the timeout elapses.
func waitForNetworkRelaunch(t *testing.T, ctx context.Context, container *testcontainers.DockerContainer, network generate.Network, configSeq uint64, timeout, interval time.Duration) {
	t.Helper()

	labels := networkRelaunchLabels(network, strconv.FormatUint(configSeq, 10))

	require.Eventually(t, func() bool {
		rc, err := container.Logs(ctx)
		if err != nil {
			t.Logf("failed reading container logs: %v", err)
			return false
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Logf("failed reading container log bytes: %v", err)
			return false
		}

		found, err := scanLogsForLabels(bytes.NewReader(data), labels)
		if err != nil {
			t.Logf("failed scanning container logs: %v", err)
			return false
		}

		for _, label := range labels {
			if _, ok := found[label]; !ok {
				t.Logf("network relaunch marker not found yet: %s", label)
				return false
			}
			t.Logf("network relaunch marker found: %s", label)
		}

		return true
	}, timeout, interval, "Timed out waiting for network relaunch marker in container logs")
}

// scanLogsForLabels reads lines from r and returns the subset of labels that appear in at
// least one line, stopping early once every label has been found.
func scanLogsForLabels(r io.Reader, labels []string) (map[string]struct{}, error) {
	remaining := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		remaining[label] = struct{}{}
	}

	found := make(map[string]struct{}, len(labels))
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		for label := range remaining {
			if strings.Contains(line, label) {
				found[label] = struct{}{}
				delete(remaining, label)
			}
		}
		if len(remaining) == 0 {
			break
		}
	}

	return found, scanner.Err()
}

// networkRelaunchLabels builds the log-line substrings that every orderer component emits
// after restarting with the given config sequence, one entry per (party, component) pair.
func networkRelaunchLabels(network generate.Network, sequence string) []string {
	labels := []string{}
	for _, party := range network.Parties {
		for _, c := range []struct{ name, ep string }{
			{"Router", party.RouterEndpoint},
			{"Consensus", party.ConsenterEndpoint},
			{"Assembler", party.AssemblerEndpoint},
		} {
			labels = append(labels, fmt.Sprintf("%s started with new config sequence %s, listening on %s", c.name, sequence, strings.Replace(c.ep, "0.0.0.0", "[::]", 1)))
		}
		for _, ep := range party.BatchersEndpoints {
			labels = append(labels, fmt.Sprintf("Batcher started with new config sequence %s, listening on %s", sequence, strings.Replace(ep, "0.0.0.0", "[::]", 1)))
		}
	}
	return labels
}

// setupArtifacts copies committer TLS credentials and patches each party's local config
// files (operations port, storage path) so the Docker Compose services can find them.
func setupArtifacts(t *testing.T, artifactsDir, dir string, numOfParties int) {
	t.Helper()
	setupCommitterArtifacts(t, artifactsDir)
	patchAllPartyLocalConfigs(t, dir, numOfParties)
}

// includeServices reads a Docker Compose YAML file, removes all service entries that are
// not listed in services, writes the result to a temp file, and returns that file's path.
func includeServices(t *testing.T, services []string, fileName string) string {
	t.Helper()
	content, err := os.ReadFile(fileName)
	require.NoError(t, err, "Failed to read docker-compose.yml")
	var config map[any]any
	err = yaml.Unmarshal(content, &config)
	require.NoError(t, err, "Failed to unmarshal docker-compose.yml")

	configuredServices := config["services"].(map[any]any)
	for service := range configuredServices {
		if !slices.Contains(services, fmt.Sprintf("%v", service)) {
			delete(configuredServices, service)
		}
	}

	tempFile, err := os.CreateTemp(filepath.Dir(fileName), "docker-compose-*.yml")
	require.NoError(t, err, "Failed to create temp docker-compose.yml")
	t.Cleanup(func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	})
	err = yaml.NewEncoder(tempFile).Encode(config)
	require.NoError(t, err, "Failed to write temp docker-compose.yml")
	return tempFile.Name()
}

// patchAllPartyLocalConfigs patches the local config files for all parties in dir.
func patchAllPartyLocalConfigs(t *testing.T, dir string, numOfParties int) {
	t.Helper()
	for i := 1; i <= numOfParties; i++ {
		offset := uint32((i - 1) * 100)
		partyDir := filepath.Join(dir, "config", fmt.Sprintf("party%d", i))
		storagePath := fmt.Sprintf("/storage/party%d", i)
		patchLocalConfig(t, filepath.Join(partyDir, "local_config_router.yaml"), 8022+offset, storagePath+"/router")
		patchLocalConfig(t, filepath.Join(partyDir, "local_config_assembler.yaml"), 8023+offset, storagePath+"/assembler")
		patchLocalConfig(t, filepath.Join(partyDir, "local_config_batcher1.yaml"), 8024+offset, storagePath+"/batcher")
		patchLocalConfig(t, filepath.Join(partyDir, "local_config_consenter.yaml"), 8025+offset, storagePath+"/consenter")
	}
}

// patchLocalConfig reads the node local config at path, updates the operations listen port
// and address, the file-store path, and (if present) the consensus WAL directory, then
// writes the modified config back to the same file.
func patchLocalConfig(t *testing.T, path string, operationsPort uint32, storagePath string) {
	t.Helper()
	var cfg ordererconfig.NodeLocalConfig
	require.NoError(t, ordererutils.ReadFromYAML(&cfg, path), "read %s", path)
	require.NotNil(t, cfg.GeneralConfig, "GeneralConfig is nil in %s", path)
	require.NotNil(t, cfg.OperationsConfig, "OperationsConfig is nil in %s", path)
	cfg.OperationsConfig.ListenPort = operationsPort
	cfg.OperationsConfig.ListenAddress = "0.0.0.0"
	require.NotNil(t, cfg.FileStore, "FileStore is nil in %s", path)
	cfg.FileStore.Path = storagePath
	if cfg.ConsensusParams != nil {
		cfg.ConsensusParams.WALDir = filepath.Join(storagePath, "wal")
	}
	require.NoError(t, ordererutils.WriteToYAML(cfg, path), "write %s", path)
}

// createSymlink removes any existing file/symlink at destination and creates a new symlink
// pointing from destination to source.
func createSymlink(t *testing.T, source, destination string) {
	t.Helper()
	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		require.Failf(t, "Failed to remove destination", "%v", err)
	}

	err := os.Symlink(source, destination)
	require.NoError(t, err)
}

// networkStats parses the network config YAML at networkConfigPath and returns the number
// of parties and the number of shards (batcher endpoints per party).
func networkStats(t *testing.T, networkConfigPath string) (numParties int, numShards int) {
	t.Helper()
	var network generate.Network
	require.NoError(t, ordererutils.ReadFromYAML(&network, networkConfigPath), "read %s", networkConfigPath)
	return len(network.Parties), len(network.Parties[0].BatchersEndpoints)
}

var peerOrg0Peers = []string{
	"sidecar.peer-org-0",
	"vc.peer-org-0",
	"verifier.peer-org-0",
	"query.peer-org-0",
	"coordinator.peer-org-0",
}

type ordererOrgEntry struct{ src, dst, cert string }

var ordererOrgs = []ordererOrgEntry{
	{"org1", "orderer-org-1", "tlsorg1-CA-cert.pem"},
	{"org2", "orderer-org-2", "tlsorg2-CA-cert.pem"},
	{"org3", "orderer-org-3", "tlsorg3-CA-cert.pem"},
	{"org4", "orderer-org-4", "tlsorg4-CA-cert.pem"},
}

// buildCommitterDirs returns the list of directories that must exist under artifactsDir
// for the committer sidecar to locate its TLS credentials and peer/orderer org MSPs.
func buildCommitterDirs(artifactsDir string) []string {
	dirs := []string{
		filepath.Join(artifactsDir, "peerOrganizations/peer-org-0/msp/tlscacerts"),
	}
	for _, peer := range peerOrg0Peers {
		dirs = append(dirs, filepath.Join(artifactsDir, "peerOrganizations/peer-org-0/peers", peer, "tls"))
	}
	for _, o := range ordererOrgs {
		dirs = append(dirs, filepath.Join(artifactsDir, "ordererOrganizations", o.dst, "msp/tlscacerts"))
	}
	return dirs
}

// copyPeerTLSCreds copies server.crt and server.key from srcDir into the tls/ subdirectory
// of each named peer under dstBase.
func copyPeerTLSCreds(t *testing.T, srcDir, dstBase string, peers []string) {
	t.Helper()
	for _, peer := range peers {
		dstDir := filepath.Join(dstBase, peer, "tls")
		require.NoError(t, armageddon.CopyFile(filepath.Join(srcDir, "server.crt"), filepath.Join(dstDir, "server.crt")))
		require.NoError(t, armageddon.CopyFile(filepath.Join(srcDir, "server.key"), filepath.Join(dstDir, "server.key")))
	}
}

// setupCommitterArtifacts creates the committer directory layout under artifactsDir and
// populates it by copying peer TLS credentials, orderer-org TLS CA certs (with the naming
// convention expected by the committer), the peer-org-0 TLS CA cert, the bootstrap config
// block (as config-block.pb.bin), and the peer-org-0 client MSP.
func setupCommitterArtifacts(t *testing.T, artifactsDir string) {
	t.Helper()

	for _, dir := range buildCommitterDirs(artifactsDir) {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}

	copyPeerTLSCreds(t,
		filepath.Join(artifactsDir, "crypto/peerOrganizations/peer-org-0/peers/peer-org-0/tls"),
		filepath.Join(artifactsDir, "peerOrganizations/peer-org-0/peers"),
		peerOrg0Peers,
	)

	for _, o := range ordererOrgs {
		src := filepath.Join(artifactsDir, "crypto/ordererOrganizations", o.src, "msp/tlscacerts", o.cert)
		dst := filepath.Join(artifactsDir, "ordererOrganizations", o.dst, "msp/tlscacerts", "tlsca."+o.dst+"-cert.pem")
		require.NoError(t, armageddon.CopyFile(src, dst))
	}

	require.NoError(t, armageddon.CopyFile(
		filepath.Join(artifactsDir, "crypto/peerOrganizations/peer-org-0/msp/tlscacerts/tlspeer-org-0-CA-cert.pem"),
		filepath.Join(artifactsDir, "peerOrganizations/peer-org-0/msp/tlscacerts/tlsca.peer-org-0-cert.pem"),
	))

	// config-block.pb.bin: sidecar uses this to bootstrap the orderer connection.
	// armageddon generate places the bootstrap block at bootstrap/bootstrap.block.
	require.NoError(t, armageddon.CopyFile(
		filepath.Join(artifactsDir, "bootstrap/bootstrap.block"),
		filepath.Join(artifactsDir, "config-block.pb.bin"),
	))

	require.NoError(t, copyDir(
		filepath.Join(artifactsDir, "crypto/peerOrganizations/peer-org-0/users/client@peer-org-0/msp"),
		filepath.Join(artifactsDir, "peerOrganizations/peer-org-0/users/client@peer-org-0/msp"),
	))
}

// copyDir recursively copies the directory tree at src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return armageddon.CopyFile(path, target)
	})
}
