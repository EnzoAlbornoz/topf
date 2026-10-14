// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package decryption provides a unified interface for reading files and
// content with automatic SOPS decryption and vals reference evaluation.
package decryption

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/postfinance/topf/internal/sops"
	"github.com/postfinance/topf/internal/vals"
)

// ReadFileWithSecrets reads a file, automatically decrypts it with SOPS and vals.
// It returns the final content and list of plaintext secret values discovered.
func ReadFileWithSecrets(path string) ([]byte, []string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("read file %q: %w", path, fs.ErrNotExist)
		}

		return nil, nil, fmt.Errorf("read file %q: %w", path, err)
	}

	var (
		content     []byte
		sopsSecrets []string
	)

	// first pass with sops
	isEncrypted, err := sops.IsEncrypted(path)
	if err != nil {
		return nil, nil, err
	}

	if isEncrypted {
		content, sopsSecrets, err = sops.Decrypt(path)
		if err != nil {
			return nil, nil, err
		}
	} else {
		//nolint:gosec // files read through a variable in our control
		content, err = os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
	}

	return evalWithSecrets(content, sopsSecrets)
}

// DecryptContent processes YAML content through the same pipeline as
// ReadFileWithSecrets: SOPS decryption if the content is SOPS-encrypted,
// followed by vals evaluation. It returns the final content and list of
// plaintext secret values discovered.
func DecryptContent(content []byte) ([]byte, []string, error) {
	var sopsSecrets []string

	// first pass with sops
	if sops.IsEncryptedContent(content) {
		var err error

		content, sopsSecrets, err = sops.DecryptContent(content)
		if err != nil {
			return nil, nil, err
		}
	}

	return evalWithSecrets(content, sopsSecrets)
}

// evalWithSecrets is the second pass of the pipeline: it evaluates vals
// references and returns the content along with all discovered secrets.
func evalWithSecrets(content []byte, sopsSecrets []string) ([]byte, []string, error) {
	content, valsSecrets, err := vals.EvalContent(content)
	if err != nil {
		return nil, nil, err
	}

	allSecrets := make([]string, 0, len(sopsSecrets)+len(valsSecrets))
	allSecrets = append(allSecrets, sopsSecrets...)
	allSecrets = append(allSecrets, valsSecrets...)

	return content, allSecrets, nil
}
