/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// Validate validates MSP configuration.
// Ensures both LocalMspID and ConfigPath are specified.
func (c *MSPConfig) Validate(vctx validation.Context) error {
	if err := errorIfEmpty(c.LocalMspID, "must not be empty"); err != nil {
		return fmt.Errorf("invalid localMspID: %w", err)
	}

	if err := validateDirectoryPath(vctx, c.ConfigPath); err != nil {
		return fmt.Errorf("invalid configPath: %w", err)
	}

	return nil
}

// Validate validates Orderer configuration.
// Check channel name and endpoint service configuration.
func (c *OrdererConfig) Validate(vctx validation.Context) error {
	if err := errorIfEmpty(c.Channel, "empty"); err != nil {
		return fmt.Errorf("invalid channel: %w", err)
	}
	return c.EndpointServiceConfig.Validate(vctx)
}

// Validate validates service endpoint configuration.
// Checks address, timeout, and TLS settings for a given service.
func (c *EndpointServiceConfig) Validate(vctx validation.Context) error {
	// TODO we should validate this better
	if err := errorIfEmpty(c.Address, "must not be empty"); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	if err := errorIfZeroDuration(c.ConnectionTimeout, "must be non-zero"); err != nil {
		return fmt.Errorf("invalid connection timeout: %w", err)
	}

	if err := c.TLS.Validate(vctx); err != nil {
		return fmt.Errorf("invalid tls configuration: %w", err)
	}

	return nil
}

// Validate validates TLS configuration for a service.
// Ensures mutual TLS is properly configured when client credentials are provided.
func (c *TLSConfig) Validate(vctx validation.Context) error {
	if !c.IsEnabled() {
		return nil
	}

	// TLS
	if len(c.RootCertPaths) == 0 {
		return errors.New("rootCertPaths must not be empty")
	}

	// check all provided root certs exist
	for _, p := range c.RootCertPaths {
		if err := validateFilePath(vctx, p); err != nil {
			return fmt.Errorf("invalid clientCertPath: %w", err)
		}
	}

	// mTLS
	if c.ClientCertPath == "" && c.ClientKeyPath == "" {
		return nil
	}

	if err := validateFilePath(vctx, c.ClientCertPath); err != nil {
		return fmt.Errorf("invalid clientCertPath: %w", err)
	}

	if err := validateFilePath(vctx, c.ClientKeyPath); err != nil {
		return fmt.Errorf("invalid clientKeyPath: %w", err)
	}

	return nil
}

// validateFilePath checks that a file path is non-empty and resolvable via FileChecker.
func validateFilePath(vctx validation.Context, p string) error {
	if p == "" {
		return errors.New("path must not be empty")
	}

	if vctx.FileChecker == nil {
		return errors.New("file checker not available")
	}

	return vctx.FileChecker.Exists(p)
}

func validateDirectoryPath(vctx validation.Context, p string) error {
	if p == "" {
		return errors.New("path must not be empty")
	}

	if vctx.DirectoryChecker == nil {
		return errors.New("directory checker not available")
	}

	return vctx.DirectoryChecker.Exists(p)
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
