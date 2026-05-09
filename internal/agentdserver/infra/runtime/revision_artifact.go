package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type RevisionArtifactService struct {
	workRoot string
}

type RevisionArtifactRequest struct {
	Definition domain.AgentDefinition
	RevisionID string
	CreatedAt  time.Time
}

type RevisionArtifactResult struct {
	Revision domain.AgentRevision
}

func NewRevisionArtifactService(workRoot string) (*RevisionArtifactService, error) {
	if workRoot == "" {
		return nil, fmt.Errorf("revision artifact work root is required")
	}

	return &RevisionArtifactService{workRoot: workRoot}, nil
}

func (s *RevisionArtifactService) Create(
	ctx context.Context,
	request RevisionArtifactRequest,
) (RevisionArtifactResult, error) {
	if err := ctx.Err(); err != nil {
		return RevisionArtifactResult{}, err
	}
	createdAt := request.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	artifactPath, err := s.ArtifactPath(request.Definition.Name, request.RevisionID)
	if err != nil {
		return RevisionArtifactResult{}, err
	}
	sourceDir := filepath.Dir(request.Definition.SourcePath)
	if strings.TrimSpace(sourceDir) == "" || sourceDir == "." {
		sourceDir = "."
	}
	sourceRoot, err := filepath.Abs(sourceDir)
	if err != nil {
		return RevisionArtifactResult{}, fmt.Errorf("resolve definition source dir: %w", err)
	}
	stagePath := artifactPath + ".staging"
	if err := os.RemoveAll(stagePath); err != nil {
		return RevisionArtifactResult{}, fmt.Errorf("remove stale artifact staging dir: %w", err)
	}
	if err := os.MkdirAll(stagePath, 0o755); err != nil {
		return RevisionArtifactResult{}, fmt.Errorf("create artifact staging dir: %w", err)
	}
	defer os.RemoveAll(stagePath)

	copier := artifactCopier{
		sourceRoot: sourceRoot,
		stagePath:  stagePath,
		copiedAt:   createdAt,
		files:      make(map[string]domain.RevisionArtifactFile),
	}
	tools := make([]domain.RevisionTool, 0, len(request.Definition.Tools))
	for _, tool := range request.Definition.Tools {
		revisionTool := domain.RevisionTool{
			AgentName:       request.Definition.Name,
			RevisionID:      request.RevisionID,
			Name:            tool.Name,
			Kind:            tool.Kind,
			OriginalCommand: tool.Command,
			Args:            append([]string(nil), tool.Args...),
			Env:             append([]string(nil), tool.Env...),
			Timeout:         tool.Timeout,
			ReadPaths:       append([]string(nil), tool.ReadPaths...),
			WritePaths:      append([]string(nil), tool.WritePaths...),
			NetworkAllow:    append([]string(nil), tool.NetworkAllow...),
			CreatedAt:       createdAt,
		}
		if revisionTool.Kind == domain.ToolKindLocalTool {
			revisionTool.Kind = domain.ToolKindCustomTool
		}
		switch revisionTool.Kind {
		case domain.ToolKindCustomTool:
			if _, err := copier.copyDeclaredPath(tool.Command); err != nil {
				return RevisionArtifactResult{}, err
			}
			revisionTool.RewrittenCommand = filepath.Join(artifactPath, filepath.Clean(tool.Command))
			for _, readPath := range tool.ReadPaths {
				if _, err := copier.copyDeclaredPath(readPath); err != nil {
					return RevisionArtifactResult{}, err
				}
			}
		case domain.ToolKindHostTool:
			revisionTool.HostCommand = tool.Command
		case domain.ToolKindMCPServer:
		default:
			return RevisionArtifactResult{}, fmt.Errorf("%w: unsupported tool kind %q", domain.ErrInvalidDefinition, tool.Kind)
		}
		tools = append(tools, revisionTool)
	}
	for _, readPath := range request.Definition.Access.Filesystem.Read {
		if _, err := copier.copyDeclaredPath(readPath); err != nil {
			return RevisionArtifactResult{}, err
		}
	}
	for _, envFile := range request.Definition.Environment.Files {
		if _, err := copier.copyDeclaredPath(envFile); err != nil {
			return RevisionArtifactResult{}, err
		}
	}

	artifactFiles := copier.artifactFiles(request.Definition.Name, request.RevisionID)
	copiedFiles := artifactRelativePaths(artifactFiles)
	for i := range tools {
		if tools[i].Kind == domain.ToolKindCustomTool {
			tools[i].CopiedFiles = append([]string(nil), copiedFiles...)
		}
	}
	if err := writeArtifactManifest(stagePath, artifactFiles); err != nil {
		return RevisionArtifactResult{}, err
	}
	if err := os.RemoveAll(artifactPath); err != nil {
		return RevisionArtifactResult{}, fmt.Errorf("remove existing artifact dir: %w", err)
	}
	if err := os.Rename(stagePath, artifactPath); err != nil {
		return RevisionArtifactResult{}, fmt.Errorf("finalize artifact dir: %w", err)
	}
	finalizedAt := createdAt

	revision := domain.AgentRevision{
		AgentName:       request.Definition.Name,
		RevisionID:      request.RevisionID,
		ContentDigest:   artifactContentDigest(request.Definition),
		SourcePath:      request.Definition.SourcePath,
		ArtifactPath:    artifactPath,
		EnvironmentJSON: "[]",
		Prompt:          request.Definition.Prompt,
		Vendor:          request.Definition.Vendor,
		Schedule:        request.Definition.Schedule,
		Status:          domain.AgentRevisionStatusFinalized,
		CreatedAt:       createdAt,
		FinalizedAt:     &finalizedAt,
		Tools:           tools,
		ArtifactFiles:   artifactFiles,
	}
	applyRevisionContractMetadata(&revision, request.Definition.Contract)

	return RevisionArtifactResult{Revision: revision}, nil
}

