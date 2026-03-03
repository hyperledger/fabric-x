package io

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const defaultMaxInputSize = 20 * 1024 * 1024

type IOFlags struct {
	Input  string
	Output string
}

func (f *IOFlags) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Input, "input", "", "Input file (optional, defaults to stdin)")
	cmd.Flags().StringVar(&f.Output, "output", "", "Output file (optional, defaults to stdout)")
}

func ResolveInput(cmd *cobra.Command, inputFile string) ([]byte, error) {

	pipe := isInputFromPipe()
	switch {
	case inputFile != "" && pipe:
		return nil, errors.New("cannot use --input and stdin together")
	case inputFile != "":
		// Prevent path traversal
		inputFile = filepath.Clean(inputFile)
		if strings.Contains(inputFile, "..") {
			return nil, errors.New("path traversal not allowed")
		}
		// read from file
		file, err := os.Open(inputFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		// check file size
		if info, err := file.Stat(); err == nil && info.Size() > defaultMaxInputSize {
			return nil, fmt.Errorf("input file exceeds maximum allowed size of %d bytes", defaultMaxInputSize)
		}
		return ReadWithLimit(file, defaultMaxInputSize)
	case pipe:
		return ReadWithLimit(cmd.InOrStdin(), defaultMaxInputSize)
	default:
		return nil, errors.New("no input provided (use --input or pipe data via stdin)")
	}
}

func WriteOutput(cmd *cobra.Command, outputFile string, data []byte) error {
	if outputFile != "" {
		return os.WriteFile(outputFile, data, 0o600)
	}
	_, err := cmd.OutOrStdout().Write(data)
	return err
}

func isInputFromPipe() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

func ReadWithLimit(r io.Reader, maxBytes int64) ([]byte, error) {
	// Read at most maxBytes + 1
	limited := io.LimitReader(r, maxBytes+1)

	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("input exceeds maximum allowed size of %d bytes", maxBytes)
	}

	return data, nil
}
