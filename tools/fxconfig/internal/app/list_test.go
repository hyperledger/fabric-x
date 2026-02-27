/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

func TestPrintResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		policies       *applicationpb.NamespacePolicies
		expectedOutput []string
		expectedLines  int
	}{
		{
			name: "empty namespace list",
			policies: &applicationpb.NamespacePolicies{
				Policies: []*applicationpb.PolicyItem{},
			},
			expectedOutput: []string{
				"Installed namespaces (0 total):",
			},
			expectedLines: 1, // Header only
		},
		{
			name: "single namespace",
			policies: &applicationpb.NamespacePolicies{
				Policies: []*applicationpb.PolicyItem{
					{
						Namespace: "testns",
						Version:   1,
						Policy:    []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectedOutput: []string{
				"Installed namespaces (1 total):",
				"0) testns: version 1 policy: 01020304",
			},
			expectedLines: 2, // Header + 1 namespace
		},
		{
			name: "multiple namespaces with different versions",
			policies: &applicationpb.NamespacePolicies{
				Policies: []*applicationpb.PolicyItem{
					{
						Namespace: "alpha",
						Version:   1,
						Policy:    []byte{0xaa},
					},
					{
						Namespace: "beta",
						Version:   10,
						Policy:    []byte{0xbb, 0xcc},
					},
					{
						Namespace: "gamma",
						Version:   100,
						Policy:    []byte{0xdd, 0xee, 0xff},
					},
				},
			},
			expectedOutput: []string{
				"Installed namespaces (3 total):",
				"0) alpha: version 1 policy: aa",
				"1) beta: version 10 policy: bbcc",
				"2) gamma: version 100 policy: ddeeff",
			},
			expectedLines: 4, // Header + 3 namespaces
		},
		{
			name: "namespace with empty policy",
			policies: &applicationpb.NamespacePolicies{
				Policies: []*applicationpb.PolicyItem{
					{
						Namespace: "emptypolicy",
						Version:   1,
						Policy:    []byte{},
					},
				},
			},
			expectedOutput: []string{
				"Installed namespaces (1 total):",
				"0) emptypolicy: version 1 policy:",
			},
			expectedLines: 2,
		},
		{
			name: "namespace with long policy",
			policies: &applicationpb.NamespacePolicies{
				Policies: []*applicationpb.PolicyItem{
					{
						Namespace: "longpolicy",
						Version:   5,
						Policy:    []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
					},
				},
			},
			expectedOutput: []string{
				"Installed namespaces (1 total):",
				"0) longpolicy: version 5 policy: 00112233445566778899",
			},
			expectedLines: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			var buf bytes.Buffer

			// Execute
			printResult(&buf, tt.policies)

			// Assert
			output := buf.String()
			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

			assert.Len(t, lines, tt.expectedLines, "output should have expected number of lines")

			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected, "output should contain expected text")
			}
		})
	}
}

func TestPrintResult_OutputFormat(t *testing.T) {
	t.Parallel()

	// Setup
	policies := &applicationpb.NamespacePolicies{
		Policies: []*applicationpb.PolicyItem{
			{
				Namespace: "ns1",
				Version:   1,
				Policy:    []byte{0x01},
			},
			{
				Namespace: "ns2",
				Version:   2,
				Policy:    []byte{0x02},
			},
		},
	}

	var buf bytes.Buffer

	// Execute
	printResult(&buf, policies)

	// Assert
	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Verify header format
	assert.True(t, strings.HasPrefix(lines[0], "Installed namespaces ("), "first line should be header")
	assert.Contains(t, lines[0], "2 total", "header should show correct count")

	// Verify namespace entry format (index) namespace: version X policy: hex)
	for i := 1; i < len(lines)-1; i++ {
		line := lines[i]
		assert.True(t, strings.HasPrefix(line, string(rune('0'+i-1))+")"), "namespace line should start with index")
		assert.Contains(t, line, "version", "namespace line should contain 'version'")
		assert.Contains(t, line, "policy:", "namespace line should contain 'policy:'")
	}

	// Verify last line is a namespace entry (no trailing empty line)
	assert.Contains(t, lines[len(lines)-1], "version", "last line should be a namespace entry")
}

func TestPrintResult_NilPolicies(t *testing.T) {
	t.Parallel()

	// Setup
	var buf bytes.Buffer

	// Execute - should not panic with nil policies
	require.NotPanics(t, func() {
		printResult(&buf, nil)
	})

	// Assert
	output := buf.String()
	assert.Contains(t, output, "Installed namespaces (0 total):", "should handle nil policies gracefully")
}

func TestPrintResult_PolicyHexEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		policy      []byte
		expectedHex string
	}{
		{
			name:        "single byte",
			policy:      []byte{0xff},
			expectedHex: "ff",
		},
		{
			name:        "multiple bytes",
			policy:      []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			expectedHex: "0123456789abcdef",
		},
		{
			name:        "all zeros",
			policy:      []byte{0x00, 0x00, 0x00},
			expectedHex: "000000",
		},
		{
			name:        "all ones",
			policy:      []byte{0xff, 0xff, 0xff},
			expectedHex: "ffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			policies := &applicationpb.NamespacePolicies{
				Policies: []*applicationpb.PolicyItem{
					{
						Namespace: "test",
						Version:   1,
						Policy:    tt.policy,
					},
				},
			}

			var buf bytes.Buffer

			// Execute
			printResult(&buf, policies)

			// Assert
			output := buf.String()
			assert.Contains(t, output, tt.expectedHex, "policy should be hex encoded correctly")
		})
	}
}
