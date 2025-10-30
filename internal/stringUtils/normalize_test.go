package stringUtils

import "testing"

func TestNormalizeStructFieldName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no changes",
			input:    "FieldName",
			expected: "FieldName",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "square brackets",
			input:    "data[0]",
			expected: "data_0_",
		},
		{
			name:     "dot separators",
			input:    "user.profile.name",
			expected: "user_profile_name",
		},
		{
			name:     "nested path with brackets",
			input:    "payload.items[0].value",
			expected: "payload_items_0__value",
		},
		{
			name:     "consecutive special characters",
			input:    "my..field[][value]",
			expected: "my__field___value_",
		},
		{
			name:     "already normalized",
			input:    "field_name",
			expected: "field_name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := NormalizeStructFieldName(tc.input)
			if actual != tc.expected {
				t.Fatalf("NormalizeStructFieldName(%q) = %q, expected %q", tc.input, actual, tc.expected)
			}
		})
	}
}
