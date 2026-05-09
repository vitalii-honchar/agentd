package result

import "strings"

const DefaultSummaryLimit = 160

func Summarize(text string, limit int) string {
	normalized := strings.Join(strings.Fields(text), " ")
	if limit < 1 || len(normalized) <= limit {
		return normalized
	}
	if limit <= 3 {
		return normalized[:limit]
	}

	return normalized[:limit-3] + "..."
}
