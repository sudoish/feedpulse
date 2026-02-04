package parser

import (
	"strings"
	"testing"
)

// Test parsing extremely nested JSON
func TestParse_DeeplyNestedJSON(t *testing.T) {
	// Create deeply nested structure (10 levels)
	nested := `{"items": [{"nested1": {"nested2": {"nested3": {"nested4": {"nested5": {"nested6": {"nested7": {"nested8": {"nested9": {"full_name": "deep/repo", "html_url": "https://github.com/deep/repo"}}}}}}}}}}]}`

	parser := NewParser()
	result := parser.Parse("GitHub", "json", []byte(nested))

	// The current parser won't find deeply nested items, but should not crash
	// This test verifies the parser doesn't panic on complex structures
	if len(result.Items) == 0 && len(result.Errors) == 0 {
		t.Error("Expected either items or error for nested structure")
	}
}

// Test very large JSON responses
func TestParse_LargeJSON(t *testing.T) {
	// Create large array (1000 items)
	var items []string
	for i := 0; i < 1000; i++ {
		items = append(items, `{"full_name":"repo`+strings.Repeat("x", i%10)+`","html_url":"https://github.com/test/repo`+strings.Repeat("x", i%10)+`"}`)
	}
	largeJSON := `{"items":[` + strings.Join(items, ",") + `]}`

	parser := NewParser()
	result := parser.Parse("GitHub", "json", []byte(largeJSON))

	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors for large JSON, got: %v", result.Errors)
	}
	if len(result.Items) != 1000 {
		t.Errorf("Expected 1000 items, got %d", len(result.Items))
	}
}

// Test empty arrays and null values
func TestParse_EmptyArrays(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		source string
	}{
		{"empty GitHub items", `{"items":[]}`, "GitHub"},
		{"empty Reddit children", `{"data":{"children":[]}}`, "Reddit"},
		{"empty Lobsters array", `[]`, "Lobsters"},
		{"empty HackerNews array", `[]`, "HackerNews"},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.source, "json", []byte(tt.json))
			if len(result.Items) != 0 {
				t.Errorf("Expected 0 items for empty array, got %d", len(result.Items))
			}
			if len(result.Errors) == 0 {
				t.Error("Expected error for empty response")
			}
		})
	}
}

// Test null values in various fields
func TestParse_NullValues(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		source string
	}{
		{
			"GitHub null fields",
			`{"items":[{"full_name":null,"html_url":"https://github.com/test/repo"}]}`,
			"GitHub",
		},
		{
			"Reddit null title",
			`{"data":{"children":[{"data":{"title":null,"url":"https://reddit.com/r/test/123"}}]}}`,
			"Reddit",
		},
		{
			"Lobsters null url",
			`[{"title":"Test","url":null,"comments_url":"https://lobste.rs/s/abc"}]`,
			"Lobsters",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.source, "json", []byte(tt.json))
			// Parser should handle nulls gracefully (either skip or use defaults)
			if len(result.Items) > 0 {
				// If items were parsed, check they have non-empty IDs
				for _, item := range result.Items {
					if item.ID == "" {
						t.Error("Parsed item should have non-empty ID")
					}
				}
			}
		})
	}
}

// Test special characters in content
func TestParse_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		source string
	}{
		{
			"HTML entities",
			`{"items":[{"full_name":"test&amp;repo","html_url":"https://github.com/test/repo&lt;script&gt;"}]}`,
			"GitHub",
		},
		{
			"quotes in title",
			`{"data":{"children":[{"data":{"title":"Test \"quoted\" title","url":"https://reddit.com/r/test/123"}}]}}`,
			"Reddit",
		},
		{
			"unicode emoji",
			`[{"title":"Test 🚀 Story","url":"https://example.com","comments_url":"https://lobste.rs/s/abc"}]`,
			"Lobsters",
		},
		{
			"RTL text",
			`[{"title":"العربية","url":"https://example.com","comments_url":"https://lobste.rs/s/abc"}]`,
			"Lobsters",
		},
		{
			"CJK characters",
			`{"items":[{"full_name":"测试/项目","html_url":"https://github.com/test/repo"}]}`,
			"GitHub",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.source, "json", []byte(tt.json))
			if len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got: %v", result.Errors)
			}
			if len(result.Items) == 0 {
				t.Error("Expected at least one item")
			}
		})
	}
}

