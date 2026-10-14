// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"go.yaml.in/yaml/v4"
)

// newPatchesProviderRuntime creates a runtime for a two node cluster whose
// patches provider is the test binary running in the given mode.
func newPatchesProviderRuntime(t *testing.T, mode string) Topf {
	t.Helper()

	t.Setenv(fakePatchesProviderEnv, mode)

	providerPath, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to locate test binary: %v", err)
	}

	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "topf.yaml"), `clusterName: test
clusterEndpoint: https://10.0.0.100:6443
kubernetesVersion: 1.35.1
patchesProvider: '`+providerPath+`'
data:
  env: prod
nodes:
  - host: cp1
    role: control-plane
    data:
      rack: a3
  - host: w1
    role: worker
    data:
      rack: b1
`)

	// node patches from the patches directory, the hostname is expected to be
	// overridden by the provider
	writeFile(t, filepath.Join(dir, "node", "cp1", "01-node.yaml"), `apiVersion: v1alpha1
kind: HostnameConfig
auto: "off"
hostname: from-file
---
apiVersion: v1alpha1
kind: KubeNodeConfig
labels:
  topf.test/from-file: "true"
`)

	// pre-generated secrets keep the runtime from generating and storing them
	bundle, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), nil)
	if err != nil {
		t.Fatalf("failed to generate secrets bundle: %v", err)
	}

	bundleYAML, err := yaml.Marshal(bundle)
	if err != nil {
		t.Fatalf("failed to marshal secrets bundle: %v", err)
	}

	writeFile(t, filepath.Join(dir, "secrets.yaml"), string(bundleYAML))

	rt, err := NewTopfRuntime(RuntimeConfig{
		ConfigPath:  filepath.Join(dir, "topf.yaml"),
		Logger:      slog.New(slog.DiscardHandler),
		TopfVersion: "test",
	})
	if err != nil {
		t.Fatalf("NewTopfRuntime() error: %v", err)
	}

	return rt
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRenderWithPatchesProvider(t *testing.T) {
	t.Run("providerPatchesAreAppliedLast", func(t *testing.T) {
		nodes, err := newPatchesProviderRuntime(t, "hostname").Render(t.Context(), false)
		if err != nil {
			t.Fatalf("Render() error: %v", err)
		}

		want := map[string][]string{
			"cp1": {"hostname: cp1-prod-a3", "topf.test/from-file"},
			"w1":  {"hostname: w1-prod-b1"},
		}

		if len(nodes) != len(want) {
			t.Fatalf("unexpected node count: got %d, want %d", len(nodes), len(want))
		}

		for _, node := range nodes {
			if node.Error != nil {
				t.Fatalf("%s: unexpected error: %v", node.Node.Host, node.Error)
			}

			configBytes, err := node.ConfigProvider().EncodeBytes(encoder.WithComments(encoder.CommentsDisabled))
			if err != nil {
				t.Fatalf("%s: failed to encode config: %v", node.Node.Host, err)
			}

			for _, s := range want[node.Node.Host] {
				if !strings.Contains(string(configBytes), s) {
					t.Errorf("%s: rendered config missing %q; got:\n%s", node.Node.Host, s, configBytes)
				}
			}
		}
	})

	t.Run("providerFailure", func(t *testing.T) {
		nodes, err := newPatchesProviderRuntime(t, "fail").Render(t.Context(), false)
		if err != nil {
			t.Fatalf("Render() error: %v", err)
		}

		for _, node := range nodes {
			if node.Error == nil || !strings.Contains(node.Error.Error(), "patches from provider") {
				t.Errorf("%s: expected patches provider error, got: %v", node.Node.Host, node.Error)
			}

			if node.ConfigBundle != nil {
				t.Errorf("%s: expected no config bundle", node.Node.Host)
			}
		}
	})
}
