package stringUtils

import "testing"

func TestShouldToLower(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		shouldConvert bool
		expected      string
	}{
		{
			name:          "convert mixed case",
			input:         "MiXeDCaSe",
			shouldConvert: true,
			expected:      "mixedcase",
		},
		{
			name:          "already lowercase",
			input:         "alreadylower",
			shouldConvert: true,
			expected:      "alreadylower",
		},
		{
			name:          "uppercase with punctuation",
			input:         "DATA-SET",
			shouldConvert: true,
			expected:      "data-set",
		},
		{
			name:          "skip conversion",
			input:         "SHOULD STAY",
			shouldConvert: false,
			expected:      "SHOULD STAY",
		},
		{
			name:          "empty string",
			input:         "",
			shouldConvert: true,
			expected:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ShouldToLower(tc.input, tc.shouldConvert)
			if actual != tc.expected {
				t.Fatalf("ShouldToLower(%q, %t) = %q, expected %q", tc.input, tc.shouldConvert, actual, tc.expected)
			}
		})
	}
}

func TestCapitalize(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase word",
			input:    "alpha",
			expected: "Alpha",
		},
		{
			name:     "already capitalized",
			input:    "Alpha",
			expected: "Alpha",
		},
		{
			name:     "single letter",
			input:    "a",
			expected: "A",
		},
		{
			name:     "leading non letter",
			input:    "1value",
			expected: "1value",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Capitalize(tc.input)
			if actual != tc.expected {
				t.Fatalf("Capitalize(%q) = %q, expected %q", tc.input, actual, tc.expected)
			}
		})
	}
}

func TestUncapitalize(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "upper case word",
			input:    "HELLO",
			expected: "hELLO",
		},
		{
			name:     "already lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "single letter",
			input:    "H",
			expected: "h",
		},
		{
			name:     "leading non letter",
			input:    "_Value",
			expected: "_Value",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Uncapitalize(tc.input)
			if actual != tc.expected {
				t.Fatalf("Uncapitalize(%q) = %q, expected %q", tc.input, actual, tc.expected)
			}
		})
	}
}
