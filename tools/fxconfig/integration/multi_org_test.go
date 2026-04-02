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

const policyAnd2of2 = "--policy=AND('Org1MSP.member', 'Org2MSP.member')"

func TestMultiOrgScenarios(t *testing.T) {
	endpoints := setupMultiOrgAdmin(t)

	t.Logf("endpoints: %v", endpoints)

	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	require.NoError(t, err)

	org1MspPath := filepath.Join(testdata, "crypto", "peerOrganizations", "Org1", "users", "endorser@org1.com", "msp")
	org2MspPath := filepath.Join(testdata, "crypto", "peerOrganizations", "Org2", "users", "endorser@org2.com", "msp")

	org1Config := generateConfigFile(t, "Org1MSP", org1MspPath, endpoints)
	org2Config := generateConfigFile(t, "Org2MSP", org2MspPath, endpoints)

	t.Run("multi_org_create_2of2", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		txOrg1 := filepath.Join(tmpDir, "tx_org1.json")
		txOrg2 := filepath.Join(tmpDir, "tx_org2.json")
		merged := filepath.Join(tmpDir, "merged.json")
		ns := "m1_create_2of2"
		policy := policyAnd2of2

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

	t.Run("multi_org_create_2of2_with_wait", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		txOrg1 := filepath.Join(tmpDir, "tx_org1.json")
		txOrg2 := filepath.Join(tmpDir, "tx_org2.json")
		merged := filepath.Join(tmpDir, "merged.json")
		ns := "m2_create_2of2_wait"
		policy := policyAnd2of2

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

	t.Run("multi_org_create_insufficient_sigs", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		txOrg1 := filepath.Join(tmpDir, "tx_org1.json")
		ns := "m3_insufficient_sigs"
		policy := policyAnd2of2

		_, err := fxconfig(t, "namespace", "create", ns, "--config="+org1Config, policy, "--output="+txFile)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txFile, "--config="+org1Config, "--output="+txOrg1)
		require.NoError(t, err)

		// submit with --wait so we know the transaction outcome before checking the list
		stdOut, err := fxconfig(t, "tx", "submit", "--wait", txOrg1, "--config="+org1Config)
		if err == nil {
			require.NotContains(t, stdOut, "Transaction status: COMMITTED",
				"expected non-COMMITTED status for insufficient endorsements, stdout: %s", stdOut)
		}

		// since --wait synchronized on the transaction outcome, a direct list check is sufficient
		stdOut, err = fxconfig(t, "namespace", "list", "--config="+org1Config)
		require.NoError(t, err)
		nss, err := parseNamespaceList(stdOut)
		require.NoError(t, err)
		require.NotContains(t, nss, Namespace{Name: ns, Version: 0})
	})

	t.Run("multi_org_create_2of3_threshold", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txFile := filepath.Join(tmpDir, "tx.json")
		txOrg1 := filepath.Join(tmpDir, "tx_org1.json")
		txOrg2 := filepath.Join(tmpDir, "tx_org2.json")
		merged := filepath.Join(tmpDir, "merged.json")
		ns := "m4_create_2of3_threshold"
		policy := "--policy=OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member')"

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

	t.Run("multi_org_update", func(t *testing.T) {
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
		ns := "m5_update"
		policy := policyAnd2of2

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

	t.Run("multi_org_merge_mismatched_txids", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		txA := filepath.Join(tmpDir, "txA.json")
		txB := filepath.Join(tmpDir, "txB.json")
		txAEndorsed := filepath.Join(tmpDir, "txA_endorsed.json")
		txBEndorsed := filepath.Join(tmpDir, "txB_endorsed.json")
		policy := policyAnd2of2

		// Create two different namespaces
		_, err := fxconfig(t, "namespace", "create", "m6_ns_a", "--config="+org1Config, policy, "--output="+txA)
		require.NoError(t, err)

		_, err = fxconfig(t, "namespace", "create", "m6_ns_b", "--config="+org1Config, policy, "--output="+txB)
		require.NoError(t, err)

		// Endorse each independently
		_, err = fxconfig(t, "tx", "endorse", txA, "--config="+org1Config, "--output="+txAEndorsed)
		require.NoError(t, err)

		_, err = fxconfig(t, "tx", "endorse", txB, "--config="+org2Config, "--output="+txBEndorsed)
		require.NoError(t, err)

		// Merge should fail with mismatched txIDs
		_, err = fxconfig(t, "tx", "merge", txAEndorsed, txBEndorsed)
		require.Error(t, err)
		require.Contains(t, err.Error(), "txID")
	})
}
