package tsInterface

import "testing"

func TestTsInterfaceAddAndGetProperty(t *testing.T) {
	ti := New(false)
	ti.AddInterfaceName("Sample")
	ti.AddProperty("id", "number", false)
	ti.AddProperty("title", "string", true)

	val, err := ti.GetProperty("id")
	if err != nil {
		t.Fatalf("GetProperty returned error for existing key: %v", err)
	}
	if val.Value != "number" {
		t.Fatalf("expected value 'number', got %q", val.Value)
	}
	if val.IsOptional {
		t.Fatalf("expected property 'id' to be required")
	}

	val, err = ti.GetProperty("title")
	if err != nil {
		t.Fatalf("GetProperty returned error for optional key: %v", err)
	}
	if val.Value != "string" {
		t.Fatalf("expected value 'string', got %q", val.Value)
	}
	if !val.IsOptional {
		t.Fatalf("expected property 'title' to be optional")
	}

	ti.AddProperty("title", "string | null", false)
	val, err = ti.GetProperty("title")
	if err != nil {
		t.Fatalf("GetProperty returned error after overwrite: %v", err)
	}
	if val.Value != "string | null" {
		t.Fatalf("expected overwritten value 'string | null', got %q", val.Value)
	}
	if val.IsOptional {
		t.Fatalf("expected overwritten property 'title' to be required")
	}
}

func TestTsInterfaceGetPropertyError(t *testing.T) {
	ti := New(false)

	if _, err := ti.GetProperty("missing"); err == nil {
		t.Fatalf("expected error for missing key, got nil")
	} else if err.Error() != "key missing not found in body" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestTsInterfaceRemoveProperty(t *testing.T) {
	ti := New(false)
	ti.AddProperty("id", "number", false)

	ti.RemoveProperty("id")

	if _, err := ti.GetProperty("id"); err == nil {
		t.Fatalf("expected error for removed key, got nil")
	}
}

func TestTsInterfacePrimaryFlag(t *testing.T) {
	ti := New(false)

	if ti.IsPrimary() {
		t.Fatalf("expected new interface to be non-primary by default")
	}

	ti.SetPrimary(true)
	if !ti.IsPrimary() {
		t.Fatalf("expected interface to be primary after SetPrimary(true)")
	}

	ti.SetPrimary(false)
	if ti.IsPrimary() {
		t.Fatalf("expected interface to be non-primary after SetPrimary(false)")
	}
}

func TestTsInterfaceStringFormatting(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *TsInterface
		expected string
	}{
		{
			name: "exported named interface",
			builder: func() *TsInterface {
				ti := New(true)
				ti.AddInterfaceName("User")
				ti.AddProperty("id", "number", false)
				ti.AddProperty("email", "string | null", true)
				return ti
			},
			expected: "export interface User { id:number; email?:string | null; }",
		},
		{
			name: "non-exported empty interface",
			builder: func() *TsInterface {
				ti := New(false)
				ti.AddInterfaceName("Empty")
				return ti
			},
			expected: "interface Empty { }",
		},
		{
			name: "anonymous interface ignores export",
			builder: func() *TsInterface {
				ti := New(true)
				ti.AddProperty("value", "boolean", false)
				return ti
			},
			expected: "{ value:boolean; }",
		},
		{
			name: "insertion order after delete",
			builder: func() *TsInterface {
				ti := New(false)
				ti.AddInterfaceName("Item")
				ti.AddProperty("alpha", "string", false)
				ti.AddProperty("beta", "boolean", false)
				ti.RemoveProperty("alpha")
				ti.AddProperty("alpha", "number", false)
				return ti
			},
			expected: "interface Item { beta:boolean; alpha:number; }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.builder().String(); got != tc.expected {
				t.Fatalf("unexpected string output\nexpected: %q\nactual:   %q", tc.expected, got)
			}
		})
	}
}
