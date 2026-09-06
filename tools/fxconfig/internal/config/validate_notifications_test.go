package config

import (
	"testing"
	"time"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationsConfig_Validate(t *testing.T) {
	t.Parallel()

	validTLS := &TLSConfig{Enabled: new(bool)}
	*validTLS.Enabled = false

	vctx := validation.Context{}

	tests := []struct {
		name          string
		cfg           NotificationsConfig
		expectedError string
	}{
		{
			name: "valid config",
			cfg: NotificationsConfig{
				EndpointServiceConfig: EndpointServiceConfig{
					Address:           "localhost:1234",
					ConnectionTimeout: 5 * time.Second,
					TLS:               validTLS,
				},
				WaitingTimeout: 30 * time.Second,
			},
			expectedError: "",
		},
		{
			name: "zero waiting timeout",
			cfg: NotificationsConfig{
				EndpointServiceConfig: EndpointServiceConfig{
					Address:           "localhost:1234",
					ConnectionTimeout: 5 * time.Second,
					TLS:               validTLS,
				},
				WaitingTimeout: 0,
			},
			expectedError: "waiting timeout must be greater than zero",
		},
		{
			name: "negative waiting timeout",
			cfg: NotificationsConfig{
				EndpointServiceConfig: EndpointServiceConfig{
					Address:           "localhost:1234",
					ConnectionTimeout: 5 * time.Second,
					TLS:               validTLS,
				},
				WaitingTimeout: -1 * time.Second,
			},
			expectedError: "waiting timeout must be greater than zero",
		},
		{
			name: "invalid embedded config",
			cfg: NotificationsConfig{
				EndpointServiceConfig: EndpointServiceConfig{
					Address:           "invalid-address",
					ConnectionTimeout: 5 * time.Second,
					TLS:               validTLS,
				},
				WaitingTimeout: 30 * time.Second,
			},
			expectedError: "invalid address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate(vctx)
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestQueriesConfig_Validate(t *testing.T) {
	t.Parallel()

	validTLS := &TLSConfig{Enabled: new(bool)}
	*validTLS.Enabled = false

	vctx := validation.Context{}

	tests := []struct {
		name          string
		cfg           QueriesConfig
		expectedError string
	}{
		{
			name: "valid config",
			cfg: QueriesConfig{
				EndpointServiceConfig: EndpointServiceConfig{
					Address:           "localhost:1234",
					ConnectionTimeout: 5 * time.Second,
					TLS:               validTLS,
				},
			},
			expectedError: "",
		},
		{
			name: "invalid embedded config",
			cfg: QueriesConfig{
				EndpointServiceConfig: EndpointServiceConfig{
					Address:           "invalid-address",
					ConnectionTimeout: 5 * time.Second,
					TLS:               validTLS,
				},
			},
			expectedError: "invalid address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate(vctx)
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
