// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestBinaryPatchesProvider(t *testing.T) {
	t.Run("passesPatchContextAsArgument", func(t *testing.T) {
		// spaces, quotes, backslashes and non-ASCII must survive the command line
		patchContext := []byte(`{"ClusterName":"my cluster","Data":{"quote":"\"it's\"","path":"C:\\dir\\","name":"zürich"}}`)

		provider := NewBinaryPatchesProvider(fakeProviderBinary(t, "echo-args"))

		output, err := provider.GetPatches(t.Context(), patchContext)
		if err != nil {
			t.Fatalf("GetPatches() error: %v", err)
		}

		var args []string
		if err := json.Unmarshal(output, &args); err != nil {
			t.Fatalf("failed to decode fake provider output %q: %v", output, err)
		}

		want := []string{"patches", string(patchContext)}
		if !slices.Equal(args, want) {
			t.Errorf("unexpected arguments:\ngot:  %q\nwant: %q", args, want)
		}
	})

	t.Run("nonZeroExitCode", func(t *testing.T) {
		provider := NewBinaryPatchesProvider(fakeProviderBinary(t, "fail"))

		_, err := provider.GetPatches(t.Context(), []byte("{}"))

		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *exec.ExitError, got: %v", err)
		}

		if exitErr.ExitCode() != 3 {
			t.Errorf("unexpected exit code: got %d, want 3", exitErr.ExitCode())
		}
	})

	t.Run("missingBinary", func(t *testing.T) {
		provider := NewBinaryPatchesProvider(filepath.Join(t.TempDir(), "does-not-exist"))

		// the error depends on the OS: exec.ErrNotFound on windows (extension
		// lookup) and fs.ErrNotExist elsewhere
		_, err := provider.GetPatches(t.Context(), []byte("{}"))
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf("expected a not found error, got: %v", err)
		}
	})

	t.Run("canceledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		provider := NewBinaryPatchesProvider(fakeProviderBinary(t, "echo-args"))

		_, err := provider.GetPatches(ctx, []byte("{}"))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	})
}
