package tsopts

import "testing"

func TestSetToLowerExportedField(t *testing.T) {
	cases := []struct {
		name  string
		input bool
	}{
		{name: "true", input: true},
		{name: "false", input: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := SetToLowercaseExportedField(tc.input)
			if cfg == nil {
				t.Fatalf("expected non-nil map")
			}
			if len(cfg) != 1 {
				t.Fatalf("expected map length 1, got %d", len(cfg))
			}
			val, ok := cfg[ToLowercase]
			if !ok {
				t.Fatalf("expected key ToLowercase to be present")
			}
			if val != tc.input {
				t.Fatalf("expected value %v, got %v", tc.input, val)
			}
		})
	}
}

func TestSetAddHeaderToInterface(t *testing.T) {
	for _, tc := range []bool{true, false} {
		cfg := SetAddHeaderToInterface(tc)
		if cfg == nil {
			t.Fatalf("expected non-nil map")
		}
		if len(cfg) != 1 {
			t.Fatalf("expected map length 1, got %d", len(cfg))
		}
		val, ok := cfg[AddHeaderToInterface]
		if !ok {
			t.Fatalf("expected key AddHeaderToInterface to be present")
		}
		if val != tc {
			t.Fatalf("expected value %v, got %v", tc, val)
		}
	}
}

func TestSetUnCapitalizeHeader(t *testing.T) {
	for _, tc := range []bool{true, false} {
		cfg := SetUnCapitalizeHeader(tc)
		if cfg == nil {
			t.Fatalf("expected non-nil map")
		}
		if len(cfg) != 1 {
			t.Fatalf("expected map length 1, got %d", len(cfg))
		}
		val, ok := cfg[UnCapitalizeHeader]
		if !ok {
			t.Fatalf("expected key UnCapitalizeHeader to be present")
		}
		if val != tc {
			t.Fatalf("expected value %v, got %v", tc, val)
		}
	}
}

func TestMergeOpts(t *testing.T) {
	lower := SetToLowercaseExportedField(true)
	header := SetAddHeaderToInterface(false)
	merged := MergeOpts(lower, header)

	if merged == nil {
		t.Fatalf("expected non-nil map")
	}
	if len(merged) != 2 {
		t.Fatalf("expected map length 2, got %d", len(merged))
	}

	if merged[ToLowercase] != true {
		t.Fatalf("expected ToLowercase to be true, got %v", merged[ToLowercase])
	}
	if merged[AddHeaderToInterface] != false {
		t.Fatalf("expected AddHeaderToInterface to be false, got %v", merged[AddHeaderToInterface])
	}

	if len(lower) != 1 {
		t.Fatalf("expected original map to remain unchanged, got len %d", len(lower))
	}
	if len(header) != 1 {
		t.Fatalf("expected original map to remain unchanged, got len %d", len(header))
	}
}

func TestMergeOptsOverrides(t *testing.T) {
	first := SetToLowercaseExportedField(true)
	second := SetToLowercaseExportedField(false)

	merged := MergeOpts(first, second)
	if len(merged) != 1 {
		t.Fatalf("expected map length 1, got %d", len(merged))
	}
	if merged[ToLowercase] != false {
		t.Fatalf("expected override to set value to false, got %v", merged[ToLowercase])
	}
}

func TestMergeOptsHandlesNil(t *testing.T) {
	merged := MergeOpts(nil, SetAddHeaderToInterface(true))
	if len(merged) != 1 {
		t.Fatalf("expected map length 1, got %d", len(merged))
	}
	if merged[AddHeaderToInterface] != true {
		t.Fatalf("expected AddHeaderToInterface to be true, got %v", merged[AddHeaderToInterface])
	}
}

func TestMergeOptsNoArgs(t *testing.T) {
	merged := MergeOpts()
	if merged == nil {
		t.Fatalf("expected non-nil map")
	}
	if len(merged) != 0 {
		t.Fatalf("expected empty map, got len %d", len(merged))
	}
}
