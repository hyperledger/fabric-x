package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedErr error
	}{
		{
			name:        "nil error",
			err:         nil,
			expectedErr: nil,
		},
		{
			name:        "insufficient funds error",
			err:         errors.New("fabric token error: insufficient funds in wallet"),
			expectedErr: ErrInsufficientFunds,
		},
		{
			name:        "counterparty account not found error",
			err:         errors.New("recipient [alice] not found on remote node"),
			expectedErr: ErrCounterpartyAccountNotFound,
		},
		{
			name:        "connection error",
			err:         fmt.Errorf("rpc error: failed to dial remote node: connection refused"),
			expectedErr: ErrConnectionError,
		},
		{
			name:        "unrelated error",
			err:         errors.New("unknown error occurred"),
			expectedErr: errors.New("unknown error occurred"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappedErr := classifyError(tt.err)

			if tt.expectedErr == nil {
				assert.NoError(t, mappedErr)
				return
			}

			if tt.name == "unrelated error" {
				assert.Equal(t, tt.expectedErr.Error(), mappedErr.Error())
				return
			}

			assert.ErrorIs(t, mappedErr, tt.expectedErr)
			assert.Contains(t, mappedErr.Error(), tt.err.Error(), "mapped error should wrap the original error context")
		})
	}
}
