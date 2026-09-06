/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import "strings"

// systemChaincodes is the set of Fabric system chaincode namespaces that
// have no equivalent in Fabric-X and must be excluded from migration.
var systemChaincodes = map[string]struct{}{
	"lscc":        {},
	"_lifecycle":  {},
	"cscc":        {},
	"qscc":        {},
	"escc":        {},
	"vscc":        {},
}

// ShouldExcludeNamespace returns true if the given namespace should be
// dropped during migration.
//
// Excluded namespaces:
//   - Fabric system chaincodes (lscc, _lifecycle, cscc, qscc, escc, vscc)
//   - PDC hash namespaces  — keys starting with "$$h$$"
//   - PDC private state    — keys starting with "$$p$$"
//   - Implicit org collections — contain "$$"
func ShouldExcludeNamespace(ns string) bool {
	if _, ok := systemChaincodes[ns]; ok {
		return true
	}
	// PDC hash and private state namespaces carry "$$" markers
	if strings.Contains(ns, "$$") {
		return true
	}
	return false
}

// ShouldExcludeKey returns true if an individual key within a namespace
// should be dropped.
//
// Even in non-PDC namespaces, Fabric may embed PDC-related metadata keys.
func ShouldExcludeKey(key string) bool {
	// Fabric embeds PDC hash keys inside public namespaces using these prefixes
	if strings.HasPrefix(key, "\x00collection") {
		return true
	}
	return false
}
