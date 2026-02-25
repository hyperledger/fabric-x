/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package config provides validation functions for fxconfig configuration.
// It ensures all required fields are present and properly formatted before
// operations are executed.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-x-committer/service/verifier/policy"
)

// ValidateLoggingConfig validates logging configuration.
// Currently a placeholder for future validation logic.
func ValidateLoggingConfig(_ MSPConfig) error {
	// TODO: Implement logging config validation
	return nil
}

// ValidateMSPConfig validates MSP configuration.
// Ensures both LocalMspID and ConfigPath are specified.
func ValidateMSPConfig(cfg MSPConfig) error {
	return errors.Join(
		errorIfEmpty(cfg.LocalMspID, "LocalMspID must be specified"),
		errorIfEmpty(cfg.ConfigPath, "ConfigPath must be specified"),
	)
}

// ValidateEndpointServiceConfig validates service endpoint configuration.
// Checks address, timeout, and TLS settings for a given service.
func ValidateEndpointServiceConfig(service string, cfg EndpointServiceConfig) error {
	return errors.Join(
		errorIfEmpty(cfg.Address, fmt.Sprintf("%v.address must be specified", service)),
		errorIfZeroDuration(cfg.ConnectionTimeout, fmt.Sprintf("%v.connectionTimeout must be non-zero", service)),
		validateTLSConfig(service, cfg.GetTLSConfig()),
	)
}

// ValidateOrdererConfig validates orderer service configuration.
func ValidateOrdererConfig(service string, cfg OrdererConfig) error {
	return ValidateEndpointServiceConfig(service, cfg.EndpointServiceConfig)
}

// ValidateQueriesConfig validates queries service configuration.
func ValidateQueriesConfig(service string, cfg QueriesConfig) error {
	return ValidateEndpointServiceConfig(service, cfg.EndpointServiceConfig)
}

// ValidateNotificationsConfig validates notifications service configuration.
func ValidateNotificationsConfig(service string, cfg NotificationsConfig) error {
	return ValidateEndpointServiceConfig(service, cfg.EndpointServiceConfig)
}

// validateTLSConfig validates TLS configuration for a service.
// Ensures mutual TLS is properly configured when client credentials are provided.
func validateTLSConfig(service string, cfg TLSConfig) error {
	if !cfg.IsEnabled() {
		return nil
	}

	// TLS
	if len(cfg.RootCertPaths) == 0 {
		return fmt.Errorf("%s: TLS requires rootCerts", service)
	}

	// mTLS
	if cfg.ClientCertPath == "" || cfg.ClientKeyPath == "" {
		return fmt.Errorf("%s: mutual TLS clientKey, clientCert, and rootCerts", service)
	}

	return nil
}

// ValidateNsConfig validates namespace configuration for create/update operations.
// Checks channel name, namespace ID, version, and policy settings.
func ValidateNsConfig(cfg NsConfig) error {
	return errors.Join(
		validateChannel(cfg),
		policy.ValidateNamespaceID(cfg.NamespaceID),
		validateVersion(cfg),
		mustHavePolicy(cfg),
	)
}

// validateChannel ensures the channel name is not empty.
func validateChannel(cfg NsConfig) error {
	return errorIfEmpty(cfg.Channel, "channel name must be specified")
}

// validateVersion ensures the version is valid.
// Version -1 indicates a create operation, >= 0 indicates an update.
func validateVersion(cfg NsConfig) error {
	if cfg.Version < -1 {
		return errors.New("invalid version: must be -1 (create) or >= 0 (update)")
	}
	return nil
}

// mustHavePolicy ensures a policy verification key path is specified.
func mustHavePolicy(cfg NsConfig) error {
	return errorIfEmpty(cfg.ThresholdPolicyVerificationKeyPath, "policy verification key path must be specified")
}

// errorIfEmpty returns an error if the string is empty or whitespace-only.
func errorIfEmpty(s, errMsg string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New(errMsg)
	}
	return nil
}

// errorIfZeroDuration returns an error if the duration is zero.
func errorIfZeroDuration(d time.Duration, errMsg string) error {
	if d == time.Duration(0) {
		return errors.New(errMsg)
	}
	return nil
}
