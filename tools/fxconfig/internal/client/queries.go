/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x-common/cmd/common/comm"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/api"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// QueryProvider constructs QueryClient instances with validation.
type QueryProvider struct {
	ValidationContext validation.Context
	Cfg               config.QueriesConfig
	// TODO make this provide once
}

// Validate checks the query configuration against the validation context.
func (f *QueryProvider) Validate() error {
	return f.Cfg.Validate(f.ValidationContext)
}

// Get creates and returns a new QueryClient instance.
func (f *QueryProvider) Get() (api.QueryClient, error) {
	return NewQueryClient(f.Cfg)
}

// QueryClient provides a gRPC client for querying namespace policies from the Fabric-X committer query service.
type QueryClient struct {
	cfg    config.QueriesConfig
	client committerpb.QueryServiceClient
	closeF func()
}

// NewQueryClient creates a new query client with the provided configuration.
// It establishes a gRPC connection with optional TLS and returns an error if connection fails.
func NewQueryClient(cfg config.QueriesConfig) (*QueryClient, error) {
	clientCfg := comm.Config{
		Timeout: cfg.ConnectionTimeout,
	}

	// TLS config
	if cfg.TLS.IsEnabled() {
		clientCfg.CertPath = cfg.TLS.ClientCertPath
		clientCfg.KeyPath = cfg.TLS.ClientKeyPath
		clientCfg.PeerCACertPath = cfg.TLS.RootCertPaths[0]
	}

	cl, err := comm.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot get grpc client: %w", err)
	}

	conn, err := cl.NewDialer(cfg.Address)()
	if err != nil {
		return nil, fmt.Errorf("dialing grpc client error: %w", err)
	}

	return &QueryClient{
		cfg:    cfg,
		client: committerpb.NewQueryServiceClient(conn),
		closeF: func() {
			_ = conn.Close()
		},
	}, nil
}

// GetNamespacePolicies retrieves all namespace policies from the query service.
// The request is bounded by the configured connection timeout.
func (qc *QueryClient) GetNamespacePolicies(ctx context.Context) (*applicationpb.NamespacePolicies, error) {
	if qc.client == nil {
		return nil, errors.New("require client")
	}

	ctx, cancel := context.WithTimeout(ctx, qc.cfg.ConnectionTimeout)
	defer cancel()

	res, err := qc.client.GetNamespacePolicies(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("getNamespacePolicies error: %w", err)
	}

	return res, nil
}

// Close terminates the gRPC connection to the query service.
func (qc *QueryClient) Close() error {
	if qc.closeF != nil {
		qc.closeF()
	}
	return nil
}
