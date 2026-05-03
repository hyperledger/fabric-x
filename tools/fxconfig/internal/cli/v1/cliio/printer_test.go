// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cliio

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const testValue1 = "value1"

func TestNewCLIPrinter(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	printer := NewCLIPrinter(&out, &errOut, FormatJSON)

	require.NotNil(t, printer)
	require.Equal(t, FormatJSON, printer.format)
}

func TestCLIPrinter_Print_JSON(t *testing.T) {
	t.Parallel()

	t.Run("success printing map", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatJSON)

		data := map[string]any{
			"key1": testValue1,
			"key2": 123,
		}

		printer.Print(data)

		var result map[string]any
		err := json.Unmarshal(out.Bytes(), &result)
		require.NoError(t, err)
		require.Equal(t, testValue1, result["key1"])
		require.InEpsilon(t, float64(123), result["key2"], 0.001)
	})

	t.Run("success printing string", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatJSON)

		printer.Print("test string")

		require.Contains(t, out.String(), "test string")
	})

	t.Run("success printing slice", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatJSON)

		data := []string{"item1", "item2", "item3"}
		printer.Print(data)

		var result []string
		err := json.Unmarshal(out.Bytes(), &result)
		require.NoError(t, err)
		require.Equal(t, data, result)
	})
}

func TestCLIPrinter_Print_YAML(t *testing.T) {
	t.Parallel()

	t.Run("success printing map", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatYAML)

		data := map[string]any{
			"key1": testValue1,
			"key2": 123,
		}

		printer.Print(data)

		var result map[string]any
		err := yaml.Unmarshal(out.Bytes(), &result)
		require.NoError(t, err)
		require.Equal(t, testValue1, result["key1"])
		require.Equal(t, 123, result["key2"])
	})

	t.Run("success printing string", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatYAML)

		printer.Print("test string")

		require.Contains(t, out.String(), "test string")
	})
}

func TestCLIPrinter_Print_Table(t *testing.T) {
	t.Parallel()

	t.Run("success printing string", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatTable)

		printer.Print("test output")

		require.Equal(t, "test output", out.String())
	})

	t.Run("success printing struct", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatTable)

		type testStruct struct {
			Field1 string
			Field2 int
		}

		data := testStruct{
			Field1: testValue1,
			Field2: 42,
		}

		printer.Print(data)

		output := out.String()
		require.Contains(t, output, testValue1)
		require.Contains(t, output, "42")
	})
}

func TestCLIPrinter_PrintError_HumanReadable(t *testing.T) {
	t.Parallel()

	t.Run("print error in table format", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatTable)

		testErr := errors.New("test error message")
		printer.PrintError(testErr)

		require.Contains(t, errOut.String(), "Error: test error message")
	})

	t.Run("print error in YAML format", func(t *testing.T) {
		t.Parallel()

		var out, errOut bytes.Buffer
		printer := NewCLIPrinter(&out, &errOut, FormatYAML)

		testErr := errors.New("test error message")
		printer.PrintError(testErr)

		require.Contains(t, errOut.String(), "Error: test error message")
	})
}

func TestCLIPrinter_PrintError_JSON(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	printer := NewCLIPrinter(&out, &errOut, FormatJSON)

	testErr := errors.New("test error message")
	printer.PrintError(testErr)

	var result map[string]any
	err := json.Unmarshal(errOut.Bytes(), &result)
	require.NoError(t, err)
	require.Equal(t, "test error message", result["error"])
}

func TestFormat_Constants(t *testing.T) {
	t.Parallel()

	require.Equal(t, FormatTable, Format("table"))
	require.Equal(t, FormatJSON, Format("json"))
	require.Equal(t, FormatYAML, Format("yaml"))
}