package llm

import "testing"

func TestFallbackAnalysis(t *testing.T) {
	result := fallbackAnalysis("Urgent: customer is angry!!")
	if !result.IsImportant {
		t.Fatalf("expected important")
	}
	if result.Priority != "high" {
		t.Fatalf("expected high priority")
	}
	if result.Sentiment != "negative" {
		t.Fatalf("expected negative sentiment")
	}
	if result.Confidence != 0.3 {
		t.Fatalf("expected low confidence")
	}
}

func TestUsageCost(t *testing.T) {
	record := UsageRecord{InputTokens: 500, OutputTokens: 1000}
	cost := record.TotalCost(0.01, 0.02)
	if cost <= 0 {
		t.Fatalf("expected positive cost")
	}
}
