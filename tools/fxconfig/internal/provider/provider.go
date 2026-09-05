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
// It ensures thread-safe, single initialization using sync.Once.
type Provider[T any, K Validatable] struct {
	once              sync.Once
	factory           func(cfg K) (T, error)
	instance          T
	err               error
	cfg               K
	initialized       bool
	ready             bool
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
// Subsequent calls return the cached instance. Thread-safe.
func (p *Provider[T, K]) Get() (T, error) {
	p.once.Do(func() {
		p.initialized = true
		if err := p.cfg.Validate(p.validationContext); err != nil {
			p.err = err
			return
		}

		p.instance, p.err = p.factory(p.cfg)
		p.ready = p.err == nil
	})

	return p.instance, p.err
}

// Validate delegates to the configuration's Validate method.
func (p *Provider[T, K]) Validate() error {
	return p.cfg.Validate(p.validationContext)
}

// Close releases the cached instance if it was initialized successfully and
// implements a Close() error method. Non-closeable instances are ignored.
func (p *Provider[T, K]) Close() error {
	if !p.initialized || !p.ready {
		return nil
	}

	closer, ok := any(p.instance).(interface{ Close() error })
	if !ok {
		return nil
	}

	return closer.Close()
}

// Validatable defines the interface for configuration types that can be validated.
type Validatable interface {
	Validate(validation.Context) error
}
