/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package config provides configuration structures and utilities for fxconfig.
// It defines the configuration schema and handles loading from multiple sources
// with a well-defined precedence hierarchy.
package config

import (
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
	resolveServiceTLS(&c.Orderer.EndpointServiceConfig, c.TLS)
	resolveServiceTLS(&c.Queries.EndpointServiceConfig, c.TLS)
	resolveServiceTLS(&c.Notifications.EndpointServiceConfig, c.TLS)

	normalizeTLS(&c.TLS)
	normalizeTLS(c.Orderer.TLS)
	normalizeTLS(c.Queries.TLS)
	normalizeTLS(c.Notifications.TLS)
}

// normalizeTLS ensures the TLS config has an explicit enabled flag.
// Sets enabled to false if not specified.
func normalizeTLS(t *TLSConfig) {
	if t == nil {
		return
	}

	if t.Enabled == nil {
		defaultValue := false
		t.Enabled = &defaultValue
	}
}

// resolveServiceTLS merges service-specific TLS config with parent TLS config.
// If the service has no TLS override, it inherits the parent config completely.
// Otherwise, it merges field-by-field, with service values taking precedence.
func resolveServiceTLS(s *EndpointServiceConfig, parent TLSConfig) {
	if s.TLS == nil {
		// No override -> inherit fully
		s.TLS = parent.Clone()
		return
	}

	// Merge field-by-field
	mergeTLS(s.TLS, parent)
}

// mergeTLS merges service-specific TLS overrides with parent TLS settings.
// Service values take precedence; parent values are used as fallbacks.
func mergeTLS(override *TLSConfig, parent TLSConfig) {
	if override.Enabled == nil {
		override.Enabled = parent.Enabled
	}

	if override.ClientKeyPath == "" {
		override.ClientKeyPath = parent.ClientKeyPath
	}

	if override.ClientCertPath == "" {
		override.ClientCertPath = parent.ClientCertPath
	}

	if len(override.RootCertPaths) == 0 {
		override.RootCertPaths = parent.RootCertPaths
	}

	if override.ServerNameOverride == "" {
		override.ServerNameOverride = parent.ServerNameOverride
	}
}

// LoggingConfig controls logging behavior.
type LoggingConfig struct {
	Level  string `mapstructure:"level" yaml:"level,omitempty" desc:"Logging level" default:"error"`
	Format string `mapstructure:"format" yaml:"format,omitempty" desc:"logging format"`
}

// MSPConfig contains MSP (Membership Service Provider) identity configuration.
// It specifies which organization identity to use for signing transactions.
type MSPConfig struct {
	LocalMspID string `mapstructure:"localMspID" yaml:"localMspID,omitempty" desc:"MSP ID of the organization"`
	ConfigPath string `mapstructure:"configPath" yaml:"configPath,omitempty" desc:"Path to MSP configuration directory"`
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

// Clone creates a shallow copy of the TLS configuration.
func (c *TLSConfig) Clone() *TLSConfig {
	if c == nil {
		return nil
	}

	// copy all non-pointer types
	clone := *c

	// copy enabled
	if c.Enabled != nil {
		v := *c.Enabled
		clone.Enabled = &v
	}

	if len(c.RootCertPaths) > 0 {
		clone.RootCertPaths = make([]string, len(clone.RootCertPaths))
		copy(clone.RootCertPaths, c.RootCertPaths)
	}

	return &clone
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

// GetTLSConfig returns the TLS configuration for this service endpoint.
// Returns an empty TLSConfig if no TLS override is configured.
func (c *EndpointServiceConfig) GetTLSConfig() TLSConfig {
	if c.TLS == nil {
		return TLSConfig{}
	}
	return *c.TLS
}
