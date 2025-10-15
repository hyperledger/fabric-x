/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/


package namespace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("valid namespace ID", func(t *testing.T) {
		t.Parallel()
		nsCfg := NsConfig{NamespaceID: "1"}
		err := validateConfig(nsCfg)
		require.NoError(t, err, "expected no error for valid namespace ID")
	})

	t.Run("empty namespace ID", func(t *testing.T) {
		t.Parallel()
		nsCfg := NsConfig{NamespaceID: ""}
		err := validateConfig(nsCfg)
		require.Error(t, err, "expected error for empty namespace ID")
	})

	t.Run("invalid namespace ID", func(t *testing.T) {
		t.Parallel()
		nsCfg := NsConfig{NamespaceID: "invalid namespace"}
		err := validateConfig(nsCfg)
		require.Error(t, err, "expected error for invalid namespace ID")
	})
}
