// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cliio

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Format specifies the output format for CLI printing.
type Format string

// Define available format types.
const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// Printer handles formatted output for CLI commands.
type Printer interface {
	Print(v any)
	PrintError(err error)
}

// CLIPrinter implements Printer with support for multiple output formats.
type CLIPrinter struct {
	out    io.Writer
	errOut io.Writer
	format Format
}

// NewCLIPrinter creates a printer with specified output writers and format.
func NewCLIPrinter(out, errOut io.Writer, format Format) *CLIPrinter {
	return &CLIPrinter{
		out:    out,
		errOut: errOut,
		format: format,
	}
}

// Print outputs data in the configured format (JSON, YAML, or table).
func (p *CLIPrinter) Print(v any) {
	if err := p.print(v); err != nil {
		p.PrintError(err)
	}
}

func (p *CLIPrinter) print(v any) error {
	switch p.format {
	case FormatJSON:
		enc := json.NewEncoder(p.out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case FormatYAML:
		data, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		_, err = p.out.Write(data)
		return err
	default:
		// return renderTable(p.out, v)
		_, err := fmt.Fprint(p.out, v)
		return err
	}
}

// PrintError outputs errors in human-readable or JSON format.
func (p *CLIPrinter) PrintError(err error) {
	// Human readable
	if p.format != FormatJSON {
		//nolint:errcheck
		fmt.Fprintf(p.errOut, "Error: %v\n", err)
		return
	}

	// Machine readable
	_ = json.NewEncoder(p.errOut).Encode(map[string]any{
		"error": err.Error(),
	})
}
