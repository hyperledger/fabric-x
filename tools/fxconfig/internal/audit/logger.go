/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// AuditConfig configures the audit logger.

var (
	defaultAuditLogger AuditLogger = &nopAuditLogger{}
	auditLoggerOnce    sync.Once
	auditLoggerMu      sync.RWMutex
)

// MustGetAuditLogger returns the global audit logger instance.
func MustGetAuditLogger(cfg *config.AuditConfig) AuditLogger {
	auditLoggerMu.RLock()
	if defaultAuditLogger != nil && !isNop(defaultAuditLogger) {
		auditLoggerMu.RUnlock()
		return defaultAuditLogger
	}
	auditLoggerMu.RUnlock()

	auditLoggerOnce.Do(func() {
		auditLoggerMu.Lock()
		defer auditLoggerMu.Unlock()
		defaultAuditLogger = newAuditLogger(cfg)
	})
	return defaultAuditLogger
}

// SetAuditLogger sets the global audit logger (for testing).
func SetAuditLogger(l AuditLogger) {
	auditLoggerMu.Lock()
	defer auditLoggerMu.Unlock()
	defaultAuditLogger = l
}

func isNop(l AuditLogger) bool {
	_, ok := l.(*nopAuditLogger)
	return ok
}

func newAuditLogger(cfg *config.AuditConfig) AuditLogger {
	if cfg == nil || !cfg.Enabled {
		return &nopAuditLogger{}
	}

	rotatorCfg := &RotatorConfig{
		MaxSize:    cfg.MaxSizeMB,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxBackups,
		Compress:   true,
		TimeUnit:   24 * time.Hour,
	}

	var sink Sink
	if cfg.WebhookURL != "" {
		sink = NewWebhookSink(cfg.WebhookURL)
	} else if cfg.SyslogEnabled && cfg.SyslogAddr != "" {
		sink = NewSyslogSink("udp", cfg.SyslogAddr, 16) // local0
	} else {
		rotator := NewRotator(rotatorCfg, cfg.OutputPath)
		sink = NewFileSink(cfg.OutputPath, rotator)
	}

	var verifier *Verifier
	if cfg.VerifyEnabled && cfg.SigningKey != "" {
		verifier = NewVerifier(cfg.SigningKey)
	}

	return &auditLogger{
		sink:      sink,
		verifier:  verifier,
		signingKey: parseKey(cfg.SigningKey),
		formatter: &JSONFormatter{},
	}
}

func parseKey(key string) []byte {
	if key == "" {
		return nil
	}
	k, err := hex.DecodeString(key)
	if err != nil {
		return nil
	}
	return k
}

type auditLogger struct {
	sink       Sink
	verifier   *Verifier
	signingKey []byte
	formatter  Formatter
	mu         sync.Mutex
}

func (a *auditLogger) logEvent(action string, result string, entry *AuditEntry) error {
	entry.Action = action
	entry.Result = result

	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.Version == "" {
		entry.Version = "1.0"
	}

	// Sign if signing key is configured
	if a.signingKey != nil {
		sig := a.sign(entry)
		entry.Signature = sig
	}

	return a.sink.Write(entry)
}

func (a *auditLogger) sign(entry *AuditEntry) string {
	// Create canonical JSON representation
	data, _ := json.Marshal(entry)
	data = canonicalize(data)

	h := hmac.New(sha256.New, a.signingKey)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func canonicalize(data []byte) []byte {
	var m map[string]any
	json.Unmarshal(data, &m)

	// Remove signature field for canonical form
	delete(m, "signature")

	out, _ := json.Marshal(m)
	return out
}

// Namespace lifecycle

func (a *auditLogger) NamespaceDeployInputValidation(ctx context.Context, e NamespaceDeployInputValidationEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "namespace",
			ID:   e.NamespaceID,
		},
		Details: map[string]any{
			"version":    e.Version,
			"policyType": e.PolicyType,
			"endorse":    e.Endorse,
			"submit":     e.Submit,
			"wait":       e.Wait,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("namespace.deploy.input_validated", e.Result, entry)
}

func (a *auditLogger) NamespaceCreationStarted(ctx context.Context, e NamespaceCreationStartedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "namespace",
			ID:   e.NamespaceID,
		},
		Details: map[string]any{
			"version": e.Version,
		},
	}
	return a.logEvent("namespace.creation.started", "pending", entry)
}

func (a *auditLogger) NamespaceCreated(ctx context.Context, e NamespaceCreatedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "namespace",
			ID:   e.Namespace,
		},
		Details: map[string]any{
			"txId":    e.TxID,
			"version": e.Version,
			"policy":   e.Policy,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("namespace.created", e.Result, entry)
}

// Transaction lifecycle

func (a *auditLogger) TransactionEndorsementStarted(ctx context.Context, e TransactionEndorsementStartedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "transaction",
			ID:   e.TxID,
		},
	}
	return a.logEvent("transaction.endorsement.started", "pending", entry)
}

