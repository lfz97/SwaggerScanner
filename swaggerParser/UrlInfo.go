package swaggerParser

// UrlInfo defines the structure for storing URL information
type UrlInfo struct {
	FullPath    string
	Method      string
	Summary     string
	ContentType string
	Parameters  []UrlInfoParameter
}
type UrlInfoParameter struct {
	Name        string
	Type        string
	In          string
	Description string
	// Schema is used for "body" parameters to describe the payload structure.
	Schema UrlInfoParameterSchema
}

// UrlInfoParameterSchema describes the structure of a parameter, especially for complex objects in the body.
type UrlInfoParameterSchema struct {
	Type string
	// Properties for "object" type schema.
	Properties map[string]UrlInfoParameterSchemaProperty
	// Items for "array" type schema.
	Items *UrlInfoParameterSchema
}

// UrlInfoParameterSchemaProperty defines a property within an object schema.
type UrlInfoParameterSchemaProperty struct {
	Type        string
	Description string
	// Items for "array" type property.
	Items *UrlInfoParameterSchema
}
