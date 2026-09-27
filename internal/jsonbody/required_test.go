package jsonbody

import (
	"reflect"
	"testing"
	"time"
)

type address struct {
	City string `json:"city"`
	Zip  string `json:"zip,omitempty"`
}

type audit struct {
	CreatedBy string `json:"createdBy"`
}

type item struct {
	Name string `json:"name"`
}

type node struct {
	Value    string  `json:"value"`
	Children []*node `json:"children,omitempty"`
}

type order struct {
	audit
	Customer string            `json:"customer"`
	Note     string            `tsOptional:"true"`
	Address  address           `json:"address"`
	Billing  *address          `json:"billing"`
	Items    []item            `json:"items"`
	Extras   map[string]item   `json:"extras,omitempty"`
	When     time.Time         `json:"when,omitempty"`
	Tree     *node             `json:"tree,omitempty"`
	Skipped  string            `json:"-"`
	Hidden   string            `tsOmit:"true"`
	Plain    string            // no json tag, so the key is "Plain"
	Tags     map[string]string `json:"tags,omitempty"`
	Raw      []byte            `json:"raw,omitempty"`
	unused   string
}

func check(t *testing.T, body string) map[string][]string {
	t.Helper()
	return NewChecker(reflect.TypeOf(order{})).Check([]byte(body))
}

func TestCheckAcceptsCompleteBody(t *testing.T) {
	body := `{
		"createdBy": "ops",
		"customer": "Ada",
		"address": {"city": "Paris"},
		"billing": null,
		"items": [{"name": "pen"}],
		"extras": {"gift": {"name": "card"}},
		"when": "2024-01-02T03:04:05Z",
		"tree": {"value": "root", "children": [{"value": "leaf"}]},
		"PLAIN": "matched without case"
	}`
	if problems := check(t, body); problems != nil {
		t.Fatalf("expected no problems, got %v", problems)
	}
	_ = order{}.unused
}

func TestCheckReportsMissingFieldsAtEveryLevel(t *testing.T) {
	body := `{
		"address": {},
		"items": [{"name": "pen"}, {}],
		"extras": {"gift": {}},
		"tree": {"children": [{"value": "leaf"}, {}]}
	}`
	want := map[string][]string{
		"createdBy":              {"This field is required."},
		"customer":               {"This field is required."},
		"billing":                {"This field is required."},
		"Plain":                  {"This field is required."},
		"address.city":           {"This field is required."},
		"items[1].name":          {"This field is required."},
		"extras.gift.name":       {"This field is required."},
		"tree.value":             {"This field is required."},
		"tree.children[1].value": {"This field is required."},
	}
	if problems := check(t, body); !reflect.DeepEqual(problems, want) {
		t.Fatalf("expected problems %v, got %v", want, problems)
	}
}

func TestCheckWithNothingToCheck(t *testing.T) {
	type allOptional struct {
		Name string `json:"name,omitempty"`
	}
	if problems := NewChecker(reflect.TypeOf(allOptional{})).Check([]byte(`{}`)); problems != nil {
		t.Fatalf("expected no problems, got %v", problems)
	}
	if checker := NewChecker(reflect.TypeOf(time.Time{})); checker.root != nil {
		t.Fatal("expected no checks for a type with its own JSON decoding")
	}
}
