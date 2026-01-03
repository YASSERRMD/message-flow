package providers

import "testing"

func TestExtractJSON(t *testing.T) {
	input := "prefix {\"key\":\"value\"} suffix"
	output := extractJSON(input)
	if output != "{\"key\":\"value\"}" {
		t.Fatalf("unexpected output: %s", output)
	}

	input = "[\"a\",\"b\"]"
	output = extractJSON(input)
	if output != input {
		t.Fatalf("unexpected output for array")
	}
}
