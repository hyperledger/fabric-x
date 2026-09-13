/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldExcludeNamespace(t *testing.T) {
	t.Parallel()

	excluded := []string{
		"lscc",
		"_lifecycle",
		"cscc",
		"qscc",
		"escc",
		"vscc",
		"$$h$$mycc",         // PDC hash namespace
		"$$p$$mycc$$coll",   // PDC private state
		"mycc$$implicitorg", // implicit org collection
	}
	for _, ns := range excluded {
		require.True(t, ShouldExcludeNamespace(ns), "expected %q to be excluded", ns)
	}

	included := []string{
		"mycc",
		"token",
		"basic",
		"fabcar",
	}
	for _, ns := range included {
		require.False(t, ShouldExcludeNamespace(ns), "expected %q to be included", ns)
	}
}

func TestShouldExcludeKey(t *testing.T) {
	t.Parallel()

	require.True(t, ShouldExcludeKey("\x00collectionMyPDC"))
	require.False(t, ShouldExcludeKey("normalKey"))
	require.False(t, ShouldExcludeKey("asset1"))
}
