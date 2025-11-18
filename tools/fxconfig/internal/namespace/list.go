/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"context"
	"fmt"
	"io"

	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
	"github.com/hyperledger/fabric-x-committer/api/protoqueryservice"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hyperledger/fabric-x-common/cmd/common/comm"
)

// List calls the committer query service and shows all installed namespace policies.
func List(out io.Writer, endpoint string) error {
	cl, err := comm.NewClient(comm.Config{})
	if err != nil {
		return fmt.Errorf("cannot get grpc client: %w", err)
	}

	conn, err := cl.NewDialer(endpoint)()
	if err != nil {
		return fmt.Errorf("dialing grpc client error: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	client := protoqueryservice.NewQueryServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	res, err := client.GetNamespacePolicies(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("cannot query existing namespaces: %w", err)
	}

	printResult(out, res)

	return nil
}

//nolint:errcheck
func printResult(out io.Writer, res *protoblocktx.NamespacePolicies) {
	fmt.Fprintf(out, "Installed namespaces (%d total):\n", len(res.GetPolicies()))
	for i, p := range res.GetPolicies() {
		fmt.Fprintf(out, "%d) %v: version %d policy: %x\n", i, p.GetNamespace(), p.GetVersion(), p.GetPolicy())
	}
	fmt.Fprintln(out, "")
}
