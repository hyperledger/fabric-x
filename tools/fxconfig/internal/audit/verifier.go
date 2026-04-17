/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Verifier provides tamper detection for audit logs.
type Verifier struct {
	signingKey []byte
}

// NewVerifier creates a new audit log verifier.
func NewVerifier(signingKey string) *Verifier {
	var key []byte
	if signingKey != "" {
		k, err := hex.DecodeString(signingKey)
		if err == nil {
			key = k
		}
	}
	return &Verifier{signingKey: key}
}

// Verify checks the integrity of audit log files in the specified directory.
func (v *Verifier) Verify(path string) ([]LogIntegrity, error) {
	var results []LogIntegrity

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var files []string
	if info.IsDir() {
		files, err = filepath.Glob(filepath.Join(path, "audit-*.log*"))
		if err != nil {
			return nil, fmt.Errorf("failed to glob log files: %w", err)
		}
	} else {
		files = []string{path}
	}

	for _, file := range files {
		result, err := v.verifyFile(file)
		if err != nil {
			results = append(results, LogIntegrity{
				FilePath:   file,
				VerifiedAt: time.Now().UTC(),
				Valid:      false,
				Error:      err.Error(),
			})
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

func (v *Verifier) verifyFile(path string) (LogIntegrity, error) {
	checksumPath := path + ".sha256"
	storedChecksum, err := os.ReadFile(checksumPath)
	if err != nil {
		return LogIntegrity{}, fmt.Errorf("checksum file not found for %s", path)
	}

	actualChecksum, err := computeFileChecksum(path)
	if err != nil {
		return LogIntegrity{}, fmt.Errorf("failed to compute checksum: %w", err)
	}

	valid := strings.TrimSpace(string(storedChecksum)) == actualChecksum

	return LogIntegrity{
		FilePath:   path,
		Checksum:   actualChecksum,
		VerifiedAt: time.Now().UTC(),
		Valid:      valid,
	}, nil
}

// VerifyEntry checks if an entry's signature is valid.
func (v *Verifier) VerifyEntry(entry *AuditEntry) bool {
	if v.signingKey == nil || entry.Signature == "" {
		return true
	}

	sig := computeEntrySignature(entry, v.signingKey)
	return hmac.Equal([]byte(entry.Signature), []byte(sig))
}

func computeFileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func computeEntrySignature(entry *AuditEntry, key []byte) string {
	entryCopy := *entry
	entryCopy.Signature = ""

	data, _ := json.Marshal(entryCopy)
	data = canonicalizeBytes(data)

	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func canonicalizeBytes(data []byte) []byte {
	var m map[string]any
	json.Unmarshal(data, &m)
	delete(m, "signature")

	out, _ := json.Marshal(m)
	return out
}