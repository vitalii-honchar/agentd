package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	DefaultServerHost       = "127.0.0.1"
	DefaultServerPort       = "18080"
	DefaultDataDir          = "./data"
	DefaultSettingsDBPath   = "./data/agentd-settings.db"
	DefaultRuntimeDBDir     = "./data/agents"
	DefaultRunLogDir        = "./data/logs"
	DefaultSQLiteMaxConns   = 4
	DefaultStartupTimeout   = 30 * time.Second
	DefaultShutdownTimeout  = 30 * time.Second
	DefaultRunStopTimeout   = 10 * time.Second
	DefaultHTTPReadTimeout  = 5 * time.Second
	DefaultHTTPWriteTimeout = 30 * time.Second
)

type Config struct {
	Production bool
	Server     ServerConfig
	Storage    StorageConfig
	Runtime    RuntimeConfig
	OpenAI     OpenAIConfig
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c ServerConfig) Address() string {
	return c.Host + ":" + c.Port
}

type StorageConfig struct {
	DataDir        string
	SettingsDBPath string
	RuntimeDBDir   string
	RunLogDir      string
	SQLiteMaxConns int
}

type RuntimeConfig struct {
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	RunStopTimeout  time.Duration
}

type OpenAIConfig struct {
	APIKey string
}

func FromEnv() (*Config, error) {
	_ = godotenv.Load()

	return FromLookup(os.LookupEnv)
}

func FromLookup(lookup func(string) (string, bool)) (*Config, error) {
	cfg := &Config{
		Production: getWithDefault(lookup, "AGENTD_PRODUCTION", "false") == "true",
		Server: ServerConfig{
			Host:         getWithDefault(lookup, "AGENTD_SERVER_HOST", DefaultServerHost),
			Port:         getWithDefault(lookup, "AGENTD_SERVER_PORT", DefaultServerPort),
			ReadTimeout:  getDuration(lookup, "AGENTD_HTTP_READ_TIMEOUT", DefaultHTTPReadTimeout),
			WriteTimeout: getDuration(lookup, "AGENTD_HTTP_WRITE_TIMEOUT", DefaultHTTPWriteTimeout),
		},
		Storage: StorageConfig{
			DataDir:        getWithDefault(lookup, "AGENTD_DATA_DIR", DefaultDataDir),
			SettingsDBPath: getWithDefault(lookup, "AGENTD_SETTINGS_DB_PATH", DefaultSettingsDBPath),
			RuntimeDBDir:   getWithDefault(lookup, "AGENTD_RUNTIME_DB_DIR", DefaultRuntimeDBDir),
			RunLogDir:      getWithDefault(lookup, "AGENTD_RUN_LOG_DIR", DefaultRunLogDir),
			SQLiteMaxConns: getInt(
				lookup,
				"AGENTD_SQLITE_MAX_CONNS",
				DefaultSQLiteMaxConns,
			),
		},
		Runtime: RuntimeConfig{
			StartupTimeout:  getDuration(lookup, "AGENTD_STARTUP_TIMEOUT", DefaultStartupTimeout),
			ShutdownTimeout: getDuration(lookup, "AGENTD_SHUTDOWN_TIMEOUT", DefaultShutdownTimeout),
			RunStopTimeout:  getDuration(lookup, "AGENTD_RUN_STOP_TIMEOUT", DefaultRunStopTimeout),
		},
		OpenAI: OpenAIConfig{
			APIKey: getWithDefault(lookup, "OPENAI_API_KEY", ""),
		},
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Server.Host) == "" {
		return fmt.Errorf("server host is required")
	}
	if strings.TrimSpace(c.Server.Port) == "" {
		return fmt.Errorf("server port is required")
	}
	if strings.TrimSpace(c.Storage.DataDir) == "" {
		return fmt.Errorf("data dir is required")
	}
	if strings.TrimSpace(c.Storage.SettingsDBPath) == "" {
		return fmt.Errorf("settings db path is required")
	}
	if strings.TrimSpace(c.Storage.RuntimeDBDir) == "" {
		return fmt.Errorf("runtime db dir is required")
	}
	if strings.TrimSpace(c.Storage.RunLogDir) == "" {
		return fmt.Errorf("run log dir is required")
	}
	if c.Storage.SQLiteMaxConns < 1 {
		return fmt.Errorf("sqlite max conns must be at least 1")
	}
	if c.Runtime.StartupTimeout <= 0 {
		return fmt.Errorf("startup timeout must be positive")
	}
	if c.Runtime.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}
	if c.Runtime.RunStopTimeout <= 0 {
		return fmt.Errorf("run stop timeout must be positive")
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

func getInt(lookup func(string) (string, bool), key string, fallback int) int {
	if value, ok := lookup(key); ok && value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
		slog.Warn("Invalid integer value", "key", key, "value", sanitize(value), "default", fallback)
	}

	return fallback
}

func sanitize(value string) string {
	replacer := strings.NewReplacer("\n", "", "\r", "", "\x00", "")

	return replacer.Replace(value)
}