// Test different timestamp formats
func TestParse_TimestampFormats(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		source string
	}{
		{
			"ISO 8601",
			`{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo","updated_at":"2024-01-01T12:00:00Z"}]}`,
			"GitHub",
		},
		{
			"Unix timestamp",
			`{"data":{"children":[{"data":{"title":"Test","url":"https://reddit.com/r/test/123","created_utc":1704110400}}]}}`,
			"Reddit",
		},
		{
			"RFC 3339",
			`[{"title":"Test","url":"https://example.com","created_at":"2024-01-01T12:00:00+00:00"}]`,
			"Lobsters",
		},
		{
			"Custom format",
			`[{"title":"Test","url":"https://example.com","created_at":"2024-01-01 12:00:00"}]`,
			"Lobsters",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.source, "json", []byte(tt.json))
			if len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got: %v", result.Errors)
			}
			if len(result.Items) == 0 {
				t.Error("Expected at least one item")
			}
		})
	}
}

// Test partial data scenarios
func TestParse_PartialData(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		source string
	}{
		{
			"GitHub missing topics",
			`{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`,
			"GitHub",
		},
		{
			"Reddit missing flair",
			`{"data":{"children":[{"data":{"title":"Test","url":"https://reddit.com/r/test/123"}}]}}`,
			"Reddit",
		},
		{
			"Lobsters missing tags",
			`[{"title":"Test","url":"https://example.com"}]`,
			"Lobsters",
		},
		{
			"Mixed complete and partial",
			`{"items":[{"full_name":"test1/repo","html_url":"https://github.com/test1/repo"},{"full_name":"test2/repo","html_url":"https://github.com/test2/repo","topics":["go"]}]}`,
			"GitHub",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.source, "json", []byte(tt.json))
			if len(result.Errors) > 0 {
				t.Errorf("Expected no errors for partial data, got: %v", result.Errors)
			}
			if len(result.Items) == 0 {
				t.Error("Expected at least one item from partial data")
			}
		})
	}
}

// Test Content-Type handling edge cases
func TestParse_ContentTypeEdgeCases(t *testing.T) {
	// Note: Content-Type is handled by fetcher, but parser should be robust
	json := `{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`

	parser := NewParser()
	result := parser.Parse("GitHub", "json", []byte(json))

	if len(result.Errors) > 0 {
		t.Errorf("Parser should handle JSON regardless of Content-Type header, got errors: %v", result.Errors)
	}
}

// Test malformed JSON edge cases
func TestParse_MalformedJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"truncated", `{"items":[{"full_name":"test`},
		{"missing quotes", `{items:[{full_name:test}]}`},
		{"trailing comma", `{"items":[{"full_name":"test/repo",}]}`},
		{"single quote", `{'items':[]}`},
		{"unescaped quote", `{"title":"Test "quote" here"}`},
		{"invalid escape", `{"title":"Test \x invalid"}`},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse("Test", "json", []byte(tt.json))
			if len(result.Errors) == 0 {
				t.Error("Expected error for malformed JSON")
			}
			if !strings.Contains(result.Errors[0], "malformed JSON") {
				t.Errorf("Expected 'malformed JSON' error, got: %s", result.Errors[0])
			}
		})
	}
}

// Test very long strings in fields
func TestParse_VeryLongStrings(t *testing.T) {
	longTitle := strings.Repeat("a", 10000)
	json := `{"items":[{"full_name":"` + longTitle + `","html_url":"https://github.com/test/repo"}]}`

	parser := NewParser()
	result := parser.Parse("GitHub", "json", []byte(json))

	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors for long strings, got: %v", result.Errors)
	}
	if len(result.Items) > 0 && len(result.Items[0].Title) != 10000 {
		t.Errorf("Expected long title to be preserved, got length: %d", len(result.Items[0].Title))
	}
}

// Test mixed valid and invalid items
func TestParse_MixedValidInvalid(t *testing.T) {
	json := `{"items":[
		{"full_name":"valid/repo","html_url":"https://github.com/valid/repo"},
		{"full_name":"missing-url"},
		{"html_url":"https://github.com/missing/name"},
		{"full_name":"another-valid/repo","html_url":"https://github.com/another-valid/repo"}
	]}`

	parser := NewParser()
	result := parser.Parse("GitHub", "json", []byte(json))

	if len(result.Items) != 2 {
		t.Errorf("Expected 2 valid items, got %d", len(result.Items))
	}
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors for invalid items, got %d", len(result.Errors))
	}
}

