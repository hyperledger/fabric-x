// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import "testing"

func TestValidateSnapshot_Success(t *testing.T) {
	t.Parallel()

	s := Snapshot{
		Channel: "mychannel",
		Namespaces: map[string]Namespace{
			"asset": {
				"a1": "v1",
				"a2": "v2",
			},
		},
	}

	if err := ValidateSnapshot(s); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestValidateSnapshot_EmptyChannel(t *testing.T) {
	t.Parallel()

	s := Snapshot{
		Channel: "",
		Namespaces: map[string]Namespace{
			"asset": {"a1": "v1"},
		},
	}

	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("expected error for empty channel")
	}
}

func TestValidateSnapshot_NoNamespaces(t *testing.T) {
	t.Parallel()

	s := Snapshot{
		Channel:    "mychannel",
		Namespaces: map[string]Namespace{},
	}

	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("expected error for no namespaces")
	}
}

func TestValidateSnapshot_EmptyNamespace(t *testing.T) {
	t.Parallel()

	s := Snapshot{
		Channel: "mychannel",
		Namespaces: map[string]Namespace{
			"asset": {},
		},
	}

	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("expected error for empty namespace")
	}
}

func TestValidateSnapshot_EmptyNamespaceName(t *testing.T) {
	t.Parallel()

	s := Snapshot{
		Channel: "mychannel",
		Namespaces: map[string]Namespace{
			"": {"a1": "v1"},
		},
	}

	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("expected error for empty namespace name")
	}
}

func TestValidateSnapshot_EmptyKey(t *testing.T) {
	t.Parallel()

	s := Snapshot{
		Channel: "mychannel",
		Namespaces: map[string]Namespace{
			"asset": {
				"": "v1",
			},
		},
	}

	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("expected error for empty key")
	}
}
