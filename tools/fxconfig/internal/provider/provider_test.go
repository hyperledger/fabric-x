/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// mockConfig implements provider.Validatable for testing.
type mockConfig struct {
	validateErr error
}

func (m *mockConfig) Validate(_ validation.Context) error {
	return m.validateErr
}

// mockService is a simple service type for testing.
type mockService struct {
	value string
}

func TestProvider_Get_Success(t *testing.T) {
	t.Parallel()

	cfg := &mockConfig{}
	factory := func(*mockConfig) (*mockService, error) {
		return &mockService{value: "test"}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	svc, err := p.Get()
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, "test", svc.value)
}

func TestProvider_Get_FactoryError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("factory error")
	cfg := &mockConfig{}
	factory := func(*mockConfig) (*mockService, error) {
		return nil, expectedErr
	}

	p := provider.New(factory, cfg, validation.Context{})

	svc, err := p.Get()
	require.Nil(t, svc)
	require.ErrorIs(t, err, expectedErr)
}

func TestProvider_Get_LazyInitialization(t *testing.T) {
	t.Parallel()

	callCount := 0
	cfg := &mockConfig{}
	factory := func(*mockConfig) (*mockService, error) {
		callCount++
		return &mockService{value: "test"}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	// Call Get multiple times
	_, err1 := p.Get()
	_, err2 := p.Get()
	_, err3 := p.Get()

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// Validation and factory should only be called once due to sync.Once
	require.Equal(t, 1, callCount)
}

func TestProvider_Get_ValidationError(t *testing.T) {
	t.Parallel()

	validationErr := errors.New("validation error")
	cfg := &mockConfig{validateErr: validationErr}
	factoryCalled := false
	factory := func(*mockConfig) (*mockService, error) {
		factoryCalled = true
		return &mockService{value: "test"}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	svc, err := p.Get()
	require.Nil(t, svc)
	require.ErrorIs(t, err, validationErr)
	require.False(t, factoryCalled, "factory must not be called if validation fails")
}

func TestProvider_Get_ValidationErrorIsCached(t *testing.T) {
	t.Parallel()

	validationErr := errors.New("validation error")
	cfg := &mockConfig{validateErr: validationErr}
	factory := func(*mockConfig) (*mockService, error) {
		return &mockService{value: "test"}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	_, err1 := p.Get()
	_, err2 := p.Get()

	require.ErrorIs(t, err1, validationErr)
	require.ErrorIs(t, err2, validationErr)
}

func TestProvider_Get_ThreadSafety(t *testing.T) {
	t.Parallel()

	callCount := 0
	var mu sync.Mutex
	cfg := &mockConfig{}
	factory := func(*mockConfig) (*mockService, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return &mockService{value: "test"}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	// Call Get concurrently from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, err := p.Get()
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	// Factory should only be called once despite concurrent access
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, callCount)
}

func TestProvider_Get_ThreadSafety_InitFailure(t *testing.T) {
	t.Parallel()

	validationErr := errors.New("init failure")
	cfg := &mockConfig{validateErr: validationErr}
	factory := func(*mockConfig) (*mockService, error) {
		return &mockService{value: "test"}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	var wg sync.WaitGroup
	numGoroutines := 20
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			svc, err := p.Get()
			assert.Nil(t, svc)
			assert.ErrorIs(t, err, validationErr)
		}()
	}

	wg.Wait()
}

func TestProvider_Validate_Success(t *testing.T) {
	t.Parallel()

	cfg := &mockConfig{validateErr: nil}
	factory := func(*mockConfig) (*mockService, error) {
		return &mockService{}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	err := p.Validate()
	require.NoError(t, err)
}

func TestProvider_Validate_Error(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("validation error")
	cfg := &mockConfig{validateErr: expectedErr}
	factory := func(*mockConfig) (*mockService, error) {
		return &mockService{}, nil
	}

	p := provider.New(factory, cfg, validation.Context{})

	err := p.Validate()
	require.ErrorIs(t, err, expectedErr)
}
