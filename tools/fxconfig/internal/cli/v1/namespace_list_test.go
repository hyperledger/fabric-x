/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
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
}

func TestNewListCommandRun(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	namespaces := []app.NamespaceQueryResult{
		{NsID: "ns1", Version: 1, Policy: []byte{0xde, 0xad}},
		{NsID: "ns2", Version: 2, Policy: []byte{0xbe, 0xef}},
	}
	mockApp.On("ListNamespaces", mock.Anything).Return(namespaces, nil)

	var out, errOut bytes.Buffer
	printer := cliio.NewCLIPrinter(&out, &errOut, cliio.FormatTable)
	cmd := newNsListCommand(&CLIContext{App: mockApp, Printer: printer})

	err := cmd.RunE(cmd, nil)

	require.NoError(t, err)
	output := out.String()
	require.Contains(t, output, "2 total")
	require.Contains(t, output, "ns1")
	require.Contains(t, output, "ns2")
	mockApp.AssertExpectations(t)
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