func (s *RevisionArtifactService) ArtifactPath(agentName, revisionID string) (string, error) {
	if !domain.IsValidAgentName(agentName) {
		return "", fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, agentName)
	}
	if revisionID == "" {
		return "", fmt.Errorf("revision id is required")
	}

	return filepath.Join(s.workRoot, agentName, revisionID), nil
}

func (s *RevisionArtifactService) ExecutionWorkDirPath(agentName, executionID string) (string, error) {
	if !domain.IsValidAgentName(agentName) {
		return "", fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, agentName)
	}
	if executionID == "" {
		return "", fmt.Errorf("execution id is required")
	}

	return filepath.Join(s.workRoot, agentName, "executions", executionID), nil
}

type artifactCopier struct {
	sourceRoot string
	stagePath  string
	copiedAt   time.Time
	files      map[string]domain.RevisionArtifactFile
}

func (c *artifactCopier) copyDeclaredPath(relativePath string) ([]domain.RevisionArtifactFile, error) {
	cleaned, err := cleanArtifactRelativePath(relativePath)
	if err != nil {
		return nil, err
	}
	sourcePath := filepath.Join(c.sourceRoot, cleaned)
	if err := ensurePathUnderRoot(c.sourceRoot, sourcePath); err != nil {
		return nil, err
	}
	info, err := os.Lstat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: declared path %q does not exist", domain.ErrInvalidDefinition, relativePath)
		}

		return nil, fmt.Errorf("stat declared path %q: %w", relativePath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: declared path %q must not be a symlink", domain.ErrInvalidDefinition, relativePath)
	}
	if info.IsDir() {
		var files []domain.RevisionArtifactFile
		err := filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			entryInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if entryInfo.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: declared path %q must not contain symlinks", domain.ErrInvalidDefinition, relativePath)
			}
			rel, err := filepath.Rel(c.sourceRoot, path)
			if err != nil {
				return err
			}
			file, err := c.copyFile(filepath.ToSlash(rel), path, entryInfo)
			if err != nil {
				return err
			}
			files = append(files, file)

			return nil
		})
		if err != nil {
			return nil, err
		}

		return files, nil
	}
	file, err := c.copyFile(cleaned, sourcePath, info)
	if err != nil {
		return nil, err
	}

	return []domain.RevisionArtifactFile{file}, nil
}

