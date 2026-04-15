/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const policyAnd2of2TLS = "--policy=AND('Org1MSP.member', 'Org2MSP.member')"

func TestSingleOrgTLSScenarios(t *testing.T) {
	endpoints := setupSingleOrgAdminWithTLS(t)

	t.Logf("TLS endpoints: %v", endpoints)

	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	require.NoError(t, err)

	// endorser MSP configuration
	org1MspPath := filepath.Join(testdata, "crypto", "peerOrganizations", "Org1", "users", "endorser@org1.com", "msp")
	org1Config := generateConfigFileWithTLS(t, "Org1MSP", org1MspPath, endpoints, true)

	configArg := "--config=" + org1Config

	t.Run("create_new_namespace_msp_tls", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "hello_msp_tls", Version: 0}

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

	t.Run("create_new_namespace_with_wait_tls", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "hello_with_wait_tls", Version: 0}

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

	t.Run("update_namespace_tls", func(t *testing.T) {
		t.Parallel()

		expectedNs := Namespace{Name: "ns1_tls", Version: 0}

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
		expectedNs = Namespace{Name: "ns1_tls", Version: 1}
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
	})
}

func TestMultiOrgTLSScenarios(t *testing.T) {
	endpoints := setupMultiOrgAdminWithTLS(t)

	t.Logf("TLS endpoints: %v", endpoints)

	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	require.NoError(t, err)

	org1MspPath := filepath.Join(testdata, "crypto", "peerOrganizations", "Org1", "users", "endorser@org1.com", "msp")
	org2MspPath := filepath.Join(testdata, "crypto", "peerOrganizations", "Org2", "users", "endorser@org2.com", "msp")

	org1Config := generateConfigFileWithTLS(t, "Org1MSP", org1MspPath, endpoints, true)
	org2Config := generateConfigFileWithTLS(t, "Org2MSP", org2MspPath, endpoints, true)

	t.Run("multi_org_create_2of2_tls", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		txOrg1 := filepath.Join(tmpDir, "tx_org1.json")
		txOrg2 := filepath.Join(tmpDir, "tx_org2.json")
		merged := filepath.Join(tmpDir, "merged.json")
		ns := "m1_create_2of2_tls"
		policy := policyAnd2of2TLS

		_, err := fxconfig(t, "namespace", "create", ns, "--config="+org1Config, policy, "--output="+txFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, "--config="+org1Config, "--output="+txOrg1)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, "--config="+org2Config, "--output="+txOrg2)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "merge", txOrg1, txOrg2, "--output="+merged)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "submit", merged, "--config="+org1Config)
		require.NoError(t, err)

		assert.EventuallyWithT(t, func(ct *assert.CollectT) {
			stdOut, err := fxconfig(t, "namespace", "list", "--config="+org1Config)
			require.NoError(ct, err)
			nss, err := parseNamespaceList(stdOut)
			require.NoError(ct, err)
			require.Contains(ct, nss, Namespace{Name: ns, Version: 0})
		}, eventuallyTimeout, eventuallyTick)
	})

	t.Run("multi_org_create_2of2_with_wait_tls", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		txOrg1 := filepath.Join(tmpDir, "tx_org1.json")
		txOrg2 := filepath.Join(tmpDir, "tx_org2.json")
		merged := filepath.Join(tmpDir, "merged.json")
		ns := "m2_create_2of2_wait_tls"
		policy := policyAnd2of2TLS

		_, err := fxconfig(t, "namespace", "create", ns, "--config="+org1Config, policy, "--output="+txFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, "--config="+org1Config, "--output="+txOrg1)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, "--config="+org2Config, "--output="+txOrg2)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "merge", txOrg1, txOrg2, "--output="+merged)
		require.NoError(t, err)

		stdOut, err := fxconfig(t, "tx", "submit", "--wait", merged, "--config="+org1Config)
		require.NoError(t, err)
		require.Contains(t, stdOut, "Transaction status: COMMITTED", "stdout: %s", stdOut)

		stdOut, err = fxconfig(t, "namespace", "list", "--config="+org1Config)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.Contains(t, nss, Namespace{Name: ns, Version: 0})
	})

	t.Run("multi_org_update_tls", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		txOrg1 := filepath.Join(tmpDir, "tx_org1.json")
		txOrg2 := filepath.Join(tmpDir, "tx_org2.json")
		merged := filepath.Join(tmpDir, "merged.json")
		updateFile := filepath.Join(tmpDir, "update.json")
		updateOrg1 := filepath.Join(tmpDir, "update_org1.json")
		updateOrg2 := filepath.Join(tmpDir, "update_org2.json")
		updateMerged := filepath.Join(tmpDir, "update_merged.json")
		ns := "m5_update_tls"
		policy := policyAnd2of2TLS

		// Create the namespace first using the 2-of-2 pipeline with --wait
		_, err := fxconfig(t, "namespace", "create", ns, "--config="+org1Config, policy, "--output="+txFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, "--config="+org1Config, "--output="+txOrg1)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, "--config="+org2Config, "--output="+txOrg2)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "merge", txOrg1, txOrg2, "--output="+merged)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "submit", "--wait", merged, "--config="+org1Config)
		require.NoError(t, err)

		// Now update the namespace
		_, err = fxconfig(t, "namespace", "update", ns,
			"--config="+org1Config, policy, "--version=0", "--output="+updateFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", updateFile, "--config="+org1Config, "--output="+updateOrg1)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", updateFile, "--config="+org2Config, "--output="+updateOrg2)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "merge", updateOrg1, updateOrg2, "--output="+updateMerged)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "submit", "--wait", updateMerged, "--config="+org1Config)
		require.NoError(t, err)

		stdOut, err := fxconfig(t, "namespace", "list", "--config="+org1Config)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.Contains(t, nss, Namespace{Name: ns, Version: 1})
	})
}
