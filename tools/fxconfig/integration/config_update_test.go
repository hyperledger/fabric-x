/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	ab "github.com/hyperledger/fabric-protos-go-apiv2/orderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/client"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

// TestNamespaceAfterConfigUpdate checks that namespace create and update still commit
// after the channel config has changed (https://github.com/hyperledger/fabric-x/issues/281).
func TestNamespaceAfterConfigUpdate(t *testing.T) {
	t.Parallel()

	endpoints := setupSingleOrgAdmin(t)

	testdata, err := filepath.Abs("testdata")
	require.NoError(t, err)
	mspPath := path.Join(testdata, "crypto", "peerOrganizations", "org1.com", "users", "endorser@org1.com", "msp")
	configArg := "--config=" + generateFxConfigFile(t, "Org1MSP", mspPath, endpoints)

	_, err = fxconfig(t, "namespace", "create", "before_cfg_update",
		configArg, policyArg, endorseArg, submitArg, waitArg)
	require.NoError(t, err)

	submitConfigUpdate(t, endpoints, filepath.Join(testdata, "crypto", "single-org.pb.bin"))

	_, err = fxconfig(t, "namespace", "update", "before_cfg_update", "--version=0",
		configArg, policyArg, endorseArg, submitArg, waitArg)
	require.NoError(t, err)

	_, err = fxconfig(t, "namespace", "create", "after_cfg_update",
		configArg, policyArg, endorseArg, submitArg, waitArg)
	require.NoError(t, err)

	stdOut, err := fxconfig(t, "namespace", "list", configArg)
	require.NoError(t, err)
	nss, err := parseNamespaceList(stdOut)
	require.NoError(t, err)
	require.Contains(t, nss, Namespace{Name: "before_cfg_update", Version: 1})
	require.Contains(t, nss, Namespace{Name: "after_cfg_update", Version: 0})
}

// submitConfigUpdate re-submits the genesis channel config as a new config transaction and
// waits until it is committed, which advances the committed config version by one.
// The mock orderer orders any envelope it receives, so no orderer signatures are needed.
func submitConfigUpdate(t *testing.T, endpoints map[string]string, genesisPath string) {
	t.Helper()

	qc, err := client.NewQueryClient(config.QueriesConfig{EndpointServiceConfig: config.EndpointServiceConfig{
		Address: endpoints["query"], ConnectionTimeout: 5 * time.Second,
	}})
	require.NoError(t, err)
	defer func() { _ = qc.Close() }()
	before, err := qc.GetConfigVersion(t.Context())
	require.NoError(t, err)

	raw, err := os.ReadFile(genesisPath)
	require.NoError(t, err)
	block, err := protoutil.UnmarshalBlock(raw)
	require.NoError(t, err)
	genesisEnv, err := protoutil.ExtractEnvelope(block, 0)
	require.NoError(t, err)
	payload, err := protoutil.UnmarshalPayload(genesisEnv.Payload)
	require.NoError(t, err)
	cfgEnv, err := protoutil.UnmarshalConfigEnvelope(payload.Data)
	require.NoError(t, err)

	// Without LastUpdate, the committer takes the TX ID from the channel header, so it must be unique.
	chHdr := protoutil.MakeChannelHeader(cb.HeaderType_CONFIG, 0, channelID, 0)
	chHdr.TxId = transaction.GenerateTxID()
	env := &cb.Envelope{Payload: protoutil.MarshalOrPanic(&cb.Payload{
		Header: protoutil.MakePayloadHeader(chHdr, &cb.SignatureHeader{}),
		Data:   protoutil.MarshalOrPanic(&cb.ConfigEnvelope{Config: cfgEnv.Config}),
	})}

	conn, err := grpc.NewClient(endpoints["orderer"], grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	stream, err := ab.NewAtomicBroadcastClient(conn).Broadcast(t.Context())
	require.NoError(t, err)
	require.NoError(t, stream.Send(env))
	resp, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, cb.Status_SUCCESS, resp.GetStatus())

	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		v, err := qc.GetConfigVersion(t.Context())
		require.NoError(ct, err)
		require.Equal(ct, before+1, v)
	}, eventuallyTimeout, eventuallyTick)
}