func (c *artifactCopier) copyFile(
	relativePath string,
	sourcePath string,
	info os.FileInfo,
) (domain.RevisionArtifactFile, error) {
	relativePath = filepath.ToSlash(relativePath)
	if file, ok := c.files[relativePath]; ok {
		return file, nil
	}
	targetPath := filepath.Join(c.stagePath, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return domain.RevisionArtifactFile{}, fmt.Errorf("create artifact file dir: %w", err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return domain.RevisionArtifactFile{}, fmt.Errorf("open declared file %q: %w", sourcePath, err)
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return domain.RevisionArtifactFile{}, fmt.Errorf("create artifact file %q: %w", targetPath, err)
	}
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(target, hasher), source); err != nil {
		_ = target.Close()

		return domain.RevisionArtifactFile{}, fmt.Errorf("copy artifact file %q: %w", relativePath, err)
	}
	if err := target.Close(); err != nil {
		return domain.RevisionArtifactFile{}, fmt.Errorf("close artifact file %q: %w", relativePath, err)
	}
	if err := os.Chmod(targetPath, info.Mode().Perm()); err != nil {
		return domain.RevisionArtifactFile{}, fmt.Errorf("chmod artifact file %q: %w", relativePath, err)
	}
	file := domain.RevisionArtifactFile{
		ArtifactRelativePath: relativePath,
		SourcePath:           sourcePath,
		SHA256:               hex.EncodeToString(hasher.Sum(nil)),
		Mode:                 int64(info.Mode().Perm()),
		SizeBytes:            info.Size(),
		CopiedAt:             c.copiedAt,
	}
	c.files[relativePath] = file

	return file, nil
}

func (c *artifactCopier) artifactFiles(agentName, revisionID string) []domain.RevisionArtifactFile {
	files := make([]domain.RevisionArtifactFile, 0, len(c.files))
	for _, file := range c.files {
		file.AgentName = agentName
		file.RevisionID = revisionID
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ArtifactRelativePath < files[j].ArtifactRelativePath
	})

	return files
}

func cleanArtifactRelativePath(path string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("%w: declared path is required", domain.ErrInvalidDefinition)
	}
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", fmt.Errorf("%w: declared path %q must stay inside the definition folder", domain.ErrInvalidDefinition, path)
	}

	return cleaned, nil
}

func ensurePathUnderRoot(root, path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: declared path %q escapes the definition folder", domain.ErrInvalidDefinition, path)
	}

	return nil
}

func artifactRelativePaths(files []domain.RevisionArtifactFile) []string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.ArtifactRelativePath
	}

	return paths
}

func writeArtifactManifest(artifactPath string, files []domain.RevisionArtifactFile) error {
	body, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal artifact manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "manifest.json"), body, 0o644); err != nil {
		return fmt.Errorf("write artifact manifest: %w", err)
	}

	return nil
}

func artifactContentDigest(definition domain.AgentDefinition) string {
	parts := []string{strings.TrimSpace(definition.RawMarkdown)}
	if definition.Contract != nil {
		parts = append(parts,
			strings.TrimSpace(definition.Contract.InputSchemaRaw),
			strings.TrimSpace(definition.Contract.OutputSchemaRaw),
		)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))

	return hex.EncodeToString(sum[:])
}

func applyRevisionContractMetadata(revision *domain.AgentRevision, contract *domain.AgentContract) {
	if revision == nil || contract == nil {
		return
	}
	inputDigest := contract.InputSchemaDigest
	if inputDigest == "" {
		inputDigest = digestArtifactString(contract.InputSchemaRaw)
	}
	outputDigest := contract.OutputSchemaDigest
	if outputDigest == "" {
		outputDigest = digestArtifactString(contract.OutputSchemaRaw)
	}
	revision.ContractInputSchemaRaw = contract.InputSchemaRaw
	revision.ContractOutputSchemaRaw = contract.OutputSchemaRaw
	revision.ContractInputSchemaDigest = inputDigest
	revision.ContractOutputSchemaDigest = outputDigest
	revision.ContractDigest = digestArtifactString(inputDigest + "\x00" + outputDigest)
}

func digestArtifactString(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))

	return hex.EncodeToString(sum[:])
}