// Test boolean and numeric coercion
func TestParse_TypeCoercionEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			"number as string",
			`{"items":[{"full_name":12345,"html_url":"https://github.com/test/repo"}]}`,
		},
		{
			"boolean as string",
			`{"items":[{"full_name":true,"html_url":"https://github.com/test/repo"}]}`,
		},
		{
			"array as string",
			`{"items":[{"full_name":["test","repo"],"html_url":"https://github.com/test/repo"}]}`,
		},
		{
			"object as string",
			`{"items":[{"full_name":{"name":"test"},"html_url":"https://github.com/test/repo"}]}`,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse("GitHub", "json", []byte(tt.json))
			// Parser should either coerce or report error gracefully
			if len(result.Items) == 0 && len(result.Errors) == 0 {
				t.Error("Expected either items or errors, got neither")
			}
		})
	}
}

// Test whitespace handling
func TestParse_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			"extra whitespace",
			`   {   "items"  :  [  {  "full_name"  :  "test/repo"  ,  "html_url"  :  "https://github.com/test/repo"  }  ]  }   `,
		},
		{
			"tabs and newlines",
			"{\n\t\"items\": [\n\t\t{\n\t\t\t\"full_name\": \"test/repo\",\n\t\t\t\"html_url\": \"https://github.com/test/repo\"\n\t\t}\n\t]\n}",
		},
		{
			"no whitespace",
			`{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse("GitHub", "json", []byte(tt.json))
			if len(result.Errors) > 0 {
				t.Errorf("Expected no errors for valid JSON with whitespace, got: %v", result.Errors)
			}
			if len(result.Items) != 1 {
				t.Errorf("Expected 1 item, got %d", len(result.Items))
			}
		})
	}
}

// Test array vs object ambiguity
func TestParse_StructureDetection(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantType string
	}{
		{"numeric array (HackerNews)", `[1, 2, 3]`, "HackerNews"},
		{"object array (Lobsters)", `[{"title":"Test","url":"https://example.com"}]`, "Lobsters"},
		{"GitHub structure", `{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`, "GitHub"},
		{"Reddit structure", `{"data":{"children":[{"data":{"title":"Test","url":"https://reddit.com/r/test/123"}}]}}`, "Reddit"},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.wantType, "json", []byte(tt.json))
			// Just verify no crash and some reasonable output
			if len(result.Items) == 0 && len(result.Errors) == 0 {
				t.Error("Expected either items or errors")
			}
		})
	}
}

// Test empty string values
func TestParse_EmptyStrings(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		source string
	}{
		{
			"empty title",
			`{"items":[{"full_name":"","html_url":"https://github.com/test/repo"}]}`,
			"GitHub",
		},
		{
			"empty URL",
			`{"items":[{"full_name":"test/repo","html_url":""}]}`,
			"GitHub",
		},
		{
			"both empty",
			`{"items":[{"full_name":"","html_url":""}]}`,
			"GitHub",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.source, "json", []byte(tt.json))
			// Note: Current parser doesn't validate empty strings, just checks field presence
			// This tests that empty strings don't cause crashes
			if len(result.Items) == 0 && len(result.Errors) == 0 {
				t.Error("Expected either items or errors")
			}
		})
	}
}

// Test numeric ID edge cases
func TestParse_NumericIDEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"zero", `[0]`},
		{"negative", `[-1]`},
		{"very large", `[9999999999]`},
		{"float", `[1.5, 2.7, 3.9]`},
		{"mixed types", `[1, "not a number", 3]`},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse("HackerNews", "json", []byte(tt.json))
			// Parser should handle or report errors gracefully
			if len(result.Items) == 0 && len(result.Errors) == 0 {
				t.Error("Expected either items or errors")
			}
		})
	}
}

// Test duplicate items
func TestParse_DuplicateItems(t *testing.T) {
	json := `{"items":[
		{"full_name":"test/repo","html_url":"https://github.com/test/repo"},
		{"full_name":"test/repo","html_url":"https://github.com/test/repo"},
		{"full_name":"different/repo","html_url":"https://github.com/different/repo"}
	]}`

	parser := NewParser()
	result := parser.Parse("GitHub", "json", []byte(json))

	if len(result.Items) != 3 {
		t.Errorf("Expected 3 items (parser doesn't dedupe), got %d", len(result.Items))
	}

	// Check that duplicate items have the same ID
	if len(result.Items) >= 2 {
		if result.Items[0].ID != result.Items[1].ID {
			t.Error("Expected duplicate items to have the same ID")
		}
		if result.Items[0].ID == result.Items[2].ID {
			t.Error("Expected different items to have different IDs")
		}
	}
}
