/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Sink defines the interface for audit output destinations.
type Sink interface {
	Write(entry *AuditEntry) error
	Close() error
}

// FileSink writes JSON lines to a file with rotation support.
type FileSink struct {
mu      sync.Mutex
file    *os.File
path    string
	rotator *Rotator
}

// NewFileSink creates a new file-based sink.
func NewFileSink(path string, rotator *Rotator) *FileSink {
	return &FileSink{
		path:    path,
		rotator: rotator,
	}
}

// Write writes an audit entry to the file.
func (s *FileSink) Write(entry *AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	if s.file == nil {
		f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open audit file: %w", err)
		}
		s.file = f
	}

	if s.rotator != nil && s.rotator.ShouldRotate() {
		if err := s.rotate(); err != nil {
			return err
		}
	}

	_, err = s.file.Write(append(data, '\n'))
	return err
}

func (s *FileSink) rotate() error {
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}

	checksum := s.computeChecksum()
	if err := s.rotator.Rotate(s.path, checksum); err != nil {
		return fmt.Errorf("failed to rotate: %w", err)
	}

	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new audit file: %w", err)
	}
	s.file = f
	return nil
}

func (s *FileSink) computeChecksum() string {
	if s.file == nil {
		return ""
	}
	stat, err := s.file.Stat()
	if err != nil || stat.Size() == 0 {
		return ""
	}
	content := make([]byte, stat.Size())
	s.file.ReadAt(content, 0)
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// Close closes the file sink.
func (s *FileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// SyslogSink sends entries to a syslog server (RFC 5424).
type SyslogSink struct {
	mu       sync.Mutex
	conn     net.Conn
	network  string
	addr     string
	facility int
	program  string
}

// NewSyslogSink creates a new syslog sink.
func NewSyslogSink(network, addr string, facility int) *SyslogSink {
	return &SyslogSink{
		network:  network,
		addr:     addr,
		facility: facility,
		program:  "fxconfig",
	}
}

// Write sends an audit entry to syslog.
func (s *SyslogSink) Write(entry *AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		conn, err := net.Dial(s.network, s.addr)
		if err != nil {
			return fmt.Errorf("failed to connect to syslog: %w", err)
		}
		s.conn = conn
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	pri := s.facility*8 + 6 // local0 + info
	timestamp := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf("<%d>%s %s %s: %s", pri, timestamp, s.program, "AUDIT", string(data))

	_, err = s.conn.Write([]byte(msg))
	return err
}

// Close closes the syslog connection.
func (s *SyslogSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	return nil
}

// WebhookSink POSTs entries to an HTTP endpoint for SIEM integration.
type WebhookSink struct {
	mu          sync.Mutex
	url         string
	headers     map[string]string
	client      *http.Client
	retryCount  int
	retryDelay  time.Duration
}

// NewWebhookSink creates a new webhook sink.
func NewWebhookSink(url string) *WebhookSink {
	return &WebhookSink{
		url:        url,
		client:     &http.Client{Timeout: 10 * time.Second},
		retryCount: 3,
		retryDelay: 500 * time.Millisecond,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// Write sends an audit entry to the webhook URL.
func (s *WebhookSink) Write(entry *AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	var lastErr error
	for i := 0; i <= s.retryCount; i++ {
		if i > 0 {
			time.Sleep(s.retryDelay)
		}

		req, err := http.NewRequest(http.MethodPost, s.url, strings.NewReader(string(data)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		for k, v := range s.headers {
			req.Header.Set(k, v)
		}

		resp, err := s.client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
			continue
		}
		lastErr = err
	}

	return lastErr
}

// Close closes the webhook sink (no-op for HTTP client).
func (s *WebhookSink) Close() error {
	return nil
}

// MultiSink writes to multiple sinks in parallel.
type MultiSink struct {
	sinks []Sink
}

// NewMultiSink creates a sink that writes to multiple destinations.
func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

// Write writes an entry to all configured sinks.
func (m *MultiSink) Write(entry *AuditEntry) error {
	var lastErr error
	for _, sink := range m.sinks {
		if err := sink.Write(entry); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Close closes all sinks.
func (m *MultiSink) Close() error {
	var lastErr error
	for _, sink := range m.sinks {
		if err := sink.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// BufferedSink wraps a sink with async buffered writes.
type BufferedSink struct {
	mu    sync.Mutex
	sink  Sink
	buf   chan *AuditEntry
	done  chan struct{}
	size  int
}

// NewBufferedSink creates a buffered sink with the specified buffer size.
func NewBufferedSink(sink Sink, size int) *BufferedSink {
	bs := &BufferedSink{
		sink: sink,
		buf:  make(chan *AuditEntry, size),
		done: make(chan struct{}),
		size: size,
	}
	go bs.flushLoop()
	return bs
}

func (bs *BufferedSink) flushLoop() {
	for {
		select {
		case entry := <-bs.buf:
			if entry != nil {
				bs.sink.Write(entry)
			}
		case <-bs.done:
			close(bs.buf)
			for entry := range bs.buf {
				if entry != nil {
					bs.sink.Write(entry)
				}
			}
			return
		}
	}
}

// Write enqueues an entry for async writing.
func (bs *BufferedSink) Write(entry *AuditEntry) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	select {
	case bs.buf <- entry:
		return nil
	default:
		return fmt.Errorf("buffer full")
	}
}

// Close closes the buffered sink.
func (bs *BufferedSink) Close() error {
	bs.mu.Lock()
	close(bs.done)
	bs.mu.Unlock()
	return bs.sink.Close()
}

// ioCloser interface for closing.
type ioCloser interface {
	Close() error
}

// ensureSinkCloser checks if a sink implements io.Closer.
func ensureSinkCloser(sink Sink) bool {
	_, ok := sink.(ioCloser)
	return ok
}

// DrainSink drains remaining entries before closing.
type DrainSink struct {
	sink Sink
}

// NewDrainSink wraps a sink with drain-on-close behavior.
func NewDrainSink(sink Sink) *DrainSink {
	return &DrainSink{sink: sink}
}

// Write writes to the underlying sink.
func (d *DrainSink) Write(entry *AuditEntry) error {
	return d.sink.Write(entry)
}

// Close drains and then closes the sink.
func (d *DrainSink) Close() error {
	if ds, ok := d.sink.(*BufferedSink); ok {
		ds.Close()
	}
	return d.sink.Close()
}