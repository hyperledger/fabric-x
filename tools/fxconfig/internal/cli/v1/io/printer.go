package io

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

type Printer interface {
	Print(v any) error
	PrintError(err error)
}

type CLIPrinter struct {
	out    io.Writer
	errOut io.Writer
	format Format
}

func NewCLIPrinter(out, errOut io.Writer, format Format) *CLIPrinter {
	return &CLIPrinter{
		out:    out,
		errOut: errOut,
		format: format,
	}
}

func (p *CLIPrinter) Print(v any) error {
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

func (p *CLIPrinter) PrintError(err error) {
	// Human readable
	if p.format != FormatJSON {
		fmt.Fprintf(p.errOut, "Error: %v\n", err)
		return
	}

	// Machine readable
	_ = json.NewEncoder(p.errOut).Encode(map[string]any{
		"error": err.Error(),
	})
}
