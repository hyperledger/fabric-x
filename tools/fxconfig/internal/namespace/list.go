/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x-common/cmd/common/comm"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// ListNamespaces queries the committer service for installed namespaces.
// It connects to the query service, retrieves all namespace policies, and formats
// the output showing namespace names, versions, and policy data in hexadecimal.
func ListNamespaces(out io.Writer, cfg config.QueriesConfig) error {
	// TODO we are very restricted with this

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
		return fmt.Errorf("cannot get grpc client: %w", err)
	}

	conn, err := cl.NewDialer(cfg.Address)()
	if err != nil {
		return fmt.Errorf("dialing grpc client error: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	client := committerpb.NewQueryServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	res, err := client.GetNamespacePolicies(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("cannot query existing namespaces: %w", err)
	}

	printResult(out, res)

	return nil
}

// printResult formats and writes namespace policy information to the output writer.
// Each namespace is displayed with its index, name, version, and policy in hexadecimal format.
//
//nolint:errcheck
func printResult(out io.Writer, res *applicationpb.NamespacePolicies) {
	fmt.Fprintf(out, "Installed namespaces (%d total):\n", len(res.GetPolicies()))
	for i, p := range res.GetPolicies() {
		fmt.Fprintf(out, "%d) %v: version %d policy: %x\n", i, p.GetNamespace(), p.GetVersion(), p.GetPolicy())
	}
	fmt.Fprintln(out, "")
}
