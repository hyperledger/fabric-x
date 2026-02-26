/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"errors"
	"fmt"

	"github.com/hyperledger/fabric-x-committer/service/verifier/policy"
)

// Validatable defines types that can be validated against a ValidationContext.
type Validatable interface {
	Validate(ctx ValidationContext) error
}

// Validate validates MSP configuration.
// Ensures both LocalMspID and ConfigPath are specified.
func (c *MSPConfig) Validate(ctx ValidationContext) error {
	if err := errorIfEmpty(c.LocalMspID, "must not be empty"); err != nil {
		return fmt.Errorf("invalid localMspID: %w", err)
	}

	if err := validateDirectoryPath(ctx, c.ConfigPath); err != nil {
		return fmt.Errorf("invalid configPath: %w", err)
	}

	return nil
}

// Validate validates Orderer configuration.
// Check channel name and endpoint service configuration.
func (c *OrdererConfig) Validate(ctx ValidationContext) error {
	if err := errorIfEmpty(c.Channel, "empty"); err != nil {
		return fmt.Errorf("invalid channel: %w", err)
	}
	return c.EndpointServiceConfig.Validate(ctx)
}

// Validate validates service endpoint configuration.
// Checks address, timeout, and TLS settings for a given service.
func (c *EndpointServiceConfig) Validate(ctx ValidationContext) error {
	// TODO we should validate this better
	if err := errorIfEmpty(c.Address, "must not be empty"); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	if err := errorIfZeroDuration(c.ConnectionTimeout, "must be non-zero"); err != nil {
		return fmt.Errorf("invalid connection timeout: %w", err)
	}

	if err := c.TLS.Validate(ctx); err != nil {
		return fmt.Errorf("invalid tls configuration: %w", err)
	}

	return nil
}

// Validate validates TLS configuration for a service.
// Ensures mutual TLS is properly configured when client credentials are provided.
func (c *TLSConfig) Validate(ctx ValidationContext) error {
	if !c.IsEnabled() {
		return nil
	}

	// TLS
	if len(c.RootCertPaths) == 0 {
		return errors.New("rootCertPaths must not be empty")
	}

	// check all provided root certs exist
	for _, p := range c.RootCertPaths {
		if err := validateFilePath(ctx, p); err != nil {
			return fmt.Errorf("invalid clientCertPath: %w", err)
		}
	}

	// mTLS
	if c.ClientCertPath == "" && c.ClientKeyPath == "" {
		return nil
	}

	if err := validateFilePath(ctx, c.ClientCertPath); err != nil {
		return fmt.Errorf("invalid clientCertPath: %w", err)
	}

	if err := validateFilePath(ctx, c.ClientKeyPath); err != nil {
		return fmt.Errorf("invalid clientKeyPath: %w", err)
	}

	return nil
}

// Validate validates namespace configuration.
// Checks namespace ID, version, and policy.
func (c *NsConfig) Validate(ctx ValidationContext) error {
	if err := policy.ValidateNamespaceID(c.NamespaceID); err != nil {
		return fmt.Errorf("invalid namespaceID: %w", err)
	}

	if err := validateVersion(c.Version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	if err := c.Policy.Validate(ctx); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	return nil
}

// Validate validates policy configuration based on policy type.
func (c *PolicyConfig) Validate(ctx ValidationContext) error {
	if c == nil {
		return errors.New("policy is required")
	}

	switch c.Type {
	case "msp":
		if c.MSP == nil {
			return errors.New("msp policy config missing")
		}
		return c.MSP.Validate(ctx)

	case "threshold":
		if c.Threshold == nil {
			return errors.New("threshold policy config missing")
		}
		return c.Threshold.Validate(ctx)

	default:
		return fmt.Errorf("unknown policy type: %s", c.Type)
	}
}

// Validate validates threshold policy configuration.
// Ensures the verification key path exists and is accessible.
func (c *ThresholdPolicyConfig) Validate(ctx ValidationContext) error {
	if err := validateFilePath(ctx, c.VerificationKeyPath); err != nil {
		return fmt.Errorf("invalid verificationKeyPath: %w", err)
	}
	return nil
}

// Validate validates MSP policy configuration.
// Checks that the policy expression is valid DSL syntax.
func (c *MSPPolicyConfig) Validate(ctx ValidationContext) error {
	if c.Expression == "" {
		return errors.New("msp policy expression must not be empty")
	}

	if ctx.PolicyChecker == nil {
		return errors.New("policy checker not available")
	}

	return ctx.PolicyChecker.Check(c.Expression)
}
