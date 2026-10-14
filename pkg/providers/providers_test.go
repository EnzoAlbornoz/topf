// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// fakeProviderEnv makes the test binary act as a provider binary when it is
// re-executed by a binary provider. Its value selects the behavior.
const fakeProviderEnv = "TOPF_TEST_FAKE_PROVIDER"

func TestMain(m *testing.M) {
	if mode, ok := os.LookupEnv(fakeProviderEnv); ok {
		os.Exit(runFakeProvider(mode))
	}

	os.Exit(m.Run())
}

// runFakeProvider emulates a provider binary and returns its exit code.
func runFakeProvider(mode string) int {
	switch mode {
	case "echo-args":
		// tests decode the arguments from stdout to assert how the binary was invoked
		if err := json.NewEncoder(os.Stdout).Encode(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		return 0
	case "fail":
		fmt.Fprintln(os.Stderr, "fake provider failure")
		return 3
	default:
		fmt.Fprintf(os.Stderr, "unknown fake provider mode %q\n", mode)
		return 1
	}
}

// fakeProviderBinary returns the path of a binary which behaves as a provider
// in the given mode for the duration of the test.
func fakeProviderBinary(t *testing.T, mode string) string {
	t.Helper()

	t.Setenv(fakeProviderEnv, mode)

	path, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to locate test binary: %v", err)
	}

	return path
}
