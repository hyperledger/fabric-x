/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package config provides configuration structures and utilities for fxconfig.
// It defines the configuration schema and handles loading from multiple sources
// with a well-defined precedence hierarchy.
package config

import (
	"cmp"
	"slices"
	"time"
)

// Config represents the complete fxconfig configuration.
// It includes settings for logging, MSP identity, TLS, and service endpoints.
type Config struct {
	Logging       LoggingConfig       `mapstructure:"logging" yaml:"logging,omitempty"`
	MSP           MSPConfig           `mapstructure:"msp" yaml:"msp,omitempty"`
	TLS           TLSConfig           `mapstructure:"tls" yaml:"tls,omitempty"`
	Orderer       OrdererConfig       `mapstructure:"orderer" yaml:"orderer,omitempty"`
	Queries       QueriesConfig       `mapstructure:"queries" yaml:"queries,omitempty"`
	Notifications NotificationsConfig `mapstructure:"notifications" yaml:"notifications,omitempty"`
}

// ResolveTLS applies TLS configuration inheritance across all services.
// Each service inherits from the parent TLS config unless it provides overrides.
// After merging, all TLS configs are normalized to have explicit enabled flags.
func (c *Config) ResolveTLS() {
	c.TLS.Normalize()

	c.Orderer.TLS = c.Orderer.TLS.InheritFrom(&c.TLS)
	c.Orderer.TLS.Normalize()

	c.Queries.TLS = c.Queries.TLS.InheritFrom(&c.TLS)
	c.Queries.TLS.Normalize()

	c.Notifications.TLS = c.Notifications.TLS.InheritFrom(&c.TLS)
	c.Notifications.TLS.Normalize()
}

// LoggingConfig controls logging behavior.
type LoggingConfig struct {
	Level  string `mapstructure:"level" yaml:"level,omitempty" desc:"Logging level" default:"error"`
	Format string `mapstructure:"format" yaml:"format,omitempty" desc:"logging format"`
}

// MSPConfig contains MSP (Membership Service Provider) identity configuration.
// It specifies which organization identity to use for signing transactions.
type MSPConfig struct {
	LocalMspID string      `mapstructure:"localMspID" yaml:"localMspID,omitempty" desc:"MSP ID of the organization"`
	ConfigPath string      `mapstructure:"configPath" yaml:"configPath,omitempty" desc:"Path to MSP configuration directory"`
	BCCSP      BCCSPConfig `mapstructure:"bccsp" yaml:"bccsp,omitempty" desc:"BCCSP (crypto provider) configuration; leave empty for default file-based keystore"`
}

// BCCSPConfig configures the BCCSP (Blockchain Crypto Service Provider).
// Leaving PKCS11.Library empty falls back to the default file-based software BCCSP,
// which reads keys from <ConfigPath>/keystore.
type BCCSPConfig struct {
	PKCS11 PKCS11Config `mapstructure:"pkcs11" yaml:"pkcs11,omitempty" desc:"PKCS#11 HSM/KMS provider configuration; enables PKCS11 mode when Library is set"`
}

// PKCS11Config configures a PKCS#11 HSM/KMS crypto provider.
// When Library is non-empty, PKCS11 mode is activated and keys are loaded from the HSM
// rather than from a file-based keystore.
type PKCS11Config struct {
	Library        string `mapstructure:"library" yaml:"library,omitempty" desc:"Path to the PKCS#11 shared library (.so)"`
	Label          string `mapstructure:"label" yaml:"label,omitempty" desc:"PKCS#11 token label"`
	Pin            string `mapstructure:"pin" yaml:"pin,omitempty" desc:"PKCS#11 user PIN"`
	Hash           string `mapstructure:"hash" yaml:"hash,omitempty" desc:"Hash family (SHA2 or SHA3)" default:"SHA2"`
	Security       int    `mapstructure:"security" yaml:"security,omitempty" desc:"Security level in bits (256 or 384)" default:"256"`
	SoftwareVerify bool   `mapstructure:"softwareVerify" yaml:"softwareVerify,omitempty" desc:"Perform signature verification in software"`
	Immutable      bool   `mapstructure:"immutable" yaml:"immutable,omitempty" desc:"Treat keys as immutable"`
}

