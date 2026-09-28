package rpc

import (
	"reflect"
	"testing"
)

func TestParamsBuilder_Params(t *testing.T) {
	t.Run("sets params when schema is not nil and slugs are provided", func(t *testing.T) {
		schema := NewHandlerSchema("GET", "/test/:id/:name", reflect.TypeFor[string]())
		bqp := NewParamsBuilder(schema)

		slugs := []string{"id", "name"}
		result := bqp.Params(slugs)

		if result != bqp {
			t.Error("Params should return the receiver for chaining")
		}

		if len(schema.params) != 2 {
			t.Errorf("Expected 2 params to be set, got %v", schema.params)
		}
	})

	t.Run("returns early when slugs is empty", func(t *testing.T) {
		schema := NewHandlerSchema("GET", "/test", reflect.TypeFor[string]())
		bqp := NewParamsBuilder(schema)

		result := bqp.Params([]string{})

		if result != bqp {
			t.Error("Params should return the receiver for chaining")
		}
	})

	t.Run("panics when schema is nil", func(t *testing.T) {
		bqp := &ParamsBuilder{Schema: nil}
		testExpectPanic(t, "Params was called on a ParamsBuilder with no Schema", func() {
			bqp.Params([]string{"id"})
		})
	})
}

func TestNewParamsBuilder(t *testing.T) {
	schema := NewHandlerSchema("GET", "/test", reflect.TypeFor[string]())
	bqp := NewParamsBuilder(schema)

	if bqp == nil {
		t.Fatal("NewParamsBuilder should not return nil")
	}

	if bqp.Schema != schema {
		t.Error("NewParamsBuilder should wrap the provided schema")
	}
}
