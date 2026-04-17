/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"time"
)

// NamespaceDeployInputValidationEvent records input validation for namespace deployment.
type NamespaceDeployInputValidationEvent struct {
	EventMeta
	NamespaceID string
	Version     int
	PolicyType  string
	Endorse     bool
	Submit      bool
	Wait        bool
	Result      string
	ErrorMsg    string
}

// NamespaceCreationStartedEvent records when namespace creation begins.
type NamespaceCreationStartedEvent struct {
	EventMeta
	NamespaceID string
	Version     int
}

// NamespaceCreatedEvent records successful namespace transaction creation.
type NamespaceCreatedEvent struct {
	EventMeta
	TxID       string
	Namespace  string
	Version    int
	Policy     PolicyInfo
	Result     string
	ErrorMsg   string
}

// PolicyInfo describes the policy used for a namespace.
type PolicyInfo struct {
	Type                 string
	MSPExpression        string
	VerificationKeyPath string
}

// TransactionEndorsementStartedEvent records when transaction endorsement begins.
type TransactionEndorsementStartedEvent struct {
	EventMeta
	TxID string
}

// TransactionEndorsedEvent records successful transaction endorsement.
type TransactionEndorsedEvent struct {
	EventMeta
	TxID            string
	SignerID        string
	SignerType      string
	NamespaceCount  int
	SignatureCount  int
	Result          string
	ErrorMsg        string
}

// TransactionSubmissionStartedEvent records when transaction submission begins.
type TransactionSubmissionStartedEvent struct {
	EventMeta
	TxID     string
	Channel  string
	Orderer  string
}

// TransactionSubmittedEvent records transaction broadcast result.
type TransactionSubmittedEvent struct {
	EventMeta
	TxID        string
	Channel     string
	Orderer     string
	EnvelopeHash string
	Result      string
	ErrorMsg    string
}

// TransactionCommitWaitStartedEvent records when waiting for transaction commit begins.
type TransactionCommitWaitStartedEvent struct {
	EventMeta
	TxID       string
	Channel    string
	Timeout    time.Duration
}

// TransactionCommittedEvent records transaction commit status.
type TransactionCommittedEvent struct {
	EventMeta
	TxID       string
	Channel    string
	Status     string
	BlockNum   uint64
	Result     string
	ErrorMsg   string
}

// TransactionMergeStartedEvent records when transaction merge begins.
type TransactionMergeStartedEvent struct {
	EventMeta
	TxCount   int
	TxIDs     []string
}

// TransactionMergedEvent records transaction merge result.
type TransactionMergedEvent struct {
	EventMeta
	InputTxIDs       []string
	MergedTxID       string
	NamespaceCount   int
	TotalEndorsements int
	UniqueEndorsers  []string
	Result           string
	ErrorMsg         string
}

// IdentityLoadStartedEvent records when MSP identity load begins.
type IdentityLoadStartedEvent struct {
	EventMeta
	MspID      string
	ConfigPath string
}

// IdentityLoadedEvent records MSP identity load result.
type IdentityLoadedEvent struct {
	EventMeta
	MspID      string
	ConfigPath string
	CertSubject string
	Result      string
	ErrorMsg   string
}

// GRPCConnectionEstablishedEvent records gRPC connection setup.
type GRPCConnectionEstablishedEvent struct {
	EventMeta
	Address     string
	TLSEnabled  bool
	MTLSEnabled bool
	ServiceType string
	Result      string
	ErrorMsg    string
}

// GRPCConnectionFailedEvent records gRPC connection failure.
type GRPCConnectionFailedEvent struct {
	EventMeta
	Address    string
	ServiceType string
	Result     string
	ErrorMsg   string
}

// MTLSConfiguredEvent records mTLS configuration result.
type MTLSConfiguredEvent struct {
	EventMeta
	ClientCertPath string
	KeyPath        string
	Result         string
	ErrorMsg       string
}

// PolicyValidationStartedEvent records when policy validation begins.
type PolicyValidationStartedEvent struct {
	EventMeta
	PolicyType string
	Expression string
}

// PolicyValidatedEvent records policy validation result.
type PolicyValidatedEvent struct {
	EventMeta
	PolicyType string
	Expression string
	Result     string
	ErrorMsg   string
}

// ConfigLoadedEvent records configuration loading result.
type ConfigLoadedEvent struct {
	EventMeta
	Sources    []string
	OrdererAddr string
	QueriesAddr string
	Result     string
	ErrorMsg   string
}

// NewEventMeta creates a new EventMeta with the current timestamp and service name.
func NewEventMeta() EventMeta {
	return EventMeta{
		Timestamp: time.Now().UTC(),
		Actor: Actor{
			Service: "fxconfig",
		},
	}
}

// WithPrincipal sets the principal field on the actor.
func (e *EventMeta) WithPrincipal(principal string) *EventMeta {
	e.Actor.Principal = principal
	return e
}

// WithMspID sets the MSP ID field on the actor.
func (e *EventMeta) WithMspID(mspID string) *EventMeta {
	e.Actor.MspID = mspID
	return e
}

// WithTraceID sets the trace ID field.
func (e *EventMeta) WithTraceID(traceID string) *EventMeta {
	e.TraceID = traceID
	return e
}