// TLSConfig specifies TLS settings for secure communication.
// Supports three modes: no TLS, server TLS (rootCerts only), and mutual TLS (all fields).
//
//nolint:revive,lll
type TLSConfig struct {
	Enabled            *bool    `mapstructure:"enabled" yaml:"enabled,omitempty" desc:"Enable/disable TLS" default:"false"`
	ClientKeyPath      string   `mapstructure:"clientKey" yaml:"clientKey,omitempty" desc:"Path to TLS client private key"`
	ClientCertPath     string   `mapstructure:"clientCert" yaml:"clientCert,omitempty" desc:"Path to TLS client certificate"`
	RootCertPaths      []string `mapstructure:"rootCerts" yaml:"rootCerts,omitempty" desc:"Paths to TLS root certificates"`
	ServerNameOverride string   `mapstructure:"serverNameOverride" yaml:"serverNameOverride,omitempty" desc:"Override TLS server name"`
}

// Normalize ensures the TLS config has an explicit enabled flag.
// Sets enabled to false if not specified.
func (c *TLSConfig) Normalize() {
	if c.Enabled == nil {
		enabled := false
		c.Enabled = &enabled
	}
}

// InheritFrom returns a new TLSConfig that merges c with parent, preferring c's values where set.
func (c *TLSConfig) InheritFrom(parent *TLSConfig) *TLSConfig {
	if c == nil {
		c = &TLSConfig{}
	}

	if parent == nil {
		return c
	}

	result := &TLSConfig{
		Enabled:            cmp.Or(c.Enabled, parent.Enabled),
		ClientKeyPath:      cmp.Or(c.ClientKeyPath, parent.ClientKeyPath),
		ClientCertPath:     cmp.Or(c.ClientCertPath, parent.ClientCertPath),
		ServerNameOverride: cmp.Or(c.ServerNameOverride, parent.ServerNameOverride),
	}

	src := parent.RootCertPaths
	if len(c.RootCertPaths) > 0 {
		src = c.RootCertPaths
	}
	result.RootCertPaths = slices.Clone(src)

	return result
}

// IsEnabled returns whether TLS is enabled for this configuration.
// Returns false if the config is nil or the enabled flag is not set.
func (c *TLSConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false // default
	}
	return *c.Enabled
}

// OrdererConfig contains configuration for the ordering service endpoint.
//
//nolint:revive,lll
type OrdererConfig struct {
	EndpointServiceConfig `mapstructure:",squash" yaml:",inline"`
	Channel               string `mapstructure:"channel" yaml:"channel,omitempty" desc:"Orderer channel name" default:"mychannel"`
}

// QueriesConfig contains configuration for the query service endpoint.
type QueriesConfig struct {
	EndpointServiceConfig `mapstructure:",squash"  yaml:",inline"`
}

// NotificationsConfig contains configuration for the notifications service endpoint.
// Includes a waiting timeout for notification processing operations.
//
//nolint:revive,lll
type NotificationsConfig struct {
	EndpointServiceConfig `mapstructure:",squash" yaml:",inline"`
	WaitingTimeout        time.Duration `mapstructure:"waitingTimeout" yaml:"waitingTimeout,omitempty" desc:"Time to wait for notification processing" default:"30s"`
}

// EndpointServiceConfig defines connection settings for a Fabric-X service.
// Each service (orderer, queries, notifications) can have its own configuration.
//
//nolint:revive,lll
type EndpointServiceConfig struct {
	Address           string        `mapstructure:"address" yaml:"address,omitempty" desc:"Service address (host:port)"`
	ConnectionTimeout time.Duration `mapstructure:"connectionTimeout" yaml:"connectionTimeout,omitempty" desc:"Connection timeout duration" default:"30s"`
	TLS               *TLSConfig    `mapstructure:"tls" yaml:"tls,omitempty" desc:"(Optional) Overrides parent TLS section"`
}
