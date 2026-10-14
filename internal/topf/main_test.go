// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/postfinance/topf/pkg/config"
)

// fakePatchesProviderEnv makes the test binary act as a patches provider when
// it is re-executed by topf. Its value selects the behavior.
const fakePatchesProviderEnv = "TOPF_TEST_FAKE_PATCHES_PROVIDER"

func TestMain(m *testing.M) {
	if mode, ok := os.LookupEnv(fakePatchesProviderEnv); ok {
		os.Exit(runFakePatchesProvider(mode, os.Args[1:]))
	}

	os.Exit(m.Run())
}

// runFakePatchesProvider emulates a patches provider binary and returns its exit code.
func runFakePatchesProvider(mode string, args []string) int {
	if mode == "fail" {
		fmt.Fprintln(os.Stderr, "fake patches provider failure")
		return 1
	}

	if len(args) != 2 || args[0] != "patches" {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %q\n", args)
		return 1
	}

	var patchCtx config.PatchContext
	if err := json.Unmarshal([]byte(args[1]), &patchCtx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode patch context: %v\n", err)
		return 1
	}

	// derive the hostname from cluster and node data to prove both are received
	fmt.Printf("apiVersion: v1alpha1\nkind: HostnameConfig\nauto: \"off\"\nhostname: %s-%s-%s\n",
		patchCtx.Node.Host, patchCtx.Data["env"], patchCtx.Node.Data["rack"])

	return 0
}
