package config

import (
	"strings"
	"testing"
	"time"
)

func TestFromLookupUsesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := FromLookup(emptyLookup)
	if err != nil {
		t.Fatalf("FromLookup: %v", err)
	}

	if cfg.ServerURL != DefaultServerURL {
		t.Fatalf("server url: got %q want %q", cfg.ServerURL, DefaultServerURL)
	}
	if cfg.OutputFormat != DefaultOutputFormat {
		t.Fatalf("output format: got %q want %q", cfg.OutputFormat, DefaultOutputFormat)
	}
	if cfg.RequestTimeout != DefaultRequestTimeout {
		t.Fatalf("request timeout: got %v want %v", cfg.RequestTimeout, DefaultRequestTimeout)
	}
}

func TestFromLookupUsesOverrides(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"AGENTD_SERVER_URL":      "https://agentd.local:19090",
		"AGENTD_OUTPUT":          OutputJSON,
		"AGENTD_REQUEST_TIMEOUT": "3s",
	}

	cfg, err := FromLookup(mapLookup(env))
	if err != nil {
		t.Fatalf("FromLookup: %v", err)
	}

	if cfg.ServerURL != "https://agentd.local:19090" {
		t.Fatalf("server url: got %q", cfg.ServerURL)
	}
	if cfg.OutputFormat != OutputJSON {
		t.Fatalf("output format: got %q want %q", cfg.OutputFormat, OutputJSON)
	}
	if cfg.RequestTimeout != 3*time.Second {
		t.Fatalf("request timeout: got %v want 3s", cfg.RequestTimeout)
	}
}

func TestFromLookupRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]string{
		"bad scheme": {
			"AGENTD_SERVER_URL": "ftp://127.0.0.1:18080",
		},
		"missing host": {
			"AGENTD_SERVER_URL": "http://",
		},
		"bad output": {
			"AGENTD_OUTPUT": "yaml",
		},
	}
	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := FromLookup(mapLookup(env))
			if err == nil {
				t.Fatal("FromLookup returned nil error")
			}
		})
	}
}

func TestFromLookupFallsBackForInvalidTimeout(t *testing.T) {
	t.Parallel()

	cfg, err := FromLookup(mapLookup(map[string]string{
		"AGENTD_REQUEST_TIMEOUT": "bad\nvalue",
	}))
	if err != nil {
		t.Fatalf("FromLookup: %v", err)
	}

	if cfg.RequestTimeout != DefaultRequestTimeout {
		t.Fatalf("request timeout: got %v want %v", cfg.RequestTimeout, DefaultRequestTimeout)
	}
	if sanitized := sanitize("bad\nvalue\r\x00"); strings.ContainsAny(sanitized, "\n\r\x00") {
		t.Fatalf("sanitize left control characters in %q", sanitized)
	}
}

func emptyLookup(string) (string, bool) {
	return "", false
}

func mapLookup(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := env[key]

		return value, ok
	}
}
