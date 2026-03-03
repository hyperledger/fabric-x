/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

// PolicyConfig defines the endorsement policy for a namespace.
type PolicyConfig struct {
	Type string `mapstructure:"type"` // "msp" | "threshold"`

	MSP       *MSPPolicyConfig       `mapstructure:"msp"`
	Threshold *ThresholdPolicyConfig `mapstructure:"threshold"`
}

// Set parses and configures the policy from a string.
// Supports "threshold:<path>" format or MSP DSL expressions.
func (c *PolicyConfig) Set(policy string) {
	policy = strings.TrimSpace(policy)

	if k, ok := strings.CutPrefix(policy, "threshold:"); ok {
		c.Type = thresholdPolicyType
		c.Threshold = &ThresholdPolicyConfig{
			VerificationKeyPath: strings.TrimSpace(k),
		}
		return
	}

	// default is msp
	c.Type = mspPolicyType
	c.MSP = &MSPPolicyConfig{Expression: policy}
}

// MSPPolicyConfig holds MSP-based policy configuration.
type MSPPolicyConfig struct {
	Expression string `mapstructure:"expression"`
}

// ThresholdPolicyConfig holds threshold ECDSA policy configuration.
type ThresholdPolicyConfig struct {
	VerificationKeyPath string `mapstructure:"verificationKeyPath"`
}

// Validate validates policy configuration based on policy type.
func (c *PolicyConfig) Validate(ctx validation.Context) error {
	if c == nil {
		return errors.New("policy is required")
	}

	switch c.Type {
	case mspPolicyType:
		if c.MSP == nil {
			return errors.New("msp policy config missing")
		}
		return c.MSP.Validate(ctx)

	case thresholdPolicyType:
		if c.Threshold == nil {
			return errors.New("threshold policy config missing")
		}
		return c.Threshold.Validate(ctx)

	default:
		return fmt.Errorf("unknown policy type: %s", c.Type)
	}
}

// Validate validates threshold policy configuration.
// Ensures the verification key path exists and is accessible.
func (c *ThresholdPolicyConfig) Validate(vctx validation.Context) error {
	if c.VerificationKeyPath == "" {
		return errors.New("threshold policy key path must not be empty")
	}

	if vctx.FileChecker == nil {
		return errors.New("file checker not available")
	}

	return vctx.FileChecker.Exists(c.VerificationKeyPath)
}

// Validate validates MSP policy configuration.
// Checks that the policy expression is valid DSL syntax.
func (c *MSPPolicyConfig) Validate(ctx validation.Context) error {
	if c.Expression == "" {
		return errors.New("msp policy expression must not be empty")
	}

	if ctx.PolicyChecker == nil {
		return errors.New("policy checker not available")
	}

	return ctx.PolicyChecker.Check(c.Expression)
}
