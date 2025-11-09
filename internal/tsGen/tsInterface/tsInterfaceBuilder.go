package tsInterface

import (
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
)

// TsInterfaceBuilder is responsible for managing and building TypeScript interfaces.
type TsInterfaceBuilder struct {
	types *orderedmap.OrderedMap[string, *TsInterface]
}

// RegisterType adds a new TsInterface to the builder.
func (t *TsInterfaceBuilder) RegisterType(tsType *TsInterface) {
	header := tsType.GetInterfaceName()
	t.types.Set(header, tsType)
}

// GetType retrieves a TsInterface by its header name.
// Returns the TsInterface and a boolean indicating if it was found.
func (t *TsInterfaceBuilder) GetType(headerName string) (*TsInterface, bool) {
	return t.types.Get(headerName)
}

// String returns the string representation of all registered TypeScript interfaces.
func (t *TsInterfaceBuilder) String() string {
	result := []string{}

	for el := t.types.Front(); el != nil; el = el.Next() {
		result = append(result, el.Value.String())
	}

	return "\n" + strings.Join(result, "\n\n") + "\n"
}

// GetTypes returns the ordered map of all registered TypeScript interfaces.
func (t *TsInterfaceBuilder) GetTypes() *orderedmap.OrderedMap[string, *TsInterface] {
	return t.types
}

// NewBuilder creates and returns a new TsInterfaceBuilder instance.
func NewBuilder() *TsInterfaceBuilder {
	return &TsInterfaceBuilder{
		types: orderedmap.NewOrderedMap[string, *TsInterface](),
	}
}
