package tags

import (
	"reflect"
	"testing"
)

func TestFieldNameOptionalAndOmitted(t *testing.T) {
	type sample struct {
		Renamed  string `json:"json_name"`
		JSONName string `json:"json_only,omitempty"`
		Plain    string
		Optional string `tsOptional:"true"`
		Skipped  string `json:"-"`
		Hidden   string `tsOmit:"true"`
	}
	typ := reflect.TypeFor[sample]()
	field := func(name string) reflect.StructField {
		f, _ := typ.FieldByName(name)
		return f
	}

	names := map[string]string{"Renamed": "json_name", "JSONName": "json_only", "Plain": "Plain"}
	for goName, want := range names {
		if got := FieldName(field(goName)); got != want {
			t.Errorf("FieldName(%s) = %q, want %q", goName, got, want)
		}
	}
	if !IsOptional(field("JSONName")) || !IsOptional(field("Optional")) || IsOptional(field("Plain")) {
		t.Error("unexpected IsOptional result")
	}
	if !IsOmitted(field("Skipped")) || !IsOmitted(field("Hidden")) || IsOmitted(field("Plain")) {
		t.Error("unexpected IsOmitted result")
	}
}

func TestIsEmbeddedAndIgnoresCase(t *testing.T) {
	type inner struct {
		Value string `json:"value"`
	}
	type sample struct {
		inner
		Explicit inner `json:",embed"`
		Named    inner `json:"named"`
		UserID   int   `json:"user_id,case:ignore"`
	}
	typ := reflect.TypeFor[sample]()

	for i, want := range []bool{true, true, false, false} {
		if got := IsEmbedded(typ.Field(i)); got != want {
			t.Errorf("IsEmbedded(%s) = %v, want %v", typ.Field(i).Name, got, want)
		}
	}
	if !IgnoresCase(typ.Field(3)) || IgnoresCase(typ.Field(2)) {
		t.Error("unexpected IgnoresCase result")
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
	type recursive struct {
		*recursive
	}

	if HasRequiredField(reflect.TypeFor[allOptional]()) {
		t.Error("expected allOptional to have no required field")
	}
	if !HasRequiredField(reflect.TypeFor[*withRequired]()) {
		t.Error("expected withRequired to have a required field")
	}
	if HasRequiredField(reflect.TypeFor[recursive]()) {
		t.Error("expected recursive embedding to stop without a required field")
	}
	_ = allOptional{}.hidden
}

func TestCheckRemovedTags(t *testing.T) {
	type sample struct {
		Legacy string `tsKey:"old"`
		Plain  string `json:"plain"`
	}
	typ := reflect.TypeFor[sample]()

	if err := CheckRemovedTags(typ.Field(0)); err == nil {
		t.Error("expected an error for tsKey")
	}
	if err := CheckRemovedTags(typ.Field(1)); err != nil {
		t.Errorf("expected no error for a json tag, got %v", err)
	}
}
