package tsopts

import "maps"

type TsGenOptsState int

const (
	ToLowercase TsGenOptsState = iota
	AddHeaderToInterface
	UnCapitalizeHeader
)

type TsGenOpts map[TsGenOptsState]bool

func SetToLowerExportedField(val bool) TsGenOpts {
	config := make(TsGenOpts)
	config[ToLowercase] = val
	return config
}

func SetAddHeaderToInterface(val bool) TsGenOpts {
	config := make(TsGenOpts)
	config[AddHeaderToInterface] = val
	return config
}

func SetUnCapitalizeHeader(val bool) TsGenOpts {
	config := make(TsGenOpts)
	config[UnCapitalizeHeader] = val
	return config
}

func MergeOpts(opts ...TsGenOpts) TsGenOpts {
	merged := make(TsGenOpts)
	for _, opt := range opts {
		maps.Copy(merged, opt)
	}
	return merged
}
