# tsGen

TsGen

- Implementation notes:
  - unexported fields are ignored
  - nested structs are flattened and defined as separate interfaces
  - pointer types are represented as nullable types (e.g., Type | null)
  - function types are represented as 'Function'
  - interface, chan, complex types are represented as 'any'
  - invalid types are represented as 'unknown'
  - Anonymous nested struct are embedded into the parent struct
  - Anonymous field in struct are ignored
  - Support to struct tag to customize the field name and type with `tskey` and `tstype` tags
