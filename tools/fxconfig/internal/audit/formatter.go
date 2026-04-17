/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"bytes"
	"encoding/json"
	"time"
)

// Formatter defines the interface for serializing audit entries.
type Formatter interface {
	Format(entry *AuditEntry) ([]byte, error)
}

// JSONFormatter outputs audit entries as JSON lines.
type JSONFormatter struct{}

// Format serializes an audit entry to JSON with consistent field ordering.
func (f *JSONFormatter) Format(entry *AuditEntry) ([]byte, error) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.Version == "" {
		entry.Version = "1.0"
	}
	return json.Marshal(entry)
}

// FormatPretty serializes an audit entry to formatted JSON for debugging.
func (f *JSONFormatter) FormatPretty(entry *AuditEntry) ([]byte, error) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.Version == "" {
		entry.Version = "1.0"
	}
	return json.MarshalIndent(entry, "", "  ")
}

// ParseEntry parses a JSON-encoded audit entry.
func ParseEntry(data []byte) (*AuditEntry, error) {
	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// ParseEntries parses multiple JSON-encoded audit entries from a byte slice.
func ParseEntries(data []byte) ([]*AuditEntry, error) {
	var entries []*AuditEntry
	decoder := json.NewDecoder(bytes.NewReader(data))
	for decoder.More() {
		var entry AuditEntry
		if err := decoder.Decode(&entry); err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}
	return entries, nil
}