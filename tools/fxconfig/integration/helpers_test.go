/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hyperledger/fabric-x-common/common/crypto/tlsgen"
	fmsp "github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	cli "github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/client"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

const (
	ordererPort      = "7050"
	sidecarPort      = "4001"
	queryServicePort = "7001"
	channelID        = "mychannel"
)

// tlsModes lists the TLS configurations under which all integration tests run.
var tlsModes = []string{"none", "tls"}

// tlsCredentials holds TLS certificate paths generated for a test run.
// When non-nil, fxconfig clients are configured with TLS using the CA cert.
type tlsCredentials struct {
	caCertPath     string                          // host-side CA cert for fxconfig rootCerts
	containerFiles []testcontainers.ContainerFile  // cert files to copy into the container
}

// generateTLSCredentials creates an ephemeral CA, generates server and client
// key pairs, writes them to disk, and returns paths and container file entries
// that match the committer test-node's expected layout (/server-certs/, /client-certs/).
func generateTLSCredentials(t *testing.T) *tlsCredentials {
	t.Helper()

	ca, err := tlsgen.NewCA()
	require.NoError(t, err)

	serverKP, err := ca.NewServerCertKeyPair("localhost", "127.0.0.1", "::1")
	require.NoError(t, err)

	clientKP, err := ca.NewClientCertKeyPair()
	require.NoError(t, err)

	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, data, 0o600))
		return p
	}

	serverCert := write("server-cert.pem", serverKP.Cert)
	serverKey := write("server-key.pem", serverKP.Key)
	clientCert := write("client-cert.pem", clientKP.Cert)
	clientKey := write("client-key.pem", clientKP.Key)
	caCert := write("ca-cert.pem", ca.CertBytes())

	return &tlsCredentials{
		caCertPath: caCert,
		containerFiles: []testcontainers.ContainerFile{
			{HostFilePath: serverCert, ContainerFilePath: "/server-certs/public-key.pem", FileMode: 0o600},
			{HostFilePath: serverKey, ContainerFilePath: "/server-certs/private-key.pem", FileMode: 0o600},
			{HostFilePath: caCert, ContainerFilePath: "/server-certs/ca-certificate.pem", FileMode: 0o600},
			{HostFilePath: clientCert, ContainerFilePath: "/client-certs/public-key.pem", FileMode: 0o600},
			{HostFilePath: clientKey, ContainerFilePath: "/client-certs/private-key.pem", FileMode: 0o600},
			{HostFilePath: caCert, ContainerFilePath: "/client-certs/ca-certificate.pem", FileMode: 0o600},
		},
	}
}

// setupSingleOrgAdmin spawns a committer test container with a single org admin lifecycle policy
// and returns a map containing the endpoints of the committers services.
func setupSingleOrgAdmin(t *testing.T, tlsMode string) (map[string]string, *tlsCredentials) {
	t.Helper()

	genesisPath, err := filepath.Abs(filepath.Join(".", "testdata", "crypto", "single-org.pb.bin"))
	require.NoError(t, err)

	return setup(t, genesisPath, tlsMode)
}

// setupMultiOrgAdmin spawns a committer test container with a multi org admin lifecycle policy
// and returns a map containing the endpoints of the committers services.
func setupMultiOrgAdmin(t *testing.T, tlsMode string) (map[string]string, *tlsCredentials) {
	t.Helper()

	genesisPath, err := filepath.Abs(filepath.Join(".", "testdata", "crypto", "multi-org.pb.bin"))
	require.NoError(t, err)

	return setup(t, genesisPath, tlsMode)
}

