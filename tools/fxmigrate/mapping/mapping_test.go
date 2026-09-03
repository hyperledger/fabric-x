// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mapping

import (
	"bytes"
	"errors"
	"testing"
)

func TestDeterminism(t *testing.T) {
	t.Parallel()
	snap := ChannelSnapshot{
		Channel: "chanA",
		Namespaces: []NamespaceState{
			{Namespace: "ns1", KVs: []KV{{Key: "k1", Value: "v1"}, {Key: "k2", Value: "v2"}}},
			{Namespace: "ns2", KVs: []KV{{Key: "a", Value: "1"}}},
		},
	}

	s1, err := Transform(snap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b1, err := WriteJSON(s1)
	if err != nil {
		t.Fatalf("json write failed: %v", err)
	}

	s2, err := Transform(snap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b2, err := WriteJSON(s2)
	if err != nil {
		t.Fatalf("json write failed: %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Fatal("determinism failed: outputs differ")
	}
}

func TestCollisionDetection(t *testing.T) {
	t.Parallel()
	s1 := ChannelSnapshot{Channel: "a.b", Namespaces: []NamespaceState{{Namespace: "c"}}}
	s2 := ChannelSnapshot{Channel: "a", Namespaces: []NamespaceState{{Namespace: "b.c"}}}

	_, err := TransformMany([]ChannelSnapshot{s1, s2})
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	if !errors.Is(err, ErrNamespaceCollision) {
		t.Fatalf("expected ErrNamespaceCollision, got: %v", err)
	}
}

func TestKeyPreservation(t *testing.T) {
	t.Parallel()
	snap := ChannelSnapshot{
		Channel: "chan",
		Namespaces: []NamespaceState{
			{Namespace: "ns", KVs: []KV{{Key: "k1", Value: "v1"}, {Key: "k2", Value: "v2"}}},
		},
	}
	states, err := Transform(snap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	if len(states[0].KVs) != 2 {
		t.Fatalf("expected 2 keys preserved, got %d", len(states[0].KVs))
	}
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("empty namespace", func(t *testing.T) {
		t.Parallel()
		snap := ChannelSnapshot{Channel: "c", Namespaces: []NamespaceState{{Namespace: ""}}}
		states, err := Transform(snap)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(states) != 1 {
			t.Fatalf("expected 1 state, got %d", len(states))
		}
	})

	t.Run("dots in names and unicode and case", func(t *testing.T) {
		t.Parallel()
		snap := ChannelSnapshot{Channel: "Ch.An", Namespaces: []NamespaceState{
			{Namespace: "Ns.One", KVs: []KV{{Key: "K", Value: "v"}}},
			{Namespace: "ünîcøde", KVs: []KV{{Key: "k", Value: "v"}}},
			{Namespace: "dup", KVs: []KV{{Key: "x", Value: "1"}, {Key: "x", Value: "2"}}},
		}}
		states, err := Transform(snap)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// key counts preserved
		totalIn := 0
		for _, ns := range snap.Namespaces {
			totalIn += len(ns.KVs)
		}
		totalOut := 0
		for _, s := range states {
			totalOut += len(s.KVs)
		}
		if totalIn != totalOut {
			t.Fatalf("key preservation mismatch: in=%d out=%d", totalIn, totalOut)
		}
	})
}

func TestDuplicateNamespaceMerge(t *testing.T) {
	t.Parallel()
	snap := ChannelSnapshot{
		Channel: "chanDup",
		Namespaces: []NamespaceState{
			{Namespace: "ns", KVs: []KV{{Key: "b", Value: "2"}, {Key: "a", Value: "1"}}},
			{Namespace: "ns", KVs: []KV{{Key: "c", Value: "3"}, {Key: "a", Value: "0"}}},
		},
	}

	states, err := Transform(snap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 merged namespace, got %d", len(states))
	}
	if len(states[0].KVs) != 4 {
		t.Fatalf("expected 4 keys after merge, got %d", len(states[0].KVs))
	}

	expected := []KV{{Key: "a", Value: "0"}, {Key: "a", Value: "1"}, {Key: "b", Value: "2"}, {Key: "c", Value: "3"}}
	for i, kv := range states[0].KVs {
		if kv != expected[i] {
			t.Fatalf("unexpected ordered kv at %d: got %+v expected %+v", i, kv, expected[i])
		}
	}

	// Determinism: repeated transform should produce identical JSON
	b1, err := WriteJSON(states)
	if err != nil {
		t.Fatalf("json write failed: %v", err)
	}
	states2, err := Transform(snap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b2, err := WriteJSON(states2)
	if err != nil {
		t.Fatalf("json write failed: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("determinism failed for merged namespace: outputs differ")
	}
}
