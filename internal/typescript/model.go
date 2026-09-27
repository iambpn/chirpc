package typescript

// RpcSchema represents the schema for an RPC endpoint, containing TypeScript type strings
// for parameter, body, query, and response types.
type RpcSchema struct {
	Param    string
	Body     string
	Query    string
	Response string

	// QueryRequired makes the query member required. It is set when the query type has a required field.
	QueryRequired bool
}
