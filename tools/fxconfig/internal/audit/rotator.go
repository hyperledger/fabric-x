/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatorConfig configures log file rotation.
type RotatorConfig struct {
	MaxSize    int           // Max size per file in MB (default 100)
	MaxAge     int           // Max age in days (default 7)
	MaxBackups int           // Max backup files to retain (default 10)
	Compress   bool          // Compress rotated files (default true)
	TimeUnit   time.Duration // Time-based rotation interval (default 24*time.Hour)
}

// DefaultRotatorConfig returns the default rotation configuration.
func DefaultRotatorConfig() *RotatorConfig {
	return &RotatorConfig{
		MaxSize:    100,
		MaxAge:     7,
		MaxBackups: 10,
		Compress:   true,
		TimeUnit:   24 * time.Hour,
	}
}

// Rotator handles log file rotation based on size and time.
type Rotator struct {
	mu       sync.Mutex
	config   *RotatorConfig
	size     int64
	lastTime time.Time
}

// NewRotator creates a new log rotator.
func NewRotator(config *RotatorConfig, path string) *Rotator {
	if config == nil {
		config = DefaultRotatorConfig()
	}
	r := &Rotator{
		config:   config,
		size:     0,
		lastTime: time.Now(),
	}

	// Check existing file size
	if info, err := os.Stat(path); err == nil {
		r.size = info.Size()
	}

	// Check if time-based rotation is needed
	go r.timeBasedRotation(path)

	return r
}

// ShouldRotate returns true if the log should be rotated.
func (r *Rotator) ShouldRotate() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	maxSizeBytes := int64(r.config.MaxSize) * 1024 * 1024
	if r.size >= maxSizeBytes {
		return true
	}

	if time.Since(r.lastTime) >= r.config.TimeUnit {
		return true
	}

	return false
}

// Rotate rotates the log file.
func (r *Rotator) Rotate(path string, checksum string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.config == nil {
		return nil
	}

	// Close current file - caller handles this
	r.size = 0

	// Create backup filename with timestamp
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)

	now := time.Now()
	backupName := fmt.Sprintf("%s-%s%s", base, now.Format("2006-01-02-150405"), ext)
	backupPath := filepath.Join(dir, backupName)

	// Rename current file to backup
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("failed to rename log file: %w", err)
		}

		// Compress if enabled
		if r.config.Compress {
			if err := r.compress(backupPath); err != nil {
				return fmt.Errorf("failed to compress rotated log: %w", err)
			}
		}

		// Write checksum file
		if checksum != "" {
			checksumPath := backupPath + ".sha256"
			if err := os.WriteFile(checksumPath, []byte(checksum), 0644); err != nil {
				return fmt.Errorf("failed to write checksum: %w", err)
			}
		}

		// Clean up old backups
		r.cleanup(dir, base, ext)
	}

	r.lastTime = now
	return nil
}

func (r *Rotator) compress(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzPath := path + ".gz"
	gz, err := os.Create(gzPath)
	if err != nil {
		return err
	}
	defer gz.Close()

	writer := gzip.NewWriter(gz)
	defer writer.Close()

	_, err = io.Copy(writer, f)
	return err
}

func (r *Rotator) cleanup(dir, base, ext string) {
	pattern := fmt.Sprintf("%s-*%s", base, ext)
	if r.config.Compress {
		pattern = fmt.Sprintf("%s-*%s.gz", base, ext)
	}

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// Sort by modification time (oldest first)
	sort.Slice(matches, func(i, j int) bool {
		iInfo, _ := os.Stat(matches[i])
		jInfo, _ := os.Stat(matches[j])
		if iInfo == nil || jInfo == nil {
			return false
		}
		return iInfo.ModTime().Before(jInfo.ModTime())
	})

	// Remove excess backups
	excess := len(matches) - r.config.MaxBackups
	for i := 0; i < excess; i++ {
		os.Remove(matches[i])
	}

	// Remove files older than MaxAge
	cutoff := time.Now().AddDate(0, 0, -r.config.MaxAge)
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil {
			if info.ModTime().Before(cutoff) {
				os.Remove(match)
				// Also remove checksum file
				os.Remove(match + ".sha256")
			}
		}
	}
}

func (r *Rotator) timeBasedRotation(path string) {
	ticker := time.NewTicker(r.config.TimeUnit)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		if time.Since(r.lastTime) >= r.config.TimeUnit {
			r.lastTime = time.Now()
			// Signal rotation - actual rotation handled by caller via ShouldRotate
		}
		r.mu.Unlock()
	}
}

// UpdateSize updates the current size after a write.
func (r *Rotator) UpdateSize(bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.size += bytes
}

// GetSize returns the current size.
func (r *Rotator) GetSize() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size
}