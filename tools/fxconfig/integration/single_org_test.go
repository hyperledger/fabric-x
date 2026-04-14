/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

//nolint:revive
//go:generate go tool cryptogen generate --config testdata/crypto-config.yaml --output testdata/crypto
//go:generate go tool configtxgen --configPath testdata --channelID mychannel --profile SingleOrgAdminChannel --outputBlock testdata/crypto/single-org.pb.bin
//go:generate go tool configtxgen --configPath testdata --channelID mychannel --profile MultiOrgAdminChannel --outputBlock testdata/crypto/multi-org.pb.bin

const (
	scaleTestNamespaceCount = 50 // test with many namespaces
)

//nolint:maintidx // large by design: one container shared across many subtests
func TestSingleOrgScenarios(t *testing.T) {
	endpoints := setupSingleOrgAdmin(t)

	t.Logf("endpoints: %v", endpoints)

	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	require.NoError(t, err)

	// endorser MSP configuration
	org1MspPath := filepath.Join(testdata, "crypto", "peerOrganizations", "Org1", "users", "endorser@org1.com", "msp")
	org1Config := generateConfigFile(t, "Org1MSP", org1MspPath, endpoints)

	configArg := "--config=" + org1Config

	t.Run("create_new_namespace_msp", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "hello_msp", Version: 0}

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

	t.Run("create_new_namespace_threshold", func(t *testing.T) {
		t.Parallel()

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

		expectedNs := Namespace{Name: "hello_threshold", Version: 0}

		// we expect no namespaces
		stdOut, err := fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, expectedNs)

		// create namespace hello
		_, err = fxconfig(t, "namespace", "create", expectedNs.Name,
			configArg, "--policy=threshold:"+thresholdKeyPath, endorseArg, submitArg)
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

	t.Run("create_output_only", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		ns := "s1_output_only"

		_, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, "--output="+txFile)
		require.NoError(t, err)

		data, err := os.ReadFile(txFile)
		require.NoError(t, err)
		_, tx, err := (&cliio.JSONCodec{}).Decode(data)
		require.NoError(t, err)
		require.Empty(t, tx.Endorsements)

		stdOut, err := fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, Namespace{Name: ns})
	})

	t.Run("create_endorse_output_only", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "endorsed.json")
		ns := "s2_endorse_output_only"

		_, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, endorseArg, "--output="+txFile)
		require.NoError(t, err)

		data, err := os.ReadFile(txFile)
		require.NoError(t, err)
		_, tx, err := (&cliio.JSONCodec{}).Decode(data)
		require.NoError(t, err)
		require.Len(t, tx.Endorsements, 1)

		stdOut, err := fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, Namespace{Name: ns})
	})

	t.Run("create_endorse_submit", func(t *testing.T) {
		t.Parallel()

		ns := "s3_endorse_submit"
		expectedNs := Namespace{Name: ns, Version: 0}

		_, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, endorseArg, submitArg)
		require.NoError(t, err)

		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)
	})

	t.Run("create_endorse_submit_wait", func(t *testing.T) {
		t.Parallel()

		ns := "s4_endorse_submit_wait"
		expectedNs := Namespace{Name: ns, Version: 0}

		stdOut, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, endorseArg, submitArg, waitArg)
		require.NoError(t, err)
		require.Contains(t, stdOut, "Transaction status: COMMITTED", "stdout: %s", stdOut)

		stdOut, err = fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.Contains(t, nss, expectedNs)
	})

	t.Run("create_output_then_tx_endorse_submit", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		endorsedFile := filepath.Join(tmpDir, "endorsed.json")
		ns := "s5_tx_endorse_submit"
		expectedNs := Namespace{Name: ns, Version: 0}

		_, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, "--output="+txFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, configArg, "--output="+endorsedFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "submit", endorsedFile, configArg)
		require.NoError(t, err)

		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, expectedNs)
		}, eventuallyTimeout, eventuallyTick)
	})

	t.Run("create_output_then_tx_endorse_submit_wait", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		endorsedFile := filepath.Join(tmpDir, "endorsed.json")
		ns := "s6_tx_endorse_submit_wait"
		expectedNs := Namespace{Name: ns, Version: 0}

		_, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, "--output="+txFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, configArg, "--output="+endorsedFile)
		require.NoError(t, err)

		stdOut, err := fxconfig(t, "tx", "submit", endorsedFile, configArg, "--wait")
		require.NoError(t, err)
		require.Contains(t, stdOut, "Transaction status: COMMITTED", "stdout: %s", stdOut)

		stdOut, err = fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.Contains(t, nss, expectedNs)
	})

	t.Run("update_endorse_submit_wait", func(t *testing.T) {
		t.Parallel()

		ns := "s7_update_endorse_submit_wait"

		_, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, endorseArg, submitArg, waitArg)
		require.NoError(t, err)

		stdOut, err := fxconfig(t,
			"namespace", "update", ns, configArg, policyArg, "--version=0", endorseArg, submitArg, waitArg)
		require.NoError(t, err)
		require.Contains(t, stdOut, "Transaction status: COMMITTED", "stdout: %s", stdOut)

		stdOut, err = fxconfig(t, "namespace", "list", configArg)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.Contains(t, nss, Namespace{Name: ns, Version: 1})
	})

	t.Run("update_output_then_tx_endorse_submit", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		updateFile := filepath.Join(tmpDir, "update.json")
		endorsedFile := filepath.Join(tmpDir, "endorsed.json")
		ns := "s8_update_tx_pipeline"

		_, err := fxconfig(t, "namespace", "create", ns, configArg, policyArg, endorseArg, submitArg, waitArg)
		require.NoError(t, err)

		_, err = fxconfig(t, "namespace", "update", ns, configArg, policyArg, "--version=0", "--output="+updateFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", updateFile, configArg, "--output="+endorsedFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "submit", endorsedFile, configArg)
		require.NoError(t, err)

		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", configArg)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, Namespace{Name: ns, Version: 1})
		}, eventuallyTimeout, eventuallyTick)
	})
}
