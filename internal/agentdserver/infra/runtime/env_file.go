package runtime

type EnvFileEntry struct {
	Key   string
	Value string
}

type EnvironmentMergeInput struct {
	FileEntries []EnvFileEntry
	Variables   []EnvFileEntry
	ToolEnv     []EnvFileEntry
}