func setup(t *testing.T, genesisPath, tlsMode string) (map[string]string, *tlsCredentials) {
	t.Helper()

	dataDirectory, err := filepath.Abs(filepath.Join(".", "testdata", "crypto"))
	require.NoError(t, err)

	// msp configuration for sidecar orderer client
	mspID := "Org1MSP"
	mspDir := "/root/artifacts/crypto/peerOrganizations/Org1/users/committer@org1.com/msp"

	cmd := []string{"run", "db", "orderer", "committer"}
	env := map[string]string{
		"SC_COORDINATOR_LOGGING_LOGSPEC":      "DEBUG",
		"SC_SIDECAR_LOGGING_LOGSPEC":          "DEBUG",
		"SC_SIDECAR_ORDERER_CHANNEL_ID":       channelID,
		"SC_SIDECAR_ORDERER_SIGNED_ENVELOPES": "true",
		"SC_SIDECAR_ORDERER_IDENTITY_MSP_ID":  mspID,
		"SC_SIDECAR_ORDERER_IDENTITY_MSP_DIR": mspDir,
		"SC_QUERY_SERVICE_SERVER_ENDPOINT":    fmt.Sprintf(":%v", queryServicePort),
		"SC_QUERY_SERVICE_LOGGING_LOGSPEC":    "DEBUG",
		"SC_ORDERER_BLOCK_SIZE":               "1",
		"SC_ORDERER_LOGGING_LOGSPEC":          "DEBUG",
		"SC_VC_LOGGING_LOGSPEC":               "DEBUG",
	}

	var creds *tlsCredentials

	if tlsMode == "none" {
		cmd = append(cmd, "--insecure")
		env["SC_SIDECAR_ORDERER_TLS_MODE"] = "none"
	} else {
		creds = generateTLSCredentials(t)
		// Set the TLS mode for every committer service and inter-service client,
		// mirroring the env vars from the committer's own Docker test suite.
		for _, key := range []string{
			"SC_COORDINATOR_SERVER_TLS_MODE",
			"SC_COORDINATOR_VERIFIER_TLS_MODE",
			"SC_COORDINATOR_VALIDATOR_COMMITTER_TLS_MODE",
			"SC_COORDINATOR_MONITORING_TLS_MODE",
			"SC_QUERY_SERVER_TLS_MODE",
			"SC_QUERY_MONITORING_TLS_MODE",
			"SC_SIDECAR_SERVER_TLS_MODE",
			"SC_SIDECAR_MONITORING_TLS_MODE",
			"SC_SIDECAR_COMMITTER_TLS_MODE",
			"SC_VC_SERVER_TLS_MODE",
			"SC_VC_MONITORING_TLS_MODE",
			"SC_VERIFIER_SERVER_TLS_MODE",
			"SC_VERIFIER_MONITORING_TLS_MODE",
			"SC_SIDECAR_ORDERER_TLS_MODE",
			"SC_SIDECAR_ORDERER_CONNECTION_TLS_MODE",
			"SC_ORDERER_SERVER_TLS_MODE",
		} {
			env[key] = tlsMode
		}
	}

	ctx := t.Context()
	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithCmd(cmd...),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      genesisPath,
			ContainerFilePath: "/root/artifacts/config-block.pb.bin",
			FileMode:          0o700,
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      dataDirectory,
			ContainerFilePath: "/root/artifacts/",
			FileMode:          0o755,
		}),
		testcontainers.WithExposedPorts(ordererPort, sidecarPort, queryServicePort),
		testcontainers.WithEnv(env),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort(ordererPort),
			wait.ForListeningPort(sidecarPort),
			wait.ForListeningPort(queryServicePort),
			wait.ForLog("Setting the last committed block number:"),
		),
	}

	if creds != nil {
		opts = append(opts, testcontainers.WithFiles(creds.containerFiles...))
	}

	committerContainer, err := testcontainers.Run(
		ctx, "ghcr.io/hyperledger/fabric-x-committer-test-node:0.1.9",
		opts...,
	)
	t.Cleanup(func() {
		testcontainers.CleanupContainer(t, committerContainer)
	})
	require.NoError(t, err)

	endpoints := make(map[string]string)
	endpoints["query"], err = committerContainer.PortEndpoint(ctx, queryServicePort, "")
	require.NoError(t, err)

	endpoints["orderer"], err = committerContainer.PortEndpoint(ctx, ordererPort, "")
	require.NoError(t, err)

	endpoints["sidecar"], err = committerContainer.PortEndpoint(ctx, sidecarPort, "")
	require.NoError(t, err)

	return endpoints, creds
}

