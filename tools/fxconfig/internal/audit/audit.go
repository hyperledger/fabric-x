/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package audit provides structured audit logging for security-critical operations.
package audit

import (
	"context"
	"time"
)

// AuditLogger defines the interface for recording security-critical audit events.
// All methods accept a context and event struct, returning an error if logging fails.
type AuditLogger interface {
	// Namespace lifecycle
	NamespaceDeployInputValidation(ctx context.Context, e NamespaceDeployInputValidationEvent) error
	NamespaceCreationStarted(ctx context.Context, e NamespaceCreationStartedEvent) error
	NamespaceCreated(ctx context.Context, e NamespaceCreatedEvent) error

	// Transaction lifecycle
	TransactionEndorsementStarted(ctx context.Context, e TransactionEndorsementStartedEvent) error
	TransactionEndorsed(ctx context.Context, e TransactionEndorsedEvent) error
	TransactionSubmissionStarted(ctx context.Context, e TransactionSubmissionStartedEvent) error
	TransactionSubmitted(ctx context.Context, e TransactionSubmittedEvent) error
	TransactionCommitWaitStarted(ctx context.Context, e TransactionCommitWaitStartedEvent) error
	TransactionCommitted(ctx context.Context, e TransactionCommittedEvent) error
	TransactionMergeStarted(ctx context.Context, e TransactionMergeStartedEvent) error
	TransactionMerged(ctx context.Context, e TransactionMergedEvent) error

	// Identity and connection events
	IdentityLoadStarted(ctx context.Context, e IdentityLoadStartedEvent) error
	IdentityLoaded(ctx context.Context, e IdentityLoadedEvent) error
	GRPCConnectionEstablished(ctx context.Context, e GRPCConnectionEstablishedEvent) error
	GRPCConnectionFailed(ctx context.Context, e GRPCConnectionFailedEvent) error
	MTLSConfigured(ctx context.Context, e MTLSConfiguredEvent) error

	// Validation events
	PolicyValidationStarted(ctx context.Context, e PolicyValidationStartedEvent) error
	PolicyValidated(ctx context.Context, e PolicyValidatedEvent) error
	ConfigLoaded(ctx context.Context, e ConfigLoadedEvent) error

	// Verification
	VerifyLogs(path string) ([]LogIntegrity, error)
}

// EventMeta contains common metadata for all audit events.
type EventMeta struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     Actor     `json:"actor"`
	TraceID   string    `json:"traceId,omitempty"`
}

// Actor identifies the principal that triggered the event.
type Actor struct {
	Principal string `json:"principal,omitempty"`
	MspID     string `json:"mspId,omitempty"`
	Service   string `json:"service,omitempty"`
}

// AuditEntry represents a complete audit log entry with all fields.
type AuditEntry struct {
	Timestamp  string       `json:"timestamp"`
	Actor      Actor        `json:"actor"`
	Action     string       `json:"action"`
	Resource   Resource     `json:"resource,omitempty"`
	Result     string       `json:"result"`
	Details    Details      `json:"details,omitempty"`
	Error      *string      `json:"error,omitempty"`
	TraceID    string       `json:"traceId,omitempty"`
	Version    string       `json:"version"`
	Signature  string       `json:"signature,omitempty"`
}

// Resource identifies the object being acted upon.
type Resource struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

// Details holds action-specific structured data.
type Details map[string]any

// LogIntegrity represents the verification result for a single log file.
type LogIntegrity struct {
	FilePath   string    `json:"filePath"`
	Checksum   string    `json:"checksum"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Valid      bool      `json:"valid"`
	Error      string    `json:"error,omitempty"`
}

// nopAuditLogger is a no-op implementation for when audit is disabled.
type nopAuditLogger struct{}

func (n *nopAuditLogger) NamespaceDeployInputValidation(ctx context.Context, e NamespaceDeployInputValidationEvent) error {
	return nil
}
func (n *nopAuditLogger) NamespaceCreationStarted(ctx context.Context, e NamespaceCreationStartedEvent) error {
	return nil
}
func (n *nopAuditLogger) NamespaceCreated(ctx context.Context, e NamespaceCreatedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionEndorsementStarted(ctx context.Context, e TransactionEndorsementStartedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionEndorsed(ctx context.Context, e TransactionEndorsedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionSubmissionStarted(ctx context.Context, e TransactionSubmissionStartedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionSubmitted(ctx context.Context, e TransactionSubmittedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionCommitWaitStarted(ctx context.Context, e TransactionCommitWaitStartedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionCommitted(ctx context.Context, e TransactionCommittedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionMergeStarted(ctx context.Context, e TransactionMergeStartedEvent) error {
	return nil
}
func (n *nopAuditLogger) TransactionMerged(ctx context.Context, e TransactionMergedEvent) error {
	return nil
}
func (n *nopAuditLogger) IdentityLoadStarted(ctx context.Context, e IdentityLoadStartedEvent) error {
	return nil
}
func (n *nopAuditLogger) IdentityLoaded(ctx context.Context, e IdentityLoadedEvent) error {
	return nil
}
func (n *nopAuditLogger) GRPCConnectionEstablished(ctx context.Context, e GRPCConnectionEstablishedEvent) error {
	return nil
}
func (n *nopAuditLogger) GRPCConnectionFailed(ctx context.Context, e GRPCConnectionFailedEvent) error {
	return nil
}
func (n *nopAuditLogger) MTLSConfigured(ctx context.Context, e MTLSConfiguredEvent) error {
	return nil
}
func (n *nopAuditLogger) PolicyValidationStarted(ctx context.Context, e PolicyValidationStartedEvent) error {
	return nil
}
func (n *nopAuditLogger) PolicyValidated(ctx context.Context, e PolicyValidatedEvent) error {
	return nil
}
func (n *nopAuditLogger) ConfigLoaded(ctx context.Context, e ConfigLoadedEvent) error {
	return nil
}
func (n *nopAuditLogger) VerifyLogs(path string) ([]LogIntegrity, error) {
	return nil, nil
}