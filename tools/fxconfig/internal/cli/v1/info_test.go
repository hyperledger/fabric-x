/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

func TestNewInfoCommand(t *testing.T) {
	t.Parallel()

	cmd := NewInfoCommand(&CLIContext{})

	require.NotNil(t, cmd)
	require.Equal(t, "info", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.RunE)
}

func TestInfoCommand_PrintsYAMLConfig(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config:  &config.Config{},
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.NotEmpty(t, outBuf.String())
}

func TestInfoCommand_NilConfigPrintsNull(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config:  nil,
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.Contains(t, outBuf.String(), "null")
}

func TestInfoCommand_PrintsEnvConfig(t *testing.T) {
	t.Parallel()

	boolPtr := func(b bool) *bool { return &b }

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config: &config.Config{
			Logging: config.LoggingConfig{
				Level:  "ERROR",
				Format: "%{color}%{level}%{color:reset} %{message}",
			},
			MSP: config.MSPConfig{
				LocalMspID: "Org1MSP",
				ConfigPath: "/path/to/msp",
			},
			TLS: config.TLSConfig{
				Enabled:        boolPtr(true),
				ClientKeyPath:  "/path/to/client.key",
				ClientCertPath: "/path/to/client.crt",
				RootCertPaths:  []string{"/path/to/ca.crt"},
			},
			Orderer: config.OrdererConfig{
				EndpointServiceConfig: config.EndpointServiceConfig{
					Address:           "localhost:7050",
					ConnectionTimeout: 30 * time.Second,
					TLS: &config.TLSConfig{
						Enabled:        boolPtr(true),
						RootCertPaths:  []string{"/path/to/orderer-ca.crt"},
						ClientCertPath: "/path/to/orderer-client.crt",
						ClientKeyPath:  "/path/to/orderer-client.key",
					},
				},
				Channel: "mychannel",
			},
			Queries: config.QueriesConfig{
				EndpointServiceConfig: config.EndpointServiceConfig{
					Address:           "localhost:7001",
					ConnectionTimeout: 30 * time.Second,
					TLS: &config.TLSConfig{
						Enabled:       boolPtr(true),
						RootCertPaths: []string{"/path/to/peer-ca.crt"},
					},
				},
			},
			Notifications: config.NotificationsConfig{
				EndpointServiceConfig: config.EndpointServiceConfig{
					Address:           "localhost:7001",
					ConnectionTimeout: 30 * time.Second,
					TLS: &config.TLSConfig{
						Enabled: boolPtr(false),
					},
				},
				WaitingTimeout: 30 * time.Second,
			},
		},
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	err := cmd.Flags().Set("format", "env")
	require.NoError(t, err)

	err = cmd.RunE(cmd, nil)
	require.NoError(t, err)

	output := outBuf.String()
	require.Contains(t, output, "FXCONFIG_LOGGING_LEVEL=ERROR")
	require.Contains(t, output, "FXCONFIG_MSP_LOCALMSPID=Org1MSP")
	require.Contains(t, output, "FXCONFIG_MSP_CONFIGPATH=/path/to/msp")
	require.Contains(t, output, "FXCONFIG_TLS_ENABLED=true")
	require.Contains(t, output, "FXCONFIG_TLS_CLIENTKEY=/path/to/client.key")
	require.Contains(t, output, "FXCONFIG_TLS_CLIENTCERT=/path/to/client.crt")
	require.Contains(t, output, "FXCONFIG_TLS_ROOTCERTS=/path/to/ca.crt")
	require.Contains(t, output, "FXCONFIG_ORDERER_ADDRESS=localhost:7050")
	require.Contains(t, output, "FXCONFIG_ORDERER_CHANNEL=mychannel")
	require.Contains(t, output, "FXCONFIG_QUERIES_ADDRESS=localhost:7001")
	require.Contains(t, output, "FXCONFIG_NOTIFICATIONS_ADDRESS=localhost:7001")
}

func TestInfoCommand_InvalidFormat(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config:  &config.Config{},
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	err := cmd.Flags().Set("format", "json")
	require.NoError(t, err)

	err = cmd.RunE(cmd, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid --format: json (want yaml|env)")
}
