/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"context"
	"testing"
	"time"

	common "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x-common/tools/pkg/comm"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type BlockQueryClient struct {
	Endpoint   string
	TLSCACerts [][]byte
	Key        []byte
	Cert       []byte
}

func NewBlockQueryClient(t *testing.T, endpoint string, key []byte, cert []byte, caCerts [][]byte) *BlockQueryClient {
	t.Helper()

	return &BlockQueryClient{
		Endpoint:   endpoint,
		TLSCACerts: caCerts,
		Key:        key,
		Cert:       cert,
	}
}

func (c *BlockQueryClient) PullBlocks(t *testing.T, ctx context.Context) <-chan *common.Block {
	t.Helper()

	client, conn := c.createClient(t)
	t.Cleanup(func() {
		conn.Close()
	})

	info, err := (*client).GetBlockchainInfo(ctx, &emptypb.Empty{})
	require.NoError(t, err, "failed to get blockchain info")

	bcHeight := info.GetHeight()
	t.Logf("Blockchain height: %d", bcHeight)
	blocksCh := make(chan *common.Block, bcHeight)

	go func() {
		defer close(blocksCh)
		for targetBlockNum := range bcHeight {
			block, err := (*client).GetBlockByNumber(ctx, &committerpb.BlockNumber{Number: targetBlockNum})
			require.NoError(t, err, "failed to get block by number")

			t.Logf("Retrieved Block Header: Number=%d, DataHash=%x\n",
				block.GetHeader().GetNumber(),
				block.GetHeader().GetDataHash(),
			)
			blocksCh <- block
		}
	}()
	return blocksCh
}

func (c *BlockQueryClient) createClient(t *testing.T) (*committerpb.BlockQueryServiceClient, *grpc.ClientConn) {
	serverRootCAs := append([][]byte{}, c.TLSCACerts...)

	// create a gRPC connection to the assembler
	grpcClient := comm.ClientConfig{
		KaOpts: comm.KeepaliveOptions{
			ClientInterval: time.Hour,
			ClientTimeout:  time.Hour,
		},
		SecOpts: comm.SecureOptions{
			Key:               c.Key,
			Certificate:       c.Cert,
			RequireClientCert: true,
			UseTLS:            true,
			ServerRootCAs:     serverRootCAs,
		},
		DialTimeout: time.Second * 5,
	}

	conn, err := grpcClient.Dial(c.Endpoint)
	require.NoError(t, err, "failed to dial gRPC server")

	blockClient := committerpb.NewBlockQueryServiceClient(conn)

	return &blockClient, conn
}
