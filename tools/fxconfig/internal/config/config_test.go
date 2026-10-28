/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestTLSConfig_IsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      *TLSConfig
		expected bool
	}{
		{"nil receiver", (*TLSConfig)(nil), false},
		{"nil Enabled pointer", &TLSConfig{}, false},
		{"Enabled false", &TLSConfig{Enabled: boolPtr(false)}, false},
		{"Enabled true", &TLSConfig{Enabled: boolPtr(true)}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, tc.cfg.IsEnabled())
		})
	}
}

func TestTLSConfig_Normalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      *TLSConfig
		wantBool bool
	}{
		{"nil Enabled", &TLSConfig{}, false},
		{"already false", &TLSConfig{Enabled: boolPtr(false)}, false},
		{"already true", &TLSConfig{Enabled: boolPtr(true)}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.cfg.Normalize()
			require.NotNil(t, tc.cfg.Enabled)
			require.Equal(t, tc.wantBool, *tc.cfg.Enabled)
		})
	}
}

func TestTLSConfig_InheritFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		child  *TLSConfig
		parent *TLSConfig
		check  func(t *testing.T, result *TLSConfig)
	}{
		{
			name:   "nil parent returns child as-is",
			child:  &TLSConfig{Enabled: boolPtr(true), ClientKeyPath: "key.pem"},
			parent: nil,
			check: func(t *testing.T, result *TLSConfig) {
				t.Helper()
				require.True(t, *result.Enabled)
				require.Equal(t, "key.pem", result.ClientKeyPath)
			},
		},
		{
			name:  "nil child inherits all parent fields",
			child: nil,
			parent: &TLSConfig{
				Enabled: boolPtr(true), ClientKeyPath: "parent-key.pem", ClientCertPath: "parent-cert.pem",
			},
			check: func(t *testing.T, result *TLSConfig) {
				t.Helper()
				require.True(t, *result.Enabled)
				require.Equal(t, "parent-key.pem", result.ClientKeyPath)
				require.Equal(t, "parent-cert.pem", result.ClientCertPath)
			},
		},
		{
			name:  "child fields win over parent",
			child: &TLSConfig{Enabled: boolPtr(false), ClientKeyPath: "child-key.pem"},
			parent: &TLSConfig{
				Enabled: boolPtr(true), ClientKeyPath: "parent-key.pem", ClientCertPath: "parent-cert.pem",
			},
			check: func(t *testing.T, result *TLSConfig) {
				t.Helper()
				require.False(t, *result.Enabled)
				require.Equal(t, "child-key.pem", result.ClientKeyPath)
				require.Equal(t, "parent-cert.pem", result.ClientCertPath)
			},
		},
		{
			name:  "empty child inherits all parent fields",
			child: &TLSConfig{},
			parent: &TLSConfig{
				Enabled: boolPtr(true), ClientKeyPath: "p-key.pem", ServerNameOverride: "override",
			},
			check: func(t *testing.T, result *TLSConfig) {
				t.Helper()
				require.True(t, *result.Enabled)
				require.Equal(t, "p-key.pem", result.ClientKeyPath)
				require.Equal(t, "override", result.ServerNameOverride)
			},
		},
		{
			name:   "child RootCertPaths win over parent",
			child:  &TLSConfig{RootCertPaths: []string{"child-ca.pem"}},
			parent: &TLSConfig{RootCertPaths: []string{"parent-ca.pem"}},
			check: func(t *testing.T, result *TLSConfig) {
				t.Helper()
				require.Equal(t, []string{"child-ca.pem"}, result.RootCertPaths)
			},
		},
		{
			name:   "empty child RootCertPaths falls back to parent",
			child:  &TLSConfig{},
			parent: &TLSConfig{RootCertPaths: []string{"parent-ca.pem"}},
			check: func(t *testing.T, result *TLSConfig) {
				t.Helper()
				require.Equal(t, []string{"parent-ca.pem"}, result.RootCertPaths)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.child.InheritFrom(tc.parent)
			tc.check(t, result)
		})
	}
}

func TestConfig_ResolveTLS_InheritsParent(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		TLS: TLSConfig{
			Enabled:       boolPtr(true),
			RootCertPaths: []string{"root-ca.pem"},
		},
	}

	cfg.ResolveTLS()

	require.True(t, *cfg.TLS.Enabled)

	require.NotNil(t, cfg.Orderer.TLS)
	require.True(t, cfg.Orderer.TLS.IsEnabled())
	require.Equal(t, []string{"root-ca.pem"}, cfg.Orderer.TLS.RootCertPaths)

	require.NotNil(t, cfg.Queries.TLS)
	require.True(t, cfg.Queries.TLS.IsEnabled())
	require.Equal(t, []string{"root-ca.pem"}, cfg.Queries.TLS.RootCertPaths)

	require.NotNil(t, cfg.Notifications.TLS)
	require.True(t, cfg.Notifications.TLS.IsEnabled())
	require.Equal(t, []string{"root-ca.pem"}, cfg.Notifications.TLS.RootCertPaths)
}

func TestConfig_ResolveTLS_ServiceOverrides(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		TLS: TLSConfig{
			Enabled:       boolPtr(true),
			RootCertPaths: []string{"parent-ca.pem"},
		},
		Orderer: OrdererConfig{
			EndpointServiceConfig: EndpointServiceConfig{
				TLS: &TLSConfig{RootCertPaths: []string{"orderer-ca.pem"}},
			},
		},
	}

	cfg.ResolveTLS()

	require.Equal(t, []string{"orderer-ca.pem"}, cfg.Orderer.TLS.RootCertPaths)
	require.True(t, cfg.Orderer.TLS.IsEnabled())

	require.Equal(t, []string{"parent-ca.pem"}, cfg.Queries.TLS.RootCertPaths)
	require.Equal(t, []string{"parent-ca.pem"}, cfg.Notifications.TLS.RootCertPaths)
}

func TestMSPConfigToFactoryOpts_Defaults(t *testing.T) {
	t.Parallel()

	mspCfg := MSPConfig{
		ConfigPath: "/tmp/msp",
	}

	opts := mspCfg.ToFactoryOpts()

	require.Equal(t, "SW", opts.Default)
	require.NotNil(t, opts.SW)
	require.Equal(t, 256, opts.SW.Security)
	require.Equal(t, "SHA2", opts.SW.Hash)
	require.NotNil(t, opts.SW.FileKeystore)
	require.Equal(t, filepath.Join("/tmp/msp", "keystore"), opts.SW.FileKeystore.KeyStorePath)
}

func TestMSPConfigToFactoryOpts_Overrides(t *testing.T) {
	t.Parallel()

	mspCfg := MSPConfig{
		ConfigPath: "/tmp/msp",
		BCCSP: BCCSPConfig{
			Default: "SW",
			SW: BCCSPSWConfig{
				Security: 384,
				Hash:     "SHA3",
				FileKeyStore: BCCSPFileKeyStoreConfig{
					KeyStorePath: "/custom/keystore",
				},
			},
		},
	}

	opts := mspCfg.ToFactoryOpts()

	require.Equal(t, "SW", opts.Default)
	require.NotNil(t, opts.SW)
	require.Equal(t, 384, opts.SW.Security)
	require.Equal(t, "SHA3", opts.SW.Hash)
	require.NotNil(t, opts.SW.FileKeystore)
	require.Equal(t, "/custom/keystore", opts.SW.FileKeystore.KeyStorePath)
}
