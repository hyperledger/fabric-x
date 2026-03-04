/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hyperledger/fabric-lib-go/common/flogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:revive
//go:generate go tool cryptogen generate --config testdata/crypto-config.yaml --output testdata/crypto
//go:generate go tool configtxgen --configPath testdata --channelID mychannel --profile OrgsChannel --outputBlock testdata/crypto/config-block.pb.bin

const (
	eventuallyTimeout = 10 * time.Second
	eventuallyTick    = 100 * time.Millisecond

	scaleTestNamespaceCount = 50 // test with many namespaces
)

func TestScenarios(t *testing.T) {
	// tame the configtxgen logger
	flogging.Init(flogging.Config{
		Format:  "",
		LogSpec: "common.tools.configtxgen=error",
	})

	// setup committer test container
	endpoints := setup(t)

	t.Logf("endpoints: %v", endpoints)

	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	require.NoError(t, err)

	// endorser MSP configuration
	localMspID := "Org1MSP"
	mspConfigPath := filepath.Join(
		testdata,
		"crypto",
		"peerOrganizations",
		"Org1",
		"users",
		"endorser@org1.com",
		"msp",
	)

	thresholdKeyPath := filepath.Join(
		testdata,
		"crypto",
		"peerOrganizations",
		"Org1",
		"users",
		"endorser@org1.com",
		"msp",
		"signcerts",
		"endorser@org1.com-cert.pem",
	)

	// Setup - Create temporary config file
	configPath := generateConfigFile(t, localMspID, mspConfigPath, endpoints)

	var (
		configArg  = "--config=" + configPath
		policyArg  = "--policy=threshold:" + thresholdKeyPath
		endorseArg = "--endorse"
		submitArg  = "--submit"
		waitArg    = "--wait"
	)

	t.Run("create_new_namespace", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "hello", Version: 0}

		// we expect no namespaces
		stdOut, err := fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, expectedNs)

		// create namespace hello
		_, err = fxconfig(t, "namespace", "create", expectedNs.Name,
			configArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		// expect one installed namespace
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)
	})

	t.Run("create_new_namespace_with_wait", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "hello_with_wait", Version: 0}

		// we expect no namespaces
		stdOut, err := fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, expectedNs)

		// create namespace hello
		_, err = fxconfig(t, "namespace", "create", expectedNs.Name,
			configArg, policyArg, endorseArg, submitArg, waitArg)
		require.NoError(t, err)

		// expect one installed namespace
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)
	})

	t.Run("duplicate_namespace_creation_fails", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "dp1", Version: 0}

		// we expect this namespace not to exist
		stdOut, err := fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, expectedNs)

		// create namespace
		_, err = fxconfig(t, "namespace", "create", expectedNs.Name,
			configArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		// expect out namespace to be installed
		// we keep the stdOut
		var expectedStdOut string
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err = fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err = parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
			expectedStdOut = stdOut
		}, eventuallyTimeout, eventuallyTick)

		// now we try to run create with the namespace again,
		// but we use a different policy, as namespace creation should fail,
		// and we expect the previous stdOut when calling list
		_, err = fxconfig(t, "namespace", "create", expectedNs.Name,
			configArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err = fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
			require.Equal(ct, expectedStdOut, stdOut)
			expectedStdOut = stdOut
		}, eventuallyTimeout, eventuallyTick)
	})

	t.Run("update_namespace", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "ns1", Version: 0}

		// we expect this namespace not to exist
		stdOut, err := fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, expectedNs)

		// create namespace
		_, err = fxconfig(t, "namespace", "create", expectedNs.Name,
			configArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		// we expect our namespace to be created
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err = fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err = parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)

		// we need to set the current version
		versionArg := "--version=0"

		// update namespace
		expectedNs = Namespace{Name: "ns1", Version: 1}
		_, err = fxconfig(t, "namespace", "update", expectedNs.Name,
			configArg, versionArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		// we expect our namespace to be updated having a version equals 1
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err = fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err = parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)

		// if we set the wrong version, it should fail
		versionArg = "--version=0"

		// update namespace
		expectedNs = Namespace{Name: "ns1", Version: 1}
		_, err = fxconfig(t, "namespace", "update", expectedNs.Name,
			configArg, versionArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		// we expect our namespace to be updated having a version equals 1
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err = fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err = parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)

		// should succeed with correct version
		versionArg = "--version=1"

		// update namespace
		expectedNs = Namespace{Name: "ns1", Version: 2}
		_, err = fxconfig(t, "namespace", "update", expectedNs.Name,
			configArg, versionArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		// we expect our namespace to be updated having a version equals 1
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)
	})

	t.Run("create_many_namespaces", func(t *testing.T) {
		t.Parallel()

		// create many namespaces
		for i := 1; i <= scaleTestNamespaceCount; i++ {
			expectedNs := Namespace{Name: fmt.Sprintf("hello%d", i)}
			_, err := fxconfig(t, "namespace", "create", expectedNs.Name,
				configArg, policyArg, endorseArg, submitArg)
			require.NoError(t, err)
		}

		// expect all our namespaces to be installed
		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Condition(ct, func() (success bool) {
				return len(nss) >= 50
			})
			for i := 1; i <= scaleTestNamespaceCount; i++ {
				require.Contains(ct, nss, Namespace{Name: fmt.Sprintf("hello%d", i), Version: 0})
			}
		}, eventuallyTimeout, eventuallyTick)
	})
}
