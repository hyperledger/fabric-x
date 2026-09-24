/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transaction

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// TestCreateNamespacesTx tests the CreateNamespacesTx function.
func TestCreateNamespacesTx(t *testing.T) {
	t.Parallel()

	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_ThresholdRule{
			ThresholdRule: &applicationpb.ThresholdRule{
				Scheme:    "ECDSA",
				PublicKey: []byte("test-public-key"),
			},
		},
	}

	tests := []struct {
		name          string
		nsPolicy      *applicationpb.NamespacePolicy
		nsID          string
		nsVersion     int
		metaNsVersion uint64
		description   string
	}{
		{
			name:          "create new namespace (version -1)",
			nsPolicy:      nsPolicy,
			nsID:          "new-namespace",
			nsVersion:     -1,
			metaNsVersion: 0,
			description:   "Should create transaction for new namespace",
		},
		{
			name:          "update existing namespace (version 0)",
			nsPolicy:      nsPolicy,
			nsID:          "existing-namespace",
			nsVersion:     0,
			metaNsVersion: 1,
			description:   "Should create transaction for namespace update",
		},
		{
			name:          "update existing namespace (version 5)",
			nsPolicy:      nsPolicy,
			nsID:          "existing-namespace",
			nsVersion:     5,
			metaNsVersion: 9,
			description:   "Should create transaction for namespace update with higher version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CreateNamespacesTx(tt.nsPolicy, tt.nsID, tt.nsVersion, tt.metaNsVersion)

			require.NotNil(t, result, tt.description)
			require.Len(t, result.Namespaces, 1, "Should have one namespace entry")

			ns := result.Namespaces[0]
			require.Equal(t, "_meta", ns.NsId, "Should target meta-namespace")
			require.Equal(t, tt.metaNsVersion, ns.NsVersion, "Meta-namespace version should match input")
			require.Len(t, ns.ReadWrites, 1, "Should have one read-write entry")

			rw := ns.ReadWrites[0]
			require.Equal(t, []byte(tt.nsID), rw.Key, "Key should be namespace ID")
			require.NotEmpty(t, rw.Value, "Value should contain serialized policy")

			// Verify version is set correctly
			if tt.nsVersion >= 0 {
				require.NotNil(t, rw.Version, "Version should be set for updates")
				require.Equal(t, uint64(tt.nsVersion), *rw.Version, "Version should match input")
			} else {
				require.Nil(t, rw.Version, "Version should be nil for creates")
			}
		})
	}
}
