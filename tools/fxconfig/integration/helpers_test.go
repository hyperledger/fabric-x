/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/client"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"

	cli "github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1"
)

const (
	ordererPort      = "7050"
	sidecarPort      = "4001"
	queryServicePort = "7001"
	channelID        = "mychannel"
	policy           = "OR('Org1MSP.member')"
)

// setup spawns a committer test container and returns a map containing the endpoints of the committers services.
func setup(t *testing.T) map[string]string {
	t.Helper()

	absPath, err := filepath.Abs(filepath.Join(".", "testdata", "crypto", "config-block.pb.bin"))
	require.NoError(t, err)

	dataDirectory, err := filepath.Abs(filepath.Join(".", "testdata", "crypto"))
	require.NoError(t, err)

	ctx := t.Context()
	committerContainer, err := testcontainers.Run(
		ctx, "ghcr.io/hyperledger/fabric-x-committer-test-node:0.1.8",
		testcontainers.WithCmd("run", "db", "orderer", "committer", "--insecure"),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      absPath,
			ContainerFilePath: "/root/material/config-block.pb.bin",
			FileMode:          0o700,
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      dataDirectory,
			ContainerFilePath: "/root/material/",
			FileMode:          0o755,
		}),
		testcontainers.WithExposedPorts(ordererPort, sidecarPort, queryServicePort),
		testcontainers.WithEnv(map[string]string{
			"SC_COORDINATOR_LOGGING_LEVEL":        "DEBUG",
			"SC_SIDECAR_LOGGING_LEVEL":            "DEBUG",
			"SC_SIDECAR_ORDERER_CHANNEL_ID":       channelID,
			"SC_SIDECAR_ORDERER_TLS_MODE":         "none",
			"SC_SIDECAR_ORDERER_SIGNED_ENVELOPES": "true",
			"SC_QUERY_SERVICE_SERVER_ENDPOINT":    fmt.Sprintf(":%v", queryServicePort),
			"SC_QUERY_SERVICE_LOGGING_LEVEL":      "DEBUG",
			"SC_ORDERER_BLOCK_SIZE":               "1",
			"SC_ORDERER_LOGGING_LEVEL":            "DEBUG",
			"SC_VC_LOGGING_LEVEL":                 "DEBUG",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort(ordererPort),
			wait.ForListeningPort(sidecarPort),
			wait.ForListeningPort(queryServicePort),
			wait.ForLog("Setting the last committed block number:"),
		),
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

	return endpoints
}

func fxconfig(tb testing.TB, args ...string) (string, error) {
	tb.Helper()

	var stdOut bytes.Buffer

	rootCmd := cli.NewRootCommand(&cli.CLIContext{}, func(cfg *config.Config) (app.Application, error) {
		vctx := validation.NewValidationContext()
		return &app.AdminApp{
			Validators:      vctx,
			MspProvider:     &msp.SignerProvider{ValidationContext: vctx, Cfg: cfg.MSP},
			QueryProvider:   &client.QueryProvider{ValidationContext: vctx, Cfg: cfg.Queries},
			OrdererProvider: &client.OrdererProvider{ValidationContext: vctx, Cfg: cfg.Orderer},
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
