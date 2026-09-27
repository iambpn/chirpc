package typescript

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"
)

type converterEmbedded struct {
	EmbeddedValue string `json:"embedded_value"`
}

type converterFixture struct {
	converterEmbedded
	Renamed   int               `json:"json_name"`
	Dashed    string            `json:"dashed-name"`
	Raw       int               `tsType:"Date | null"`
	Optional  string            `tsOptional:"true"`
	Encoded   int               `json:"encoded,string"`
	Bytes     []byte            `json:"bytes"`
	Values    map[string]string `json:"values"`
	Fixed     [2]string         `json:"fixed"`
	CreatedAt time.Time         `json:"created_at"`
	Duration  time.Duration     `json:"duration"`
	Anything  any               `json:"anything"`
	Omitted   string            `tsOmit:"true"`
}

func TestGutsConverterUsesCompilerASTAndChirpcTags(t *testing.T) {
	converter, err := newGutsConverter(nil)
	if err != nil {
		t.Fatal(err)
	}

	output, err := converter.inlineStruct(reflect.TypeOf(converterFixture{}))
	if err != nil {
		t.Fatal(err)
	}

	compact := func(value string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, value)
	}
	compactOutput := compact(output)
	checks := []string{
		"Typescript__ConverterEmbedded & {",
		"json_name: number;",
		`"dashed-name": string;`,
		"Raw: Date | null;",
		"Optional?: string;",
		"encoded: string;",
		"bytes: string;",
		"values: Record<string, string> | null;",
		"fixed: [string, string];",
		"created_at: string;",
		"duration: number;",
		"anything: unknown;",
	}
	for _, check := range checks {
		if !strings.Contains(compactOutput, compact(check)) {
			t.Fatalf("expected generated type to contain %q, got:\n%s", check, output)
		}
	}
	if strings.Contains(output, "Omitted") {
		t.Fatalf("tsOmit field should not be generated, got:\n%s", output)
	}

	declarations, err := converter.declarationStrings()
	if err != nil {
		t.Fatal(err)
	}
	if len(declarations) != 1 || !strings.Contains(declarations[0], "interface Typescript__ConverterEmbedded") {
		t.Fatalf("expected embedded declaration, got %#v", declarations)
	}
}

func TestGutsConverterResponseTypeAndNamedReuse(t *testing.T) {
	type response struct {
		Body converterEmbedded
	}

	converter, err := newGutsConverter(nil)
	if err != nil {
		t.Fatal(err)
	}

	first, err := converter.responseType(reflect.TypeOf(response{}))
	if err != nil {
		t.Fatal(err)
	}
	second, err := converter.responseType(reflect.TypeOf(response{}))
	if err != nil {
		t.Fatal(err)
	}
	if first != "Typescript__ConverterEmbedded" || second != first {
		t.Fatalf("unexpected response types: %q and %q", first, second)
	}

	declarations, err := converter.declarationStrings()
	if err != nil {
		t.Fatal(err)
	}
	if len(declarations) != 1 {
		t.Fatalf("expected one reused declaration, got %d", len(declarations))
	}
}

func TestGutsConverterRejectsResponseWithoutBody(t *testing.T) {
	converter, err := newGutsConverter(nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := converter.responseType(reflect.TypeOf(struct{ Value string }{})); err == nil {
		t.Fatal("expected response without Body to fail")
	}
}

func TestPackageNameSkipsMajorVersion(t *testing.T) {
	cases := map[string]string{
		"github.com/iambpn/chirpc/v1":    "chirpc",
		"github.com/acme/api/v12":        "api",
		"github.com/acme/models":         "models",
		"github.com/acme/version/vendor": "vendor",
		"v2":                             "v2",
		"github.com/acme/v2beta":         "v2beta",
	}
	for importPath, want := range cases {
		if got := packageName(importPath); got != want {
			t.Errorf("packageName(%q) = %q, want %q", importPath, got, want)
		}
	}
}

func TestGutsConverterUsesTypeOverrides(t *testing.T) {
	type userID struct{ value [16]byte }
	type payload struct {
		ID       userID   `json:"id"`
		Optional *userID  `json:"optional"`
		Many     []userID `json:"many"`
	}

	converter, err := newGutsConverter(map[reflect.Type]string{reflect.TypeOf(userID{}): "string"})
	if err != nil {
		t.Fatal(err)
	}

	output, err := converter.inlineStruct(reflect.TypeOf(payload{}))
	if err != nil {
		t.Fatal(err)
	}

	for _, check := range []string{"id: string;", "optional: string | null;", "many: string[];"} {
		if !strings.Contains(output, check) {
			t.Fatalf("expected generated type to contain %q, got:\n%s", check, output)
		}
	}
	if declarations, _ := converter.declarationStrings(); len(declarations) != 0 {
		t.Fatalf("expected no declaration for the overridden type, got %v", declarations)
	}
	_ = userID{}.value
}

func TestGutsConverterRejectsTsKey(t *testing.T) {
	type legacy struct {
		Name string `json:"name" tsKey:"fullName"`
	}

	converter, err := newGutsConverter(nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = converter.inlineStruct(reflect.TypeOf(legacy{}))
	if err == nil || !strings.Contains(err.Error(), "field Name uses the tsKey tag, which is no longer supported") {
		t.Fatalf("expected a tsKey error, got %v", err)
	}
}