func generateConfigFile(
	tb testing.TB,
	localMspID string,
	mspConfigPath string,
	endpoints map[string]string,
	tlsCreds *tlsCredentials,
) string {
	tb.Helper()
	tmpDir := tb.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
msp:
  localMspID: ` + localMspID + `
  configPath: ` + mspConfigPath + `

orderer:
  address: ` + endpoints["orderer"] + `
  channel: ` + channelID + `
  connectionTimeout: 30s

queries:
  address: ` + endpoints["query"] + `
  connectionTimeout: 20s

notifications:
  address: ` + endpoints["sidecar"] + `
  connectionTimeout: 15s
  waitingTimeout: 15s
`
	if tlsCreds != nil {
		configContent += `
tls:
  enabled: true
  rootCerts:
    - ` + tlsCreds.caCertPath + `
`
	}

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(tb, err)
	return configPath
}

func fxconfig(tb testing.TB, args ...string) (string, error) {
	tb.Helper()

	var stdOut bytes.Buffer

	rootCmd := cli.NewRootCommand(&cli.CLIContext{}, func(cfg *config.Config) (app.Application, error) {
		vctx := validation.NewValidationContext()
		return &app.AdminApp{
			Validators: vctx,
			MspProvider: provider.New[fmsp.SigningIdentity, *config.MSPConfig](
				func(cfg *config.MSPConfig) (fmsp.SigningIdentity, error) {
					return msp.GetSignerIdentityFromMSP(*cfg)
				},
				&cfg.MSP,
				vctx,
			),
			QueryProvider: provider.New[adapters.QueryClient, *config.QueriesConfig](
				func(cfg *config.QueriesConfig) (adapters.QueryClient, error) {
					return client.NewQueryClient(*cfg)
				},
				&cfg.Queries,
				vctx,
			),
			OrdererProvider: provider.New[adapters.OrdererClient, *config.OrdererConfig](
				func(cfg *config.OrdererConfig) (adapters.OrdererClient, error) {
					return client.NewOrdererClient(*cfg)
				},
				&cfg.Orderer,
				vctx,
			),
			NotificationProvider: provider.New[adapters.NotificationClient, *config.NotificationsConfig](
				func(cfg *config.NotificationsConfig) (adapters.NotificationClient, error) {
					return client.NewNotificationClient(*cfg)
				},
				&cfg.Notifications,
				vctx,
			),
		}, nil
	})

	rootCmd.SetContext(tb.Context())
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&stdOut)

	tb.Logf("fxconfig %v", args)
	err := rootCmd.Execute()
	if err != nil {
		return "", err
	}

	out := stdOut.String()
	tb.Logf("> %v", out)

	return out, nil
}

type Namespace struct {
	Name    string
	Version int
}

// parseNamespaceList parses the output of 'fxconfig namespace list' command.
// Expected format: "N) name: version X policy: <hex>".
// Example: "0) perf: version 0 policy: 0a05454344534112b201...".
func parseNamespaceList(output string) ([]Namespace, error) { //nolint:gocognit
	namespaces := make([]Namespace, 0)

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		// Skip header, empty lines, and error messages
		if line == "" ||
			strings.HasPrefix(line, "Installed namespaces") ||
			strings.HasPrefix(line, "Error:") ||
			strings.HasPrefix(line, "Usage:") ||
			strings.HasPrefix(line, "Flags:") {
			continue
		}

		// Parse line format: "0) perf: version 0 policy: ..."
		if idx := strings.Index(line, ")"); idx > 0 {
			rest := strings.TrimSpace(line[idx+1:])

			// Split by "version" keyword
			parts := strings.Split(rest, " version ")
			if len(parts) != 2 {
				continue
			}

			// Extract name (before ":")
			namePart := strings.TrimSpace(parts[0])
			if colonIdx := strings.Index(namePart, ":"); colonIdx > 0 {
				name := strings.TrimSpace(namePart[:colonIdx])

				// Extract version (ignore policy)
				versionPart := strings.TrimSpace(parts[1])
				versionPolicyParts := strings.Split(versionPart, " policy: ")

				version := 0
				_, err := fmt.Sscanf(versionPolicyParts[0], "%d", &version)
				if err != nil {
					return nil, fmt.Errorf("failed to parse version from line '%s': %w", line, err)
				}

				namespaces = append(namespaces, Namespace{
					Name:    name,
					Version: version,
				})
			}
		}
	}

	return namespaces, nil
}
