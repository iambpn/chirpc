package rpc

import (
	"reflect"
	"testing"
)

func TestBodyQueryParamType_BodyType(t *testing.T) {
	t.Run("sets body type when schema is not nil", func(t *testing.T) {
		schema := NewHandlerSchema("POST", "/test", reflect.TypeOf(""))
		bqp := NewBodyQueryParamType(schema)

		type TestBody struct {
			Name string
		}
		body := TestBody{}

		result := bqp.BodyType(body)

		if result != bqp {
			t.Error("BodyType should return the receiver for chaining")
		}

		if schema.bodyType == nil {
			t.Error("Expected body type to be set")
		}
	})

	t.Run("panics when schema is nil", func(t *testing.T) {
		bqp := &BodyQueryParamType{Schema: nil}
		testExpectPanic(t, "BodyType was called on a BodyQueryParamType with no Schema", func() {
			bqp.BodyType(struct{}{})
		})
	})
}

func TestBodyQueryParamType_QueryType(t *testing.T) {
	t.Run("sets query type when schema is not nil", func(t *testing.T) {
		schema := NewHandlerSchema("GET", "/test", reflect.TypeOf(""))
		bqp := NewBodyQueryParamType(schema)

		type TestQuery struct {
			Page int
		}
		query := TestQuery{}

		result := bqp.QueryType(query)

		if result != bqp {
			t.Error("QueryType should return the receiver for chaining")
		}

		if schema.queryType == nil {
			t.Error("Expected query type to be set")
		}
	})

	t.Run("panics when schema is nil", func(t *testing.T) {
		bqp := &BodyQueryParamType{Schema: nil}
		testExpectPanic(t, "QueryType was called on a BodyQueryParamType with no Schema", func() {
			bqp.QueryType(struct{}{})
		})
	})
}

func TestBodyQueryParamType_Params(t *testing.T) {
	t.Run("sets params when schema is not nil and slugs are provided", func(t *testing.T) {
		schema := NewHandlerSchema("GET", "/test/:id/:name", reflect.TypeOf(""))
		bqp := NewBodyQueryParamType(schema)

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
		schema := NewHandlerSchema("GET", "/test", reflect.TypeOf(""))
		bqp := NewBodyQueryParamType(schema)

		result := bqp.Params([]string{})

		if result != bqp {
			t.Error("Params should return the receiver for chaining")
		}
	})

	t.Run("panics when schema is nil", func(t *testing.T) {
		bqp := &BodyQueryParamType{Schema: nil}
		testExpectPanic(t, "Params was called on a BodyQueryParamType with no Schema", func() {
			bqp.Params([]string{"id"})
		})
	})
}

func TestNewBodyQueryParamType(t *testing.T) {
	schema := NewHandlerSchema("GET", "/test", reflect.TypeOf(""))
	bqp := NewBodyQueryParamType(schema)

	if bqp == nil {
		t.Fatal("NewBodyQueryParamType should not return nil")
	}

	if bqp.Schema != schema {
		t.Error("NewBodyQueryParamType should wrap the provided schema")
	}
}
