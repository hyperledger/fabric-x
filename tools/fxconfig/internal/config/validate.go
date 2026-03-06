/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
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

	if err := vctx.DirectoryChecker.Exists(c.ConfigPath); err != nil {
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
	if err := validateEndpoint(c.Address); err != nil {
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
		if err := vctx.FileChecker.Exists(p); err != nil {
			return fmt.Errorf("invalid rootCertPath: %w", err)
		}
	}

	// mTLS
	if c.ClientCertPath == "" && c.ClientKeyPath == "" {
		return nil
	}

	// check that client cert exists
	if err := vctx.FileChecker.Exists(c.ClientCertPath); err != nil {
		return fmt.Errorf("invalid clientCertPath: %w", err)
	}

	// check that client key exists
	if err := vctx.FileChecker.Exists(c.ClientKeyPath); err != nil {
		return fmt.Errorf("invalid clientKeyPath: %w", err)
	}

	// try to load key pair
	if _, err := tls.LoadX509KeyPair(c.ClientCertPath, c.ClientKeyPath); err != nil {
		return fmt.Errorf("invalid cert/key pair: %w", err)
	}

	return nil
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
	if d == 0 {
		return errors.New(errMsg)
	}
	return nil
}

func validateEndpoint(endpoint string) error {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return err
	}
	if host == "" {
		return errors.New("host is empty")
	}
	if port == "" {
		return errors.New("port is empty")
	}
	return nil
}
