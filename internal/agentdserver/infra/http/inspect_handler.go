package http

import (
	stdhttp "net/http"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleInspect(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	agent, err := s.inspectUseCase.Inspect(r.Context(), r.PathValue("name"))
	if err != nil {
		writeQueryError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, toAgentDetail(agent))
}

func (s *Server) handleListRevisions(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	revisions, err := s.revisionUseCase.ListRevisions(r.Context(), r.PathValue("name"))
	if err != nil {
		writeQueryError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, model.RevisionListResponse{Revisions: toRevisionSummaries(revisions)})
}

func (s *Server) handleInspectRevision(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	revision, err := s.revisionUseCase.InspectRevision(
		r.Context(),
		r.PathValue("name"),
		r.PathValue("revision_id"),
	)
	if err != nil {
		writeQueryError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, model.RevisionInspectResponse{Revision: toRevisionDetail(revision)})
}

func toAgentDetail(agent domain.Agent) model.AgentDetail {
	return model.AgentDetail{
		AgentSummary: toAgentSummary(agent),
		Revision:     agent.Revision,
		VendorName:   agent.Vendor.Name,
		VendorModel:  agent.Vendor.Model,
		LastRunID:    agent.LastRunID,
		RecentError:  agent.LastError,
	}
}

func toRevisionSummaries(revisions []domain.AgentRevision) []model.RevisionSummary {
	summaries := make([]model.RevisionSummary, 0, len(revisions))
	for _, revision := range revisions {
		summaries = append(summaries, toRevisionSummary(revision))
	}

	return summaries
}

func toRevisionSummary(revision domain.AgentRevision) model.RevisionSummary {
	return model.RevisionSummary{
		RevisionID:   revision.RevisionID,
		Status:       string(revision.Status),
		CreatedAt:    revision.CreatedAt,
		Latest:       revision.IsLatestFinalized,
		SourcePath:   revision.SourcePath,
		ArtifactPath: revision.ArtifactPath,
		FinalizedAt:  revision.FinalizedAt,
		ErrorMessage: revision.ErrorMessage,
	}
}

func toRevisionDetail(revision domain.AgentRevision) model.RevisionDetail {
	return model.RevisionDetail{
		RevisionSummary: toRevisionSummary(revision),
		Prompt:          revision.Prompt,
		Tools:           toRevisionTools(revision.Tools),
		ArtifactFiles:   toRevisionArtifactFiles(revision.ArtifactFiles),
		Environment:     toRevisionEnvironment(revision.Environment),
	}
}

func toRevisionTools(tools []domain.RevisionTool) []model.RevisionTool {
	response := make([]model.RevisionTool, 0, len(tools))
	for _, tool := range tools {
		response = append(response, model.RevisionTool{
			Name:             tool.Name,
			Kind:             string(tool.Kind),
			OriginalCommand:  tool.OriginalCommand,
			RewrittenCommand: tool.RewrittenCommand,
			HostCommand:      tool.HostCommand,
			CopiedFiles:      append([]string(nil), tool.CopiedFiles...),
		})
	}

	return response
}

func toRevisionArtifactFiles(files []domain.RevisionArtifactFile) []model.RevisionArtifactFile {
	response := make([]model.RevisionArtifactFile, 0, len(files))
	for _, file := range files {
		response = append(response, model.RevisionArtifactFile{
			Path:       file.ArtifactRelativePath,
			SourcePath: file.SourcePath,
			SHA256:     file.SHA256,
			SizeBytes:  file.SizeBytes,
		})
	}

	return response
}

func toRevisionEnvironment(environment []domain.RevisionEnvironment) []model.RevisionEnvironment {
	response := make([]model.RevisionEnvironment, 0, len(environment))
	for _, entry := range environment {
		response = append(response, model.RevisionEnvironment{
			Key:    entry.Key,
			Value:  entry.Value,
			Source: string(entry.Source),
			Masked: entry.Masked,
		})
	}

	return response
}
