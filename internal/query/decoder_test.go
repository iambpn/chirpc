package query

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

type pagination struct {
	Page int `json:"page,omitempty"`
}

type searchQuery struct {
	pagination
	Term    string    `json:"term"`
	Limit   *int      `json:"limit,omitempty"`
	Tags    []string  `json:"tag" tsOptional:"true"`
	Active  bool      `json:"active" tsOptional:"true"`
	Score   float64   `json:"score,omitempty"`
	Since   time.Time `json:"since,omitempty"`
	Count   uint8     `json:"count,omitempty"`
	Skipped string    `json:"-"`
	hidden  string
}

func decode(t *testing.T, raw string) (searchQuery, map[string][]string) {
	t.Helper()
	decoder, err := NewDecoder(reflect.TypeFor[searchQuery]())
	if err != nil {
		t.Fatalf("NewDecoder returned error: %v", err)
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatal(err)
	}
	var result searchQuery
	problems := decoder.Decode(values, reflect.ValueOf(&result).Elem())
	return result, problems
}

func TestDecodeFillsSupportedFields(t *testing.T) {
	result, problems := decode(t, "term=go&page=2&limit=10&tag=a&tag=b&active=true&score=1.5&since=2024-01-02T03:04:05Z&count=7&Skipped=x")
	if problems != nil {
		t.Fatalf("expected no problems, got %v", problems)
	}

	since, _ := time.Parse(time.RFC3339, "2024-01-02T03:04:05Z")
	if result.Term != "go" || result.Page != 2 || result.Limit == nil || *result.Limit != 10 ||
		!reflect.DeepEqual(result.Tags, []string{"a", "b"}) || !result.Active || result.Score != 1.5 ||
		!result.Since.Equal(since) || result.Count != 7 || result.Skipped != "" {
		t.Fatalf("unexpected result %+v", result)
	}
	_ = result.hidden
}

func TestDecodeReportsMissingAndInvalidValues(t *testing.T) {
	_, problems := decode(t, "page=two&count=300&active=maybe")

	want := map[string][]string{
		"term":   {"This field is required."},
		"page":   {`The value "two" is not a valid whole number.`},
		"count":  {`The value "300" is not a valid non-negative whole number.`},
		"active": {`The value "maybe" is not a valid boolean.`},
	}
	if !reflect.DeepEqual(problems, want) {
		t.Fatalf("expected problems %v, got %v", want, problems)
	}
}

func TestNewDecoderRejectsUnsupportedTypes(t *testing.T) {
	type nested struct{ Name string }
	cases := map[string]any{
		"nested struct":    struct{ Inner nested }{},
		"map":              struct{ Values map[string]string }{},
		"embedded pointer": struct{ *pagination }{},
		"duplicate key": struct {
			pagination
			Page int `json:"page"`
		}{},
		"tsKey tag": struct {
			Name string `tsKey:"name"`
		}{},
	}
	for name, value := range cases {
		if _, err := NewDecoder(reflect.TypeOf(value)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	if _, err := NewDecoder(reflect.TypeFor[string]()); err == nil || !strings.Contains(err.Error(), "must be a struct") {
		t.Errorf("expected a struct error, got %v", err)
	}
}
