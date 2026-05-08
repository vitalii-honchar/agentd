package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

const DefaultTableCellLimit = 72

type Output struct {
	format string
	writer io.Writer
}

func NewOutput(format string, writer io.Writer) Output {
	if writer == nil {
		writer = os.Stdout
	}

	return Output{format: format, writer: writer}
}

func (o Output) Write(value any) error {
	if o.format == config.OutputJSON {
		encoder := json.NewEncoder(o.writer)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")

		return encoder.Encode(value)
	}
	if text, ok := value.(string); ok {
		_, err := fmt.Fprintln(o.writer, text)

		return err
	}
	_, err := fmt.Fprintln(o.writer, value)

	return err
}

func (o Output) WriteTable(headers []string, rows [][]string) error {
	table := tabwriter.NewWriter(o.writer, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		if err := writeTableRow(table, headers); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := writeTableRow(table, row); err != nil {
			return err
		}
	}

	return table.Flush()
}

func writeTableRow(writer io.Writer, values []string) error {
	for index, value := range values {
		if index > 0 {
			if _, err := fmt.Fprint(writer, "\t"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(writer, value); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(writer)

	return err
}

func TrimTableCell(value string, limit int) string {
	normalized := strings.Join(strings.Fields(value), " ")
	if limit < 1 || len(normalized) <= limit {
		return normalized
	}
	if limit <= 3 {
		return normalized[:limit]
	}

	return normalized[:limit-3] + "..."
}
