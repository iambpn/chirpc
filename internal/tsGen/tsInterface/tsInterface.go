package tsInterface

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
)

// interfaceValue represents a TypeScript interface property with its type and optional flag.
type interfaceValue struct {
	Value      string
	IsOptional bool
}

// TsInterface models a TypeScript interface, supporting named and anonymous interfaces.
type TsInterface struct {
	shouldExport bool
	name         string
	body         *orderedmap.OrderedMap[string, *interfaceValue]
	isPrimary    bool
}

// AddInterfaceName sets the name of the TypeScript interface.
func (t *TsInterface) AddInterfaceName(headerName string) {
	t.name = headerName
}

// GetInterfaceName returns the name of the TypeScript interface.
func (t *TsInterface) GetInterfaceName() string {
	return t.name
}

// AddProperty adds a property to the TypeScript interface.
func (t *TsInterface) AddProperty(key, value string, isOptional bool) {
	t.body.Set(key, &interfaceValue{
		Value:      value,
		IsOptional: isOptional,
	})
}

// RemoveProperty removes a property from the TypeScript interface.
func (t *TsInterface) RemoveProperty(key string) {
	t.body.Delete(key)
}

// GetProperty retrieves a property from the TypeScript interface by key.
func (t *TsInterface) GetProperty(key string) (*interfaceValue, error) {
	if val, ok := t.body.Get(key); ok {
		return val, nil
	}

	return nil, fmt.Errorf("key '%s' not found in TsInterface", key)
}

// IsPrimary returns whether this interface is marked as primary.
// Primary interfaces are directly added by the user, while non-primary ones are
// referenced types that are automatically discovered and registered.
func (t *TsInterface) IsPrimary() bool {
	return t.isPrimary
}

// SetPrimary marks this interface as primary or secondary.
// Primary interfaces are directly added by the user, while non-primary ones are
// referenced types that are automatically discovered and registered.
func (t *TsInterface) SetPrimary(isPrimary bool) {
	t.isPrimary = isPrimary
}

// String returns the TypeScript interface as a string representation.
func (t *TsInterface) String() string {
	result := []string{}

	// if header name is empty then build an anonymous interface
	// otherwise build a named interface
	if t.name == "" {
		result = append(result, "{")
	} else {
		if t.shouldExport {
			result = append(result, "export")
		}

		result = append(result, fmt.Sprintf("interface %s {", t.name))
	}

	for el := t.body.Front(); el != nil; el = el.Next() {
		k := el.Key
		v := el.Value
		optionalSign := ""
		if v.IsOptional {
			optionalSign = "?"
		}
		result = append(result, fmt.Sprintf("%s%s:%s;", k, optionalSign, v.Value))
	}

	result = append(result, "}")

	return strings.Join(result, " ")
}

// New creates a new TsInterface instance.
func New(shouldExport bool) *TsInterface {
	return &TsInterface{
		shouldExport: shouldExport,
		body:         orderedmap.NewOrderedMap[string, *interfaceValue](),
	}
}
