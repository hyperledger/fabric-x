// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package mapping contains a minimal proof-of-concept for deterministic
// namespace transformation from Fabric channel-scoped namespaces into
// Fabric-X global namespace identifiers.
//
// The package intentionally uses a simple concatenation-based mapping
// (channel.namespace). This design is chosen for the POC to expose
// ambiguity and collision risks that a production migration pipeline
// would need to address. For example, the pairs (channel="a.b", namespace="c")
// and (channel="a", namespace="b.c") both map to "a.b.c" with the current
// scheme. This behaviour is deliberate for demonstration purposes.
//
// Possible future approaches to avoid such collisions include: escaping dots,
// length-prefixed encoding, hashing, or a structured encoding format.
// These are noted for future work but are NOT implemented here to keep the
// POC minimal and focused.
package mapping

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// ErrNamespaceCollision is returned when two different source namespaces
// map to the same Fabric-X namespace identifier.
var ErrNamespaceCollision = errors.New("namespace collision")

// KV is a simple key/value pair used in the mock snapshot model.
type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NamespaceState represents state for a single namespace in a channel snapshot.
type NamespaceState struct {
	Namespace string `json:"namespace"`
	KVs       []KV   `json:"kvs"`
}

// ChannelSnapshot is a minimal representation of a channel-scoped snapshot.
type ChannelSnapshot struct {
	Channel    string           `json:"channel"`
	Namespaces []NamespaceState `json:"namespaces"`
}

// FabricXState is the target structure produced by the POC transform.
type FabricXState struct {
	Namespace string `json:"namespace"`
	KVs       []KV   `json:"kvs"`
}

// MapNamespace deterministically maps a channel+namespace to a Fabric-X namespace.
// NOTE: This is intentionally a simple concatenation and therefore collision-prone.
// The POC aims to demonstrate this risk rather than silently hiding it.
// Do not use this as-is in production; consider safer encodings (see package doc).
func MapNamespace(channel, namespace string) string {
	return fmt.Sprintf("%s.%s", channel, namespace)
}

// Transform converts a single ChannelSnapshot into FabricXState entries.
// It merges duplicate namespace entries within the same channel and
// produces deterministic ordering of namespaces and keys.
func Transform(snapshot ChannelSnapshot) ([]FabricXState, error) {
	accum := map[string]FabricXState{}

	for _, ns := range snapshot.Namespaces {
		mapped := MapNamespace(snapshot.Channel, ns.Namespace)
		cur := accum[mapped]
		if cur.KVs == nil {
			cur = FabricXState{Namespace: mapped, KVs: make([]KV, 0, len(ns.KVs))}
		}
		cur.KVs = append(cur.KVs, ns.KVs...)
		accum[mapped] = cur
	}

	// Sort namespaces and keys to ensure deterministic output across runs.
	// Determinism is important so the same input always yields identical output.
	states := make([]FabricXState, 0, len(accum))
	for _, s := range accum {
		states = append(states, s)
	}

	// Use slices.SortFunc for clearer sorting intent.
	slices.SortFunc(states, func(a, b FabricXState) int { return compareString(a.Namespace, b.Namespace) })

	for i := range states {
		slices.SortFunc(states[i].KVs, func(a, b KV) int {
			if a.Key == b.Key {
				return compareString(a.Value, b.Value)
			}
			return compareString(a.Key, b.Key)
		})
	}

	return states, nil
}

// TransformMany converts multiple ChannelSnapshots and detects namespace collisions
// that occur when different source channel+namespace pairs map to the same target.
//
//nolint:gocognit // complexity acceptable for small POC mapping logic
func TransformMany(snapshots []ChannelSnapshot) ([]FabricXState, error) {
	mappedSource := map[string]string{} // mapped -> source identifier
	accum := map[string]FabricXState{}

	for _, snap := range snapshots {
		for _, ns := range snap.Namespaces {
			mapped := MapNamespace(snap.Channel, ns.Namespace)
			src := fmt.Sprintf("%s:%s", snap.Channel, ns.Namespace)
			if prev, ok := mappedSource[mapped]; ok {
				if prev != src {
					return nil, fmt.Errorf("%w: %s vs %s", ErrNamespaceCollision, prev, src)
				}
			} else {
				mappedSource[mapped] = src
			}

			cur := accum[mapped]
			if cur.KVs == nil {
				cur = FabricXState{Namespace: mapped, KVs: make([]KV, 0, len(ns.KVs))}
			}
			cur.KVs = append(cur.KVs, ns.KVs...)
			accum[mapped] = cur
		}
	}

	// Sort namespaces and keys to ensure deterministic output across runs.
	// Determinism is important so the same input always yields identical output.
	states := make([]FabricXState, 0, len(accum))
	for _, s := range accum {
		states = append(states, s)
	}

	// Sort namespaces deterministically.
	slices.SortFunc(states, func(a, b FabricXState) int { return compareString(a.Namespace, b.Namespace) })

	for i := range states {
		// Sort keys within a namespace so output order is stable.
		slices.SortFunc(states[i].KVs, func(a, b KV) int {
			if a.Key == b.Key {
				return compareString(a.Value, b.Value)
			}
			return compareString(a.Key, b.Key)
		})
	}

	return states, nil
}

// WriteJSON returns an indented JSON representation of the states.
func WriteJSON(states []FabricXState) ([]byte, error) {
	return json.MarshalIndent(states, "", "  ")
}

// compareString is a small helper returning -1/0/1 for a < b / == / >.
func compareString(a, b string) int {
	if a == b {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}
