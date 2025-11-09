package tsInterface

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
)

type interfaceValue struct {
	Value      string
	IsOptional bool
}

type TsInterface struct {
	shouldExport bool
	name         string
	body         *orderedmap.OrderedMap[string, *interfaceValue]
	isPrimary    bool
}

func (t *TsInterface) AddInterfaceName(headerName string) {
	t.name = headerName
}

func (t *TsInterface) GetInterfaceName() string {
	return t.name
}

func (t *TsInterface) AddProperty(key, value string, isOptional bool) {
	t.body.Set(key, &interfaceValue{
		Value:      value,
		IsOptional: isOptional,
	})
}

func (t *TsInterface) RemoveProperty(key string) {
	t.body.Delete(key)
}

func (t *TsInterface) GetProperty(key string) (*interfaceValue, error) {
	if val, ok := t.body.Get(key); ok {
		return val, nil
	}

	return nil, fmt.Errorf("key '%s' not found in TsInterface", key)
}

func (t *TsInterface) IsPrimary() bool {
	return t.isPrimary
}

func (t *TsInterface) SetPrimary(isPrimary bool) {
	t.isPrimary = isPrimary
}

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

func New(shouldExport bool) *TsInterface {
	return &TsInterface{
		shouldExport: shouldExport,
		body:         orderedmap.NewOrderedMap[string, *interfaceValue](),
	}
}
