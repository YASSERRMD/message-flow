package providers

import (
	"strings"
	"time"
)

func joinLines(messages []string) string {
	result := ""
	for i, msg := range messages {
		if i > 0 {
			result += "\n"
		}
		result += msg
	}
	return result
}

func averageLatency(current time.Duration, new time.Duration, count int64) time.Duration {
	if count <= 1 {
		return new
	}
	return time.Duration(((current * time.Duration(count-1)) + new) / time.Duration(count))
}

func extractJSON(text string) string {
	start := strings.IndexAny(text, "{[")
	end := strings.LastIndexAny(text, "}]")
	if start == -1 || end == -1 || end <= start {
		return text
	}
	return text[start : end+1]
}
