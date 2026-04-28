/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// resetApp reinitializes the kingpin app to avoid state leaks between tests.
// It sets a terminate handler that panics instead of calling os.Exit, allowing
// the test's deferred recover() to catch the termination.
func resetApp(t *testing.T) {
	t.Helper()
	app.Terminate(func(_ int) {
		panic("kingpin-exit")
	})
}

func TestComputeUpdateMissingOriginalFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetApp(t)

	os.Args = []string{
		"configtxlator",
		"compute_update",
		"--channel_id=testchannel",
	}

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, err := app.Parse(os.Args[1:])
		if err != nil {
			// Kingpin returns an error when required flags are missing.
			// This is the expected path — it means .Required() is working.
			t.Logf("app.Parse correctly returned error: %v", err)
			return
		}
		t.Error("expected app.Parse to return an error when --original and --updated are missing")
	}()

	if panicked {
		// Kingpin called Terminate, which we turned into a panic. That also means
		// it detected the missing required flag. Both paths are acceptable.
		t.Log("kingpin terminated as expected for missing required flags")
	}
}

func TestComputeUpdateWithValidFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetApp(t)

	origFile := filepath.Join(t.TempDir(), "original.pb")
	updatedFile := filepath.Join(t.TempDir(), "updated.pb")
	outFile := filepath.Join(t.TempDir(), "output.pb")

	// Write empty files so kingpin can open them.
	os.WriteFile(origFile, []byte{}, 0644)
	os.WriteFile(updatedFile, []byte{}, 0644)

	os.Args = []string{
		"configtxlator",
		"compute_update",
		"--channel_id=testchannel",
		"--original=" + origFile,
		"--updated=" + updatedFile,
		"--output=" + outFile,
	}

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		t.Fatalf("app.Parse failed unexpectedly: %v", err)
	}

	if cmd != computeUpdate.FullCommand() {
		t.Fatalf("expected command %q, got %q", computeUpdate.FullCommand(), cmd)
	}

	// Verify file handles are non-nil — this is the core assertion.
	// Before this fix, they would be nil and the deferred Close() would panic.
	if *computeUpdateOriginal == nil {
		t.Fatal("computeUpdateOriginal file handle is nil")
	}
	if *computeUpdateUpdated == nil {
		t.Fatal("computeUpdateUpdated file handle is nil")
	}
	if *computeUpdateDest == nil {
		t.Fatal("computeUpdateDest file handle is nil")
	}

	// Clean up file handles opened by kingpin.
	(*computeUpdateOriginal).Close()
	(*computeUpdateUpdated).Close()
	(*computeUpdateDest).Close()
}

func TestComputeUpdateNilGuardsPreventPanic(t *testing.T) {
	// Directly call computeUpdt with nil to verify it returns an error
	// rather than panicking with a nil pointer dereference.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("computeUpdt panicked with nil inputs: %v", r)
		}
	}()

	err := computeUpdt(nil, nil, nil, "testchannel")
	if err == nil {
		t.Fatal("expected error from computeUpdt with nil inputs, got nil")
	}
	t.Logf("computeUpdt correctly returned error: %v", err)
}
