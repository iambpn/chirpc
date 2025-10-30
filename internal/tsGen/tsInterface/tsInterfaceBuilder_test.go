package tsInterface

import "testing"

func TestTsInterfaceBuilderRegisterAndQuery(t *testing.T) {
	builder := NewBuilder()
	user := New(true)
	user.AddInterfaceName("User")

	builder.RegisterType(user)

	got, ok := builder.QueryType("User")
	if !ok {
		t.Fatalf("expected to find registered type")
	}
	if got != user {
		t.Fatalf("expected QueryType to return registered instance")
	}

	if _, ok := builder.QueryType("Unknown"); ok {
		t.Fatalf("expected missing type lookup to return ok=false")
	}
}

func TestTsInterfaceBuilderStringFormatting(t *testing.T) {
	builder := NewBuilder()

	user := New(true)
	user.AddInterfaceName("User")
	user.AddProperty("id", "number", false)
	builder.RegisterType(user)

	payload := New(false)
	payload.AddInterfaceName("Payload")
	payload.AddProperty("name", "string", false)
	payload.AddProperty("flag", "boolean", true)
	builder.RegisterType(payload)

	expected := "\nexport interface User { id:number; }\n\ninterface Payload { name:string; flag?:boolean; }\n"
	if got := builder.String(); got != expected {
		t.Fatalf("unexpected builder string output\nexpected: %q\nactual:   %q", expected, got)
	}
}

func TestTsInterfaceBuilderGetTypesIsLive(t *testing.T) {
	builder := NewBuilder()
	types := builder.GetTypes()

	if types == nil {
		t.Fatalf("expected GetTypes to return non-nil map")
	}

	direct := New(false)
	direct.AddInterfaceName("Direct")
	types.Set(direct.GetInterfaceName(), direct)

	got, ok := builder.QueryType("Direct")
	if !ok {
		t.Fatalf("expected QueryType to see type inserted via GetTypes")
	}
	if got != direct {
		t.Fatalf("expected QueryType to return the same instance inserted via GetTypes")
	}
}

func TestTsInterfaceBuilderEmptyString(t *testing.T) {
	builder := NewBuilder()

	if got := builder.String(); got != "\n\n" {
		t.Fatalf("expected empty builder string to be \n\n, got %q", got)
	}
}

func TestTsInterfaceBuilderRegisterOverridesExisting(t *testing.T) {
	builder := NewBuilder()

	original := New(false)
	original.AddInterfaceName("Item")
	original.AddProperty("id", "number", false)
	builder.RegisterType(original)

	replacement := New(false)
	replacement.AddInterfaceName("Item")
	replacement.AddProperty("id", "string", false)
	builder.RegisterType(replacement)

	got, ok := builder.QueryType("Item")
	if !ok {
		t.Fatalf("expected to find type after replacement")
	}
	if got != replacement {
		t.Fatalf("expected QueryType to return replacement instance")
	}
}
