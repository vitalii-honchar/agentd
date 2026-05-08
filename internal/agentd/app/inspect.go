package app

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func NewInspectCommand(client QueryClient, output Output) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <agent_name>",
		Short: "Inspect an Agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			if agentName, revisionID, ok := strings.Cut(args[0], ":"); ok {
				revision, err := client.InspectRevision(cmd.Context(), agentName, revisionID)
				if err != nil {
					return err
				}
				if output.format == "json" {
					return output.Write(revision)
				}

				return output.Write(formatRevisionInspect(revision.Revision))
			}
			agent, err := client.Inspect(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(agent)
			}

			return output.Write(fmt.Sprintf(
				"name: %s\nstatus: %s\nschedule: %s\nvendor: %s/%s\nrevision: %s\nlast_run: %s",
				agent.Name,
				agent.Status,
				agent.ScheduleType,
				agent.VendorName,
				agent.VendorModel,
				agent.Revision,
				agent.LastRunID,
			))
		},
	}
}

func formatRevisionInspect(revision RevisionDetail) string {
	var builder strings.Builder
	writeRevisionLine(&builder, "revision", revision.RevisionID)
	writeRevisionLine(&builder, "status", revision.Status)
	if !revision.CreatedAt.IsZero() {
		writeRevisionLine(&builder, "created", revision.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	writeRevisionLine(&builder, "latest", fmt.Sprintf("%t", revision.Latest))
	writeRevisionLine(&builder, "source", revision.SourcePath)
	writeRevisionLine(&builder, "artifact", revision.ArtifactPath)
	if revision.Prompt != "" {
		builder.WriteString("prompt:\n")
		builder.WriteString(revision.Prompt)
		builder.WriteString("\n")
	}
	if len(revision.Tools) > 0 {
		builder.WriteString("tools:\n")
		for _, tool := range revision.Tools {
			fmt.Fprintf(&builder, "- %s %s\n", tool.Name, tool.Kind)
			writeRevisionLine(&builder, "  original", tool.OriginalCommand)
			writeRevisionLine(&builder, "  rewritten", tool.RewrittenCommand)
			writeRevisionLine(&builder, "  host", tool.HostCommand)
			if len(tool.CopiedFiles) > 0 {
				writeRevisionLine(&builder, "  copied", strings.Join(tool.CopiedFiles, ","))
			}
		}
	}
	if len(revision.ArtifactFiles) > 0 {
		builder.WriteString("artifact_files:\n")
		for _, file := range revision.ArtifactFiles {
			fmt.Fprintf(&builder, "- %s sha256=%s size=%d source=%s\n", file.Path, file.SHA256, file.SizeBytes, file.SourcePath)
		}
	}
	if len(revision.Environment) > 0 {
		builder.WriteString("environment:\n")
		for _, entry := range revision.Environment {
			fmt.Fprintf(&builder, "- %s=%s source=%s masked=%t\n", entry.Key, entry.Value, entry.Source, entry.Masked)
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

func writeRevisionLine(builder *strings.Builder, key string, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(builder, "%s: %s\n", key, value)
}
