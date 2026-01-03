package llm

import "strings"

func fallbackAnalysis(message string) *AnalysisResult {
	text := strings.ToLower(message)
	urgentWords := []string{"urgent", "asap", "deadline", "important"}
	sentimentPos := []string{"happy", "excited", "great", "thanks"}
	sentimentNeg := []string{"angry", "upset", "frustrated", "issue"}

	isImportant := containsAny(text, urgentWords)
	priority := "low"
	if isImportant {
		priority = "high"
	}

	sentiment := "neutral"
	if containsAny(text, sentimentNeg) {
		sentiment = "negative"
	} else if containsAny(text, sentimentPos) {
		sentiment = "positive"
	}

	exclamations := strings.Count(message, "!")
	questions := strings.Count(message, "?")
	if exclamations >= 2 && priority != "high" {
		isImportant = true
		priority = "medium"
	}
	hasAction := isImportant || questions > 0

	return &AnalysisResult{
		IsImportant:    isImportant,
		Priority:       priority,
		Reason:         "keyword fallback",
		HasAction:      hasAction,
		ActionRequired: "review",
		Sentiment:      sentiment,
		SentimentScore: 0,
		Topics:         []string{},
		Confidence:     0.3,
	}
}

func containsAny(text string, words []string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}
