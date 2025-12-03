package tsopts

import "maps"

// TsGenOptsState represents the state options for TypeScript code generation.
type TsGenOptsState int

const (
	// ToLowercase indicates whether exported fields should be converted to lowercase.
	ToLowercase TsGenOptsState = iota
	// AddHeaderToInterface indicates whether to generate a named TypeScript interface.
	// When false, an anonymous interface is generated instead.
	AddHeaderToInterface
	// UnCapitalizeHeader indicates whether to uncapitalize the header in the generated TypeScript code.
	UnCapitalizeHeader
)

// TsGenOpts is a map of TsGenOptsState to bool, representing enabled/disabled options for TypeScript generation.
type TsGenOpts map[TsGenOptsState]bool

// SetToLowercaseExportedField returns a TsGenOpts with the ToLowercase option set to the given value.
func SetToLowercaseExportedField(val bool) TsGenOpts {
	config := make(TsGenOpts)
	config[ToLowercase] = val
	return config
}

// SetAddHeaderToInterface returns a TsGenOpts with the AddHeaderToInterface option set to the given value.
func SetAddHeaderToInterface(val bool) TsGenOpts {
	config := make(TsGenOpts)
	config[AddHeaderToInterface] = val
	return config
}

// SetUnCapitalizeHeader returns a TsGenOpts with the UnCapitalizeHeader option set to the given value.
func SetUnCapitalizeHeader(val bool) TsGenOpts {
	config := make(TsGenOpts)
	config[UnCapitalizeHeader] = val
	return config
}

// MergeOpts merges multiple TsGenOpts into a single TsGenOpts.
// When the same option key appears in multiple maps, later values override earlier ones.
func MergeOpts(opts ...TsGenOpts) TsGenOpts {
	merged := make(TsGenOpts)
	for _, opt := range opts {
		maps.Copy(merged, opt)
	}
	return merged
}
