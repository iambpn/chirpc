package tsInterface

import (
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
)

type TsInterfaceBuilder struct {
	types *orderedmap.OrderedMap[string, *TsInterface]
}

func (t *TsInterfaceBuilder) RegisterType(tsType *TsInterface) {
	header := tsType.GetInterfaceName()
	t.types.Set(header, tsType)
}

func (t *TsInterfaceBuilder) QueryType(headerName string) (*TsInterface, bool) {
	return t.types.Get(headerName)
}

func (t *TsInterfaceBuilder) String() string {
	result := []string{}

	for el := t.types.Front(); el != nil; el = el.Next() {
		result = append(result, el.Value.String())
	}

	return "\n" + strings.Join(result, "\n\n") + "\n"
}

func (t *TsInterfaceBuilder) GetTypes() *orderedmap.OrderedMap[string, *TsInterface] {
	return t.types
}

func NewBuilder() *TsInterfaceBuilder {
	return &TsInterfaceBuilder{
		types: orderedmap.NewOrderedMap[string, *TsInterface](),
	}
}
