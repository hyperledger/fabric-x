/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
)

func makeMetaNamespaceQueryProvider(version uint64) *provider.Provider[adapters.QueryClient, *config.QueriesConfig] {
	return makeQueryProvider(&mockQueryClient{policies: &applicationpb.NamespacePolicies{Policies: []*applicationpb.PolicyItem{
		{Namespace: "_meta", Version: version, Policy: []byte("meta-policy")},
	}}}, nil)
}

func TestDeployNamespaceInputValidate(t *testing.T) {
	t.Parallel()

	vctx := fakeValidationContext()

	tests := []struct {
		name        string
		input       DeployNamespaceInput
		expectError bool
	}{
		{
			name: "valid msp policy",
			input: DeployNamespaceInput{
				NsID:    "testns",
				Version: -1,
				Policy:  PolicyConfig{Type: mspPolicyType, MSP: &MSPPolicyConfig{Expression: "OR('Org1MSP.member')"}},
			},
		},
		{
			name: "invalid namespace id with hyphen",
			input: DeployNamespaceInput{
				NsID:    "test-ns",
				Version: -1,
				Policy:  PolicyConfig{Type: mspPolicyType, MSP: &MSPPolicyConfig{Expression: "OR('Org1MSP.member')"}},
			},
			expectError: true,
		},
		{
			name: "invalid version",
			input: DeployNamespaceInput{
				NsID:    "testns",
				Version: -2,
				Policy:  PolicyConfig{Type: mspPolicyType, MSP: &MSPPolicyConfig{Expression: "OR('Org1MSP.member')"}},
			},
			expectError: true,
		},
		{
			name: "unknown policy type",
			input: DeployNamespaceInput{
				NsID:    "testns",
				Version: -1,
				Policy:  PolicyConfig{Type: "unknown"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate(vctx)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDeployNamespace_CreateOnly(t *testing.T) {
	t.Parallel()

	a := &AdminApp{Validators: fakeValidationContext(), QueryProvider: makeMetaNamespaceQueryProvider(0)}
	input := &DeployNamespaceInput{
		NsID:    "testns",
		Version: -1,
		Policy:  PolicyConfig{Type: mspPolicyType, MSP: &MSPPolicyConfig{Expression: "OR('Org1MSP.member')"}},
		Endorse: false,
		Submit:  false,
	}

	out, status, err := a.DeployNamespace(t.Context(), input)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotEmpty(t, out.TxID)
	require.NotNil(t, out.Tx)
	require.Equal(t, UnknownStatus, status)
}

func TestDeployNamespace_CreateOnly_NoMetaNamespacePresent(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		Validators:    fakeValidationContext(),
		QueryProvider: makeQueryProvider(&mockQueryClient{policies: &applicationpb.NamespacePolicies{}}, nil),
	}
	input := &DeployNamespaceInput{
		NsID:    "testns",
		Version: -1,
		Policy:  PolicyConfig{Type: mspPolicyType, MSP: &MSPPolicyConfig{Expression: "OR('Org1MSP.member')"}},
		Endorse: false,
		Submit:  false,
	}

	out, status, err := a.DeployNamespace(t.Context(), input)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotEmpty(t, out.TxID)
	require.NotNil(t, out.Tx)
	require.Len(t, out.Tx.Namespaces, 1)
	require.Equal(t, uint64(0), out.Tx.Namespaces[0].NsVersion)
	require.Equal(t, UnknownStatus, status)
}

func TestDeployNamespace_ValidationError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{Validators: fakeValidationContext(), QueryProvider: makeMetaNamespaceQueryProvider(0)}
	input := &DeployNamespaceInput{
		NsID:    "testns",
		Version: -2, // invalid
		Policy:  PolicyConfig{Type: mspPolicyType, MSP: &MSPPolicyConfig{Expression: "OR('Org1MSP.member')"}},
	}

	_, _, err := a.DeployNamespace(t.Context(), input)
	require.Error(t, err)
}

func validDeployInput() *DeployNamespaceInput {
	return &DeployNamespaceInput{
		NsID:    "testns",
		Version: -1,
		Policy:  PolicyConfig{Type: mspPolicyType, MSP: &MSPPolicyConfig{Expression: "OR('Org1MSP.member')"}},
	}
}

func TestDeployNamespace_EndorseOnly(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		Validators:    fakeValidationContext(),
		QueryProvider: makeMetaNamespaceQueryProvider(0),
		MspProvider:   makeMSPProvider(&testSigningIdentity{}, nil),
	}
	input := validDeployInput()
	input.Endorse = true
	input.Submit = false

	out, status, err := a.DeployNamespace(t.Context(), input)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotEmpty(t, out.TxID)
	require.NotNil(t, out.Tx)
	require.NotEmpty(t, out.Tx.Endorsements, "endorsed tx should carry endorsements")
	require.Equal(t, UnknownStatus, status)
}

func TestDeployNamespace_EndorseError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		Validators:    fakeValidationContext(),
		QueryProvider: makeMetaNamespaceQueryProvider(0),
		MspProvider:   makeMSPProvider(nil, errors.New("msp not configured")),
	}
	input := validDeployInput()
	input.Endorse = true

	_, _, err := a.DeployNamespace(t.Context(), input)
	require.Error(t, err)
}

func TestDeployNamespace_EndorseAndSubmit(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		Validators:      fakeValidationContext(),
		QueryProvider:   makeMetaNamespaceQueryProvider(0),
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{}, nil),
	}
	input := validDeployInput()
	input.Endorse = true
	input.Submit = true
	input.Wait = false

	out, status, err := a.DeployNamespace(t.Context(), input)
	require.NoError(t, err)
	require.Nil(t, out)
	require.Equal(t, UnknownStatus, status)
}

func TestDeployNamespace_EndorseAndSubmitError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		Validators:      fakeValidationContext(),
		QueryProvider:   makeMetaNamespaceQueryProvider(0),
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{broadcastErr: errors.New("orderer unavailable")}, nil),
	}
	input := validDeployInput()
	input.Endorse = true
	input.Submit = true
	input.Wait = false

	_, _, err := a.DeployNamespace(t.Context(), input)
	require.Error(t, err)
}

func TestDeployNamespace_EndorseAndSubmitWithWait(t *testing.T) {
	t.Parallel()

	const expectedStatus = 1

	a := &AdminApp{
		Validators:           fakeValidationContext(),
		QueryProvider:        makeMetaNamespaceQueryProvider(0),
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(&mockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{status: expectedStatus}, nil),
	}
	input := validDeployInput()
	input.Endorse = true
	input.Submit = true
	input.Wait = true

	out, status, err := a.DeployNamespace(t.Context(), input)
	require.NoError(t, err)
	require.Nil(t, out)
	require.Equal(t, expectedStatus, status)
}

func TestDeployNamespace_EndorseAndSubmitWithWaitError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		Validators:           fakeValidationContext(),
		QueryProvider:        makeMetaNamespaceQueryProvider(0),
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(&mockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(nil, errors.New("notification service unavailable")),
	}
	input := validDeployInput()
	input.Endorse = true
	input.Submit = true
	input.Wait = true

	_, _, err := a.DeployNamespace(t.Context(), input)
	require.Error(t, err)
}
