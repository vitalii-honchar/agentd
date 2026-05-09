package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type EnvFileEntry struct {
	Key   string
	Value string
}

type EnvironmentMergeInput struct {
	FileEntries []EnvFileEntry
	Variables   []EnvFileEntry
	ToolEnv     []EnvFileEntry
}

func ParseEnvFile(path string, body []byte) ([]EnvFileEntry, error) {
	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
	entries := make([]EnvFileEntry, 0, len(lines))
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("%w: %s:%d invalid env line", domain.ErrInvalidDefinition, path, index+1)
		}
		key = strings.TrimSpace(key)
		if !isValidEnvKey(key) {
			return nil, fmt.Errorf("%w: %s:%d invalid env key %q", domain.ErrInvalidDefinition, path, index+1, key)
		}
		entries = append(entries, EnvFileEntry{
			Key:   key,
			Value: parseEnvValue(value),
		})
	}

	return entries, nil
}

func MergeEnvironment(input EnvironmentMergeInput) []EnvFileEntry {
	values := make(map[string]string, len(input.FileEntries)+len(input.Variables)+len(input.ToolEnv))
	for _, entry := range input.FileEntries {
		values[entry.Key] = entry.Value
	}
	for _, entry := range input.Variables {
		values[entry.Key] = entry.Value
	}
	for _, entry := range input.ToolEnv {
		values[entry.Key] = entry.Value
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	merged := make([]EnvFileEntry, 0, len(keys))
	for _, key := range keys {
		merged = append(merged, EnvFileEntry{Key: key, Value: values[key]})
	}

	return merged
}

func MaskEnvironmentValue(value string) string {
	if value == "" {
		return ""
	}

	return "********"
}

func parseEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	if index := strings.Index(value, " #"); index >= 0 {
		value = value[:index]
	}

	return strings.TrimSpace(value)
}

func isValidEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for index, r := range key {
		if r == '_' || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') {
			continue
		}
		if index > 0 && '0' <= r && r <= '9' {
			continue
		}

		return false
	}

	return true
}
