package tsGen

import (
	"reflect"
	"strings"
	"testing"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/tsGen/tsInterface"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

type addressFixture struct {
	Street string `json:"street,omitempty"`
	Zip    int    `tsOptional:"true"`
}

type profileFixture struct {
	ID        int
	Name      string `tsKey:"fullName"`
	Age       *int
	Emails    []string
	Scores    [2]float64
	Meta      map[string]bool
	Address   addressFixture
	Residence *addressFixture
	Inline    struct {
		Flag     bool
		Alias    string `tsKey:"alias"`
		Optional int    `json:"optional,omitempty"`
	}
	Callback func()
	Anything any
	Generic  GenericAny
	hidden   string
}

func TestNewDefaultOptions(t *testing.T) {
	gen := New()

	if !gen.opt[tsopts.AddHeaderToInterface] {
		t.Fatalf("New() should enable AddHeaderToInterface by default")
	}

	if gen.builder == nil {
		t.Fatalf("New() should initialize a builder")
	}
}

func TestAddTypeRejectsNonStruct(t *testing.T) {
	gen := New()

	if err := gen.AddType(reflect.TypeOf(42)); err == nil {
		t.Fatalf("AddType should reject non-struct types")
	}
}

func TestAddValueRejectsNonStruct(t *testing.T) {
	gen := New()

	if err := gen.AddValue(42); err == nil {
		t.Fatalf("AddType should reject non-struct types")
	}
}

func TestAddTypeWithNameRejectsNonStruct(t *testing.T) {
	gen := New()

	if err := gen.AddTypeWithName(reflect.TypeOf(42), "number"); err == nil {
		t.Fatalf("AddType should reject non-struct types")
	}
}

func TestAddValueShouldAcceptPointer(t *testing.T) {
	gen := New()

	if err := gen.AddValue(&struct{}{}); err != nil {
		t.Fatalf("AddValue should accept pointer values")
	}
}

func TestAddValueWithNameUsesExplicitHeader(t *testing.T) {
	gen := New(tsopts.SetAddHeaderToInterface(false))

	type sample struct {
		Field int
	}

	if err := gen.AddValueWithName(sample{}, "CustomHeader"); err != nil {
		t.Fatalf("AddValueWithName failed: %v", err)
	}

	if _, ok := gen.GetRegisteredTypes().Get("CustomHeader"); !ok {
		t.Fatalf("expected interface with explicit header to be registered")
	}
}

func TestRegisterStructWithTags(t *testing.T) {
	gen := New(tsopts.SetToLowerExportedField(true))

	if err := gen.AddValue(profileFixture{}); err != nil {
		t.Fatalf("AddValue returned error: %v", err)
	}

	types := gen.GetRegisteredTypes()

	addressInf, ok := types.Get("TsGen__AddressFixture")
	if !ok {
		t.Fatalf("address interface not registered")
	}

	assertProperty(t, addressInf, "street", "string", true)
	assertProperty(t, addressInf, "zip", "number", true)

	profileInf, ok := types.Get("TsGen__ProfileFixture")
	if !ok {
		t.Fatalf("profile interface not registered")
	}

	assertProperty(t, profileInf, "id", "number", false)
	assertProperty(t, profileInf, "fullName", "string", false)
	assertProperty(t, profileInf, "age", "number | null", false)
	assertProperty(t, profileInf, "emails", "string[]", false)
	assertProperty(t, profileInf, "scores", "number[]", false)
	assertProperty(t, profileInf, "meta", "{ [key: string]: boolean }", false)
	assertProperty(t, profileInf, "address", "TsGen__AddressFixture", false)
	assertProperty(t, profileInf, "residence", "TsGen__AddressFixture | null", false)
	assertProperty(t, profileInf, "callback", "Function", false)
	assertProperty(t, profileInf, "anything", "any", false)
	assertProperty(t, profileInf, "generic", "any", false)

	inlineValue, err := profileInf.GetProperty("inline")
	if err != nil {
		t.Fatalf("inline property missing: %v", err)
	}

	if !strings.Contains(inlineValue.Value, "Flag:boolean") {
		t.Fatalf("inline struct should include Flag field, got %q", inlineValue.Value)
	}
	if !strings.Contains(inlineValue.Value, "alias:string") {
		t.Fatalf("inline struct should include alias field, got %q", inlineValue.Value)
	}
	if !strings.Contains(inlineValue.Value, "optional?:number") {
		t.Fatalf("inline struct should mark optional field, got %q", inlineValue.Value)
	}

	if _, err := profileInf.GetProperty("hidden"); err == nil {
		t.Fatalf("unexported fields should not be surfaced")
	}
}

func TestAddValueIsIdempotent(t *testing.T) {
	gen := New()

	type sample struct {
		Field int
	}

	if err := gen.AddValue(sample{}); err != nil {
		t.Fatalf("first AddValue returned error: %v", err)
	}

	if err := gen.AddValue(sample{}); err != nil {
		t.Fatalf("second AddValue returned error: %v", err)
	}

	if count := countInterfaces(gen.GetRegisteredTypes()); count != 1 {
		t.Fatalf("expected single registered interface, got %d", count)
	}
}

type nestedKind struct {
	Value string
}

type kindFixture struct {
	Bool       bool
	Int        int
	Uint       uint16
	Float      float32
	String     string
	Slice      []int
	Array      [3]uint8
	Map        map[string]nestedKind
	Nested     nestedKind
	Pointer    *nestedKind
	PtrBasic   *int
	Func       func()
	Interface  interface{}
	Chan       chan int
	Any        any
	Generic    GenericAny
	InlineAnon struct {
		Count int
	}
}

func TestGetTypeCoversKinds(t *testing.T) {
	gen := New()

	typ := reflect.TypeOf(kindFixture{})

	cases := map[string]string{
		"Bool":       "boolean",
		"Int":        "number",
		"Uint":       "number",
		"Float":      "number",
		"String":     "string",
		"Slice":      "number[]",
		"Array":      "number[]",
		"Map":        "{ [key: string]: TsGen__NestedKind }",
		"Nested":     "TsGen__NestedKind",
		"Pointer":    "TsGen__NestedKind | null",
		"PtrBasic":   "number | null",
		"Func":       "Function",
		"Interface":  "any",
		"Chan":       "any",
		"Any":        "any",
		"Generic":    "any",
		"InlineAnon": "{ Count:number; }",
	}

	for fieldName, want := range cases {
		field, ok := typ.FieldByName(fieldName)
		if !ok {
			t.Fatalf("field %s not found in fixture", fieldName)
		}

		if got := gen.GetType(field); got != want {
			t.Fatalf("GetType(%s) = %q, want %q", fieldName, got, want)
		}
	}

	if _, exists := gen.builder.QueryType("TsGen__NestedKind"); !exists {
		t.Fatalf("expected nested struct type to be registered")
	}
}

func TestBuilderStringFormatting(t *testing.T) {
	gen := New()

	type sample struct {
		Field int
	}

	if err := gen.AddValue(sample{}); err != nil {
		t.Fatalf("AddValue returned error: %v", err)
	}

	output := gen.String()

	if !strings.HasPrefix(output, "\n") {
		t.Fatalf("builder output should start with newline, got %q", output)
	}

	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("builder output should end with newline, got %q", output)
	}

	if strings.Count(output, "interface") != 1 {
		t.Fatalf("expected single interface in output, got %q", output)
	}
}

func countInterfaces(m *orderedmap.OrderedMap[string, *tsInterface.TsInterface]) int {
	count := 0
	for el := m.Front(); el != nil; el = el.Next() {
		count++
	}
	return count
}

func assertProperty(t *testing.T, inf *tsInterface.TsInterface, key, wantValue string, wantOptional bool) {
	t.Helper()

	prop, err := inf.GetProperty(key)
	if err != nil {
		t.Fatalf("property %s missing: %v", key, err)
	}

	if prop.Value != wantValue {
		t.Fatalf("property %s value = %q, want %q", key, prop.Value, wantValue)
	}

	if prop.IsOptional != wantOptional {
		t.Fatalf("property %s optional = %v, want %v", key, prop.IsOptional, wantOptional)
	}
}
