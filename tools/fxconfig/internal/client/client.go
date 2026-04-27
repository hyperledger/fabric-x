/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package client provides gRPC client implementations for Fabric-X services.
// It includes clients for orderer, query, and notification services with TLS support.
package client

import (
	"fmt"
	"os"

	"github.com/hyperledger/fabric-lib-go/common/flogging"
	"google.golang.org/grpc"

	"github.com/hyperledger/fabric-x-common/tools/pkg/comm"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

var logger = flogging.MustGetLogger("client")

func newClientConn(cfg *config.EndpointServiceConfig) (*grpc.ClientConn, error) {
	secOpts, err := createSecOpts(cfg.TLS)
	if err != nil {
		return nil, err
	}

	cc := comm.ClientConfig{
		SecOpts:     *secOpts,
		DialTimeout: cfg.ConnectionTimeout,
	}

	return cc.Dial(cfg.Address)
}

func createSecOpts(tlsConfig *config.TLSConfig) (*comm.SecureOptions, error) {
	var secOpts comm.SecureOptions

	// let's see if we use TLS
	if !tlsConfig.IsEnabled() {
		return &secOpts, nil
	}

	secOpts.UseTLS = true
	secOpts.ServerNameOverride = tlsConfig.ServerNameOverride

	// set rootCAs
	serverRootCAs := make([][]byte, 0, len(tlsConfig.RootCertPaths))
	for _, rootCertPath := range tlsConfig.RootCertPaths {
		rootCert, err := loadFile(rootCertPath)
		if err != nil {
			return nil, err
		}
		serverRootCAs = append(serverRootCAs, rootCert)
	}
	secOpts.ServerRootCAs = serverRootCAs

	// mTLS: both key and cert must be provided; if either is absent, skip mTLS
	if tlsConfig.ClientKeyPath == "" || tlsConfig.ClientCertPath == "" {
		if tlsConfig.ClientKeyPath != "" || tlsConfig.ClientCertPath != "" {
			logger.Warn("mTLS disabled: both clientKey and clientCert must be set; ignoring partial mTLS configuration")
		}
		return &secOpts, nil
	}

	// load client key and client cert
	keyBytes, err := loadFile(tlsConfig.ClientKeyPath)
	if err != nil {
		return nil, err
	}
	certBytes, err := loadFile(tlsConfig.ClientCertPath)
	if err != nil {
		return nil, err
	}

	secOpts.RequireClientCert = true
	secOpts.Key = keyBytes
	secOpts.Certificate = certBytes

	return &secOpts, nil
}

func loadFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed opening file %s: %w", path, err)
	}
	return b, nil
}
