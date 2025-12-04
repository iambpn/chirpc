package rpc

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
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

	t.Run("prints warning when schema is nil", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		bqp := &BodyQueryParamType{Schema: nil}
		result := bqp.BodyType("test")

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "Warning: Cannot set body type because Schema is nil") {
			t.Errorf("Expected warning message, got: %s", output)
		}

		if result != bqp {
			t.Error("BodyType should return the receiver even when schema is nil")
		}
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

	t.Run("prints warning when schema is nil", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		bqp := &BodyQueryParamType{Schema: nil}
		result := bqp.QueryType("test")

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "Warning: Cannot set query type because Schema is nil") {
			t.Errorf("Expected warning message, got: %s", output)
		}

		if result != bqp {
			t.Error("QueryType should return the receiver even when schema is nil")
		}
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

		if schema.paramsType == "" {
			t.Error("Expected params type to be set")
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

	t.Run("prints warning when schema is nil", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		bqp := &BodyQueryParamType{Schema: nil}
		result := bqp.Params([]string{"id"})

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "Warning: Cannot set params type because Schema is nil") {
			t.Errorf("Expected warning message, got: %s", output)
		}

		if result != bqp {
			t.Error("Params should return the receiver even when schema is nil")
		}
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
