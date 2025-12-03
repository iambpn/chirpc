package rpc

// RpcSchema represents the schema for an RPC endpoint, containing TypeScript type strings
// for parameter, body, query, and response types.
type RpcSchema struct {
	Param    string
	Body     string
	Query    string
	Response string
}
