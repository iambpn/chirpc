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
	Renamed   int               `json:"json_name" tsKey:"tsName"`
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
	converter, err := newGutsConverter()
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
		"tsName: number;",
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

	converter, err := newGutsConverter()
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
	converter, err := newGutsConverter()
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

func TestHasRequiredField(t *testing.T) {
	type optionalEmbedded struct {
		Page int `json:"page,omitempty"`
	}
	type allOptional struct {
		optionalEmbedded
		Filter  string `tsOptional:"true"`
		Skipped string `json:"-"`
		hidden  string
	}
	type withRequired struct {
		optionalEmbedded
		Filter string `json:"filter"`
	}

	if hasRequiredField(reflect.TypeOf(allOptional{}), map[reflect.Type]bool{}) {
		t.Error("expected allOptional to have no required field")
	}
	if !hasRequiredField(reflect.TypeOf(&withRequired{}), map[reflect.Type]bool{}) {
		t.Error("expected withRequired to have a required field")
	}
	_ = allOptional{}.hidden
}
