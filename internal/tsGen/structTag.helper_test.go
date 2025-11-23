package tsGen

import (
	"reflect"
	"testing"

	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

type sampleStruct struct{}

func TestGetHeaderName(t *testing.T) {
	tests := []struct {
		name    string
		valType reflect.Type
		opts    tsopts.TsGenOpts
		want    string
	}{
		{
			name:    "header disabled",
			valType: reflect.TypeOf(sampleStruct{}),
			opts:    tsopts.SetAddHeaderToInterface(false),
			want:    "",
		},
		{
			name:    "adds package prefix",
			valType: reflect.TypeOf(sampleStruct{}),
			opts:    tsopts.SetAddHeaderToInterface(true),
			want:    "TsGen__SampleStruct",
		},
		{
			name:    "uncapitalized package",
			valType: reflect.TypeOf(sampleStruct{}),
			opts: tsopts.MergeOpts(
				tsopts.SetAddHeaderToInterface(true),
				tsopts.SetUnCapitalizeHeader(true),
			),
			want: "tsGen__SampleStruct",
		},
		{
			name:    "builtin type",
			valType: reflect.TypeOf(0),
			opts:    tsopts.SetAddHeaderToInterface(true),
			want:    "__Int",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := getHeaderName(tt.valType, tt.opts)
			if got != tt.want {
				t.Fatalf("getHeaderName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetHeaderNamePointer(t *testing.T) {
	got := getHeaderName(reflect.TypeOf(&sampleStruct{}), tsopts.SetAddHeaderToInterface(true))
	if got != "TsGen__SampleStruct" {
		t.Fatalf("getHeaderName(pointer) = %q, want %q", got, "TsGen__SampleStruct")
	}
}

func TestGetTagType(t *testing.T) {
	type tagged struct {
		WithTag string `tsType:"string"`
		Without string
	}

	typ := reflect.TypeOf(tagged{})

	fieldWithTag, _ := typ.FieldByName("WithTag")
	if got := getTagType(fieldWithTag); got != "string" {
		t.Fatalf("getTagType(with tag) = %q, want %q", got, "string")
	}

	fieldWithout, _ := typ.FieldByName("Without")
	if got := getTagType(fieldWithout); got != "" {
		t.Fatalf("getTagType(without tag) = %q, want empty string", got)
	}
}

func TestGetTagKey(t *testing.T) {
	type keySample struct {
		DefaultField string
		TaggedField  string `tsKey:"CustomKey"`
		JsonField    string `json:"json_key"`
	}

	typ := reflect.TypeOf(keySample{})

	defaultField, _ := typ.FieldByName("DefaultField")
	if got := getTagKey(defaultField); got != "" {
		t.Fatalf("getTagKey(default) = %q, want %q", got, "")
	}

	taggedField, _ := typ.FieldByName("TaggedField")
	if got := getTagKey(taggedField); got != "CustomKey" {
		t.Fatalf("getTagKey(tagged) = %q, want %q", got, "CustomKey")
	}

	jsonField, _ := typ.FieldByName("JsonField")
	if got := getTagKey(jsonField); got != "json_key" {
		t.Fatalf("getTagKey(tagged) = %q, want %q", got, "json_key")
	}
}

func TestGetJsonTagValue(t *testing.T) {
	type jsonSample struct {
		WithName string `json:"first_name,omitempty"`
		WithDash string `json:"-"`
		EmptyTag string `json:",omitempty"`
		NoTag    string
	}

	typ := reflect.TypeOf(jsonSample{})

	withName, _ := typ.FieldByName("WithName")
	if got := getJsonTagValue(withName); got != "first_name" {
		t.Fatalf("getJsonTagValue(name) = %q, want %q", got, "first_name")
	}

	withDash, _ := typ.FieldByName("WithDash")
	if got := getJsonTagValue(withDash); got != "-" {
		t.Fatalf("getJsonTagValue(dash) = %q, want %q", got, "-")
	}

	emptyTag, _ := typ.FieldByName("EmptyTag")
	if got := getJsonTagValue(emptyTag); got != "" {
		t.Fatalf("getJsonTagValue(empty) = %q, want empty string", got)
	}

	noTag, _ := typ.FieldByName("NoTag")
	if got := getJsonTagValue(noTag); got != "" {
		t.Fatalf("getJsonTagValue(no tag) = %q, want empty string", got)
	}
}

func TestIsFieldOptional(t *testing.T) {
	type optionalSample struct {
		ExplicitTrue  string `tsOptional:"true"`
		CapitalTrue   string `tsOptional:"TRUE"`
		ExplicitFalse string `tsOptional:"false"`
		Missing       string
		Unexpected    string `tsOptional:"sometimes"`
	}

	typ := reflect.TypeOf(optionalSample{})

	explicitTrue, _ := typ.FieldByName("ExplicitTrue")
	if !isFieldOptional(explicitTrue) {
		t.Fatalf("isFieldOptional(true) = false, want true")
	}

	capitalTrue, _ := typ.FieldByName("CapitalTrue")
	if !isFieldOptional(capitalTrue) {
		t.Fatalf("isFieldOptional(TRUE) = false, want true")
	}

	explicitFalse, _ := typ.FieldByName("ExplicitFalse")
	if isFieldOptional(explicitFalse) {
		t.Fatalf("isFieldOptional(false) = true, want false")
	}

	missing, _ := typ.FieldByName("Missing")
	if isFieldOptional(missing) {
		t.Fatalf("isFieldOptional(missing) = true, want false")
	}

	unexpected, _ := typ.FieldByName("Unexpected")
	if isFieldOptional(unexpected) {
		t.Fatalf("isFieldOptional(unexpected value) = true, want false")
	}
}

func TestIsJsonTagOptional(t *testing.T) {
	type jsonOptionalSample struct {
		WithOmitEmpty string `json:"with,omitempty"`
		WithOmitZero  string `json:"withzero,omitzero"`
		Without       string `json:"without"`
		NoTag         string
	}

	typ := reflect.TypeOf(jsonOptionalSample{})

	withOmitEmpty, _ := typ.FieldByName("WithOmitEmpty")
	if !isJsonTagOptional(withOmitEmpty) {
		t.Fatalf("isJsonTagOptional(with omitempty) = false, want true")
	}

	withOmitZero, _ := typ.FieldByName("WithOmitZero")
	if !isJsonTagOptional(withOmitZero) {
		t.Fatalf("isJsonTagOptional(with omitzero) = false, want true")
	}

	without, _ := typ.FieldByName("Without")
	if isJsonTagOptional(without) {
		t.Fatalf("isJsonTagOptional(without optional flag) = true, want false")
	}

	noTag, _ := typ.FieldByName("NoTag")
	if isJsonTagOptional(noTag) {
		t.Fatalf("isJsonTagOptional(no tag) = true, want false")
	}
}

func TestIsFieldIgnored(t *testing.T) {
	type ignoredSample struct {
		IgnoredField string `tsOmit:"true"`
		NotIgnored   string
		NotIgnored2  string `tsOmit:"false"`
	}

	typ := reflect.TypeOf(ignoredSample{})

	ignoredField, _ := typ.FieldByName("IgnoredField")
	if !isFieldOmitted(ignoredField) {
		t.Fatalf("structTagValue should be true but got %q", ignoredField.Tag.Get(structTagOmit))
	}

	notIgnored, _ := typ.FieldByName("NotIgnored")
	if isFieldOmitted(notIgnored) {
		t.Fatalf("structTagValue should be empty but got %q", notIgnored.Tag.Get(structTagOmit))
	}

	notIgnored2, _ := typ.FieldByName("NotIgnored2")
	if isFieldOmitted(notIgnored2) {
		t.Fatalf("structTagValue should be false but got %q", notIgnored2.Tag.Get(structTagOmit))
	}

}
