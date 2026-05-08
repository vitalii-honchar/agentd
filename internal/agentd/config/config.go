package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	DefaultServerURL      = "http://127.0.0.1:18080"
	DefaultOutputFormat   = OutputText
	DefaultRequestTimeout = 30 * time.Second

	OutputText = "text"
	OutputJSON = "json"
)

type Config struct {
	ServerURL      string
	OutputFormat   string
	RequestTimeout time.Duration
}

func FromEnv() (*Config, error) {
	_ = godotenv.Load()

	return FromLookup(os.LookupEnv)
}

func FromLookup(lookup func(string) (string, bool)) (*Config, error) {
	cfg := &Config{
		ServerURL:      getWithDefault(lookup, "AGENTD_SERVER_URL", DefaultServerURL),
		OutputFormat:   getWithDefault(lookup, "AGENTD_OUTPUT", DefaultOutputFormat),
		RequestTimeout: getDuration(lookup, "AGENTD_REQUEST_TIMEOUT", DefaultRequestTimeout),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.ServerURL) == "" {
		return fmt.Errorf("server url is required")
	}
	parsed, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("server url is invalid: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("server url scheme must be http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("server url host is required")
	}
	if c.OutputFormat != OutputText && c.OutputFormat != OutputJSON {
		return fmt.Errorf("output format must be text or json")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	return nil
}

func getWithDefault(lookup func(string) (string, bool), key, fallback string) string {
	if value, ok := lookup(key); ok && value != "" {
		return value
	}

	return fallback
}

func getDuration(
	lookup func(string) (string, bool),
	key string,
	fallback time.Duration,
) time.Duration {
	if value, ok := lookup(key); ok && value != "" {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
		slog.Warn("Invalid duration value", "key", key, "value", sanitize(value), "default", fallback)
	}

	return fallback
}

func sanitize(value string) string {
	replacer := strings.NewReplacer("\n", "", "\r", "", "\x00", "")

	return replacer.Replace(value)
}
