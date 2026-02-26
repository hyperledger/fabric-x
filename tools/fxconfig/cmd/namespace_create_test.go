/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

func TestCreateCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		deployFunc    deployF
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing namespace ID argument",
			args:          []string{"namespace", "create"},
			deployFunc:    fakeDeploySuccess,
			expectError:   true,
			errorContains: "accepts 1 arg(s), received 0",
		},
		{
			name: "successful creation with all flags",
			args: []string{
				"namespace", "create",
				"1",
				"--policy", "threshold:/tmp/some/path/pk.pem",
			},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
		{
			name: "deploy function returns error",
			args: []string{
				"namespace", "create",
				"2",
				"--policy", "threshold:/tmp/some/path/pk.pem",
			},
			deployFunc:    fakeDeployError,
			expectError:   true,
			errorContains: "deployment failed",
		},
		{
			name:        "help flag displays usage",
			args:        []string{"namespace", "create", "--help"},
			deployFunc:  fakeDeploySuccess,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup with mock config
			rootCmd := setupCreateCommandWithConfig(t, tt.deployFunc)

			var outBuf, errBuf bytes.Buffer
			rootCmd.SetOut(&outBuf)
			rootCmd.SetErr(&errBuf)
			rootCmd.SetArgs(tt.args)

			// Execute
			err := rootCmd.Execute()

			// Assert
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					output := errBuf.String() + err.Error()
					assert.Contains(t, output, tt.errorContains, "error should contain expected text")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateCommand_NamespaceIDValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		namespaceID string
		expectError bool
	}{
		{
			name:        "valid namespace ID",
			namespaceID: "1",
			expectError: false,
		},
		{
			name:        "namespace ID with underscores",
			namespaceID: "123",
			expectError: false,
		},
		{
			name:        "empty namespace ID",
			namespaceID: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			rootCmd := setupCreateCommandWithConfig(t, fakeDeploySuccess)
			args := []string{
				"namespace", "create",
			}
			if tt.namespaceID != "" {
				args = append(args, tt.namespaceID)
			}
			args = append(args,
				"--policy", "threshold:/tmp/pk.pem",
			)
			rootCmd.SetArgs(args)

			// Execute
			err := rootCmd.Execute()

			// Assert
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewCreateCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newCreateCommand(fakeDeploySuccess)

	// Assert
	require.NotNil(t, cmd, "newCreateCommand should return a non-nil command")
	assert.Equal(t, "create NAMESPACE_NAME", cmd.Use, "command use should be 'create NAMESPACE_NAME'")
	assert.NotEmpty(t, cmd.Short, "command should have a short description")
	assert.NotNil(t, cmd.RunE, "command should have a RunE function")

	policyFlag := cmd.Flag("policy")
	require.NotNil(t, policyFlag, "policy flag should exist")
}

// Test helpers

func setupCreateCommandWithConfig(t *testing.T, deploy deployF) *cobra.Command {
	t.Helper()

	createCmd := newCreateCommand(deploy)
	return setupNamespaceCommandWithConfig(t, createCmd)
}

func setupNamespaceCommandWithConfig(t *testing.T, cmd *cobra.Command) *cobra.Command {
	t.Helper()

	// Create proper command hierarchy: root -> namespace -> create
	rootCmd := &cobra.Command{Use: "fxconfig"}
	namespaceCmd := &cobra.Command{Use: "namespace"}

	namespaceCmd.AddCommand(cmd)
	rootCmd.AddCommand(namespaceCmd)

	// Mock config in context
	cfg := &config.Config{
		MSP: config.MSPConfig{
			LocalMspID: "TestMSP",
			ConfigPath: "/tmp/msp",
		},
		Orderer: config.OrdererConfig{
			EndpointServiceConfig: config.EndpointServiceConfig{
				Address:           "localhost:7050",
				ConnectionTimeout: 30 * time.Second,
			},
		},
		Queries: config.QueriesConfig{
			EndpointServiceConfig: config.EndpointServiceConfig{
				Address:           "localhost:7001",
				ConnectionTimeout: 30 * time.Second,
			},
		},
		Notifications: config.NotificationsConfig{
			EndpointServiceConfig: config.EndpointServiceConfig{
				Address:           "localhost:7002",
				ConnectionTimeout: 30 * time.Second,
			},
			WaitingTimeout: 45 * time.Second,
		},
	}

	// Mock config validator
	vctx := &config.ValidationContext{
		FileChecker:      &fakeFC{},
		DirectoryChecker: &fakeDC{},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, configKey, cfg)
	ctx = context.WithValue(ctx, configValidatorKey, vctx)
	rootCmd.SetContext(ctx)

	return rootCmd
}

type fakeFC struct{}

func (fakeFC) Exists(_ string) error {
	return nil
}

type fakeDC struct{}

func (fakeDC) Exists(_ string) error {
	return nil
}

func fakeDeploySuccess(_ config.ValidationContext, _ config.Config, _ config.NsConfig) error {
	return nil
}

func fakeDeployError(_ config.ValidationContext, _ config.Config, _ config.NsConfig) error {
	return errors.New("deployment failed")
}