func (a *auditLogger) TransactionEndorsed(ctx context.Context, e TransactionEndorsedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "transaction",
			ID:   e.TxID,
		},
		Details: map[string]any{
			"signerId":        e.SignerID,
			"signerType":      e.SignerType,
			"namespaceCount":  e.NamespaceCount,
			"signatureCount":  e.SignatureCount,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("transaction.endorsed", e.Result, entry)
}

func (a *auditLogger) TransactionSubmissionStarted(ctx context.Context, e TransactionSubmissionStartedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "transaction",
			ID:   e.TxID,
		},
		Details: map[string]any{
			"channel": e.Channel,
			"orderer": e.Orderer,
		},
	}
	return a.logEvent("transaction.submission.started", "pending", entry)
}

func (a *auditLogger) TransactionSubmitted(ctx context.Context, e TransactionSubmittedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "transaction",
			ID:   e.TxID,
		},
		Details: map[string]any{
			"channel":       e.Channel,
			"orderer":       e.Orderer,
			"envelopeHash": e.EnvelopeHash,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("transaction.submitted", e.Result, entry)
}

func (a *auditLogger) TransactionCommitWaitStarted(ctx context.Context, e TransactionCommitWaitStartedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "transaction",
			ID:   e.TxID,
		},
		Details: map[string]any{
			"channel": e.Channel,
			"timeout": e.Timeout.String(),
		},
	}
	return a.logEvent("transaction.commit.wait.started", "pending", entry)
}

func (a *auditLogger) TransactionCommitted(ctx context.Context, e TransactionCommittedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "transaction",
			ID:   e.TxID,
		},
		Details: map[string]any{
			"channel":   e.Channel,
			"status":    e.Status,
			"blockNum":  e.BlockNum,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("transaction.committed", e.Result, entry)
}

func (a *auditLogger) TransactionMergeStarted(ctx context.Context, e TransactionMergeStartedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"txCount": len(e.TxIDs),
			"txIds":   e.TxIDs,
		},
	}
	return a.logEvent("transaction.merge.started", "pending", entry)
}

func (a *auditLogger) TransactionMerged(ctx context.Context, e TransactionMergedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Resource: Resource{
			Type: "transaction",
			ID:   e.MergedTxID,
		},
		Details: map[string]any{
			"inputTxIds":        e.InputTxIDs,
			"namespaceCount":    e.NamespaceCount,
			"totalEndorsements": e.TotalEndorsements,
			"uniqueEndorsers":   e.UniqueEndorsers,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("transaction.merged", e.Result, entry)
}

// Identity and connection events

func (a *auditLogger) IdentityLoadStarted(ctx context.Context, e IdentityLoadStartedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"mspId":      e.MspID,
			"configPath": e.ConfigPath,
		},
	}
	return a.logEvent("identity.load.started", "pending", entry)
}

func (a *auditLogger) IdentityLoaded(ctx context.Context, e IdentityLoadedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"mspId":       e.MspID,
			"configPath":  e.ConfigPath,
			"certSubject": e.CertSubject,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("identity.loaded", e.Result, entry)
}

func (a *auditLogger) GRPCConnectionEstablished(ctx context.Context, e GRPCConnectionEstablishedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"address":     e.Address,
			"tlsEnabled":  e.TLSEnabled,
			"mtlsEnabled": e.MTLSEnabled,
			"serviceType": e.ServiceType,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("grpc.connection.established", e.Result, entry)
}

func (a *auditLogger) GRPCConnectionFailed(ctx context.Context, e GRPCConnectionFailedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"address":     e.Address,
			"serviceType": e.ServiceType,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("grpc.connection.failed", e.Result, entry)
}

func (a *auditLogger) MTLSConfigured(ctx context.Context, e MTLSConfiguredEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"clientCertPath": e.ClientCertPath,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("mtls.configured", e.Result, entry)
}

// Validation events

func (a *auditLogger) PolicyValidationStarted(ctx context.Context, e PolicyValidationStartedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"policyType": e.PolicyType,
			"expression": e.Expression,
		},
	}
	return a.logEvent("policy.validation.started", "pending", entry)
}

func (a *auditLogger) PolicyValidated(ctx context.Context, e PolicyValidatedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"policyType": e.PolicyType,
			"expression": e.Expression,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("policy.validated", e.Result, entry)
}

func (a *auditLogger) ConfigLoaded(ctx context.Context, e ConfigLoadedEvent) error {
	entry := &AuditEntry{
		Actor:   e.EventMeta.Actor,
		TraceID: e.EventMeta.TraceID,
		Details: map[string]any{
			"sources":      e.Sources,
			"ordererAddr":  e.OrdererAddr,
			"queriesAddr":  e.QueriesAddr,
		},
	}
	if e.ErrorMsg != "" {
		entry.Error = &e.ErrorMsg
	}
	return a.logEvent("config.loaded", e.Result, entry)
}

// VerifyLogs implements AuditLogger.
func (a *auditLogger) VerifyLogs(path string) ([]LogIntegrity, error) {
	if a.verifier == nil {
		return nil, fmt.Errorf("verifier not configured")
	}
	return a.verifier.Verify(path)
}