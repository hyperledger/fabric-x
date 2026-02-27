/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"context"
	"fmt"
	"io"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	client2 "github.com/hyperledger/fabric-x/tools/fxconfig/internal/client"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// ListNamespaces queries the committer service for installed namespaces.
// It connects to the query service, retrieves all namespace policies, and formats
// the output showing namespace names, versions, and policy data in hexadecimal.
func ListNamespaces(vctx config.ValidationContext, cfg config.QueriesConfig, out io.Writer) error {
	if err := cfg.Validate(vctx); err != nil {
		return err
	}

	// TODO we move this creation somewhere else
	qc, err := client2.NewQueryClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot query existing namespaces: %w", err)
	}

	res, err := qc.GetNamespacePolicies(context.Background())
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

// parsePolicy extracts and formats policy information from serialized bytes.
// Returns base64-encoded public key for threshold policies or string representation for MSP policies.
// func parsePolicy(b []byte) string {
//	var p applicationpb.NamespacePolicy
//	if err := proto.Unmarshal(b, &p); err != nil {
//		panic(err)
//	}
//
//	switch r := p.Rule.(type) {
//	case *applicationpb.NamespacePolicy_ThresholdRule:
//		return base64.StdEncoding.EncodeToString(r.ThresholdRule.GetPublicKey())
//	case *applicationpb.NamespacePolicy_MspRule:
//		var en common.SignaturePolicy
//		if err := proto.Unmarshal(r.MspRule, &en); err != nil {
//			panic(err)
//		}
//		// TODO: some pretty print would be beautiful
//		return en.String()
//	default:
//		return "error parsing policy"
//	}
// }
