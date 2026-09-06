/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package provider implements lazy initialization and validation for service instances.
package provider

import (
	"sync"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// Provider manages lazy initialization of service instances with validation support.
// It uses a mutex-guarded initialization that retries on transient errors while caching
// permanent failures (like validation errors).
type Provider[T any, K Validatable] struct {
	mu                sync.Mutex
	factory           func(cfg K) (T, error)
	instance          T
	err               error
	initialized       bool
	cfg               K
	validationContext validation.Context
}

// New creates a Provider with the given factory function, configuration, and validation context.
// The factory is called lazily on the first Get() invocation.
func New[T any, K Validatable](
	factory func(cfg K) (T, error),
	cfg K,
	validationContext validation.Context,
) *Provider[T, K] {
	return &Provider[T, K]{
		factory:           factory,
		cfg:               cfg,
		validationContext: validationContext,
	}
}

// Get returns the service instance, validating the config and initializing the service instance on first call.
// Subsequent calls return the cached instance if initialization succeeded or failed with a permanent error.
// If initialization fails with a transient error, subsequent calls will retry. Thread-safe.
func (p *Provider[T, K]) Get() (T, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return p.instance, p.err
	}

	if err := p.cfg.Validate(p.validationContext); err != nil {
		p.err = err
		p.initialized = true
		return p.instance, p.err
	}

	p.instance, p.err = p.factory(p.cfg)
	if p.err == nil {
		p.initialized = true
	}
	return p.instance, p.err
}

// Validate delegates to the configuration's Validate method.
func (p *Provider[T, K]) Validate() error {
	return p.cfg.Validate(p.validationContext)
}

// Validatable defines the interface for configuration types that can be validated.
type Validatable interface {
	Validate(validation.Context) error
}
