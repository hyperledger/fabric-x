/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
)

func TestNewListCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newNsListCommand(&CLIContext{App: &testApp{}})

	// Assert
	require.NotNil(t, cmd, "newNsListCommand should return a non-nil command")
	require.Equal(t, "list", cmd.Use, "command use should be 'list'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.RunE, "command should have a RunE function")
	require.NotNil(t, cmd.Flags().Lookup("format"), "command should have a --format flag")
}

func TestNewListCommandRun(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	namespaces := []app.NamespaceQueryResult{
		{NsID: "ns1", Version: 1, Policy: []byte{0xde, 0xad}},
		{NsID: "ns2", Version: 2, Policy: []byte{0xbe, 0xef}},
	}
	mockApp.On("ListNamespaces", mock.Anything).Return(namespaces, nil)

	var out bytes.Buffer
	cmd := newNsListCommand(&CLIContext{App: mockApp})
	cmd.SetOut(&out)

	err := cmd.RunE(cmd, nil)

	require.NoError(t, err)
	output := out.String()
	require.Contains(t, output, "2 total")
	require.Contains(t, output, "ns1")
	require.Contains(t, output, "ns2")
	mockApp.AssertExpectations(t)
}

func TestNewListCommandRunJSON(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	namespaces := []app.NamespaceQueryResult{
		{NsID: "ns1", Version: 1, Policy: []byte{0xde, 0xad}},
		{NsID: "ns2", Version: 2, Policy: []byte{0xbe, 0xef}},
	}
	mockApp.On("ListNamespaces", mock.Anything).Return(namespaces, nil)

	var out bytes.Buffer
	cmd := newNsListCommand(&CLIContext{App: mockApp})
	cmd.SetOut(&out)
	require.NoError(t, cmd.Flags().Set("format", "json"))

	err := cmd.RunE(cmd, nil)

	require.NoError(t, err)

	var results []map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &results))
	require.Len(t, results, 2)
	require.Equal(t, "ns1", results[0]["name"])
	require.Equal(t, "ns2", results[1]["name"])
	require.InDelta(t, float64(1), results[0]["version"], 0)
	require.InDelta(t, float64(2), results[1]["version"], 0)
	require.Contains(t, results[0], "policyString")
	mockApp.AssertExpectations(t)
}

func TestNewListCommandRunYAML(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	namespaces := []app.NamespaceQueryResult{
		{NsID: "myns", Version: 3, Policy: []byte{0xca, 0xfe}},
	}
	mockApp.On("ListNamespaces", mock.Anything).Return(namespaces, nil)

	var out bytes.Buffer
	cmd := newNsListCommand(&CLIContext{App: mockApp})
	cmd.SetOut(&out)
	require.NoError(t, cmd.Flags().Set("format", "yaml"))

	err := cmd.RunE(cmd, nil)

	require.NoError(t, err)
	output := out.String()
	require.Contains(t, output, "myns")
	require.Contains(t, output, "version: 3")
	mockApp.AssertExpectations(t)
}

func TestNewListCommandRunInvalidFormat(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	cmd := newNsListCommand(&CLIContext{App: mockApp})
	require.NoError(t, cmd.Flags().Set("format", "xml"))

	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid --format")
}

func TestNewListCommandRun_AppError(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	mockApp.On("ListNamespaces", mock.Anything).Return(nil, context.DeadlineExceeded)

	cmd := newNsListCommand(&CLIContext{App: mockApp})

	err := cmd.RunE(cmd, nil)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	mockApp.AssertExpectations(t)
}
