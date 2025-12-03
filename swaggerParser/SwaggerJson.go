package swaggerParser

type SwaggerJson struct {
	Host     string                     `json:"host"`
	BasePath string                     `json:"basePath"`
	Schemes  []string                   `json:"schemes"`
	Paths    map[string]map[string]Path `json:"paths"`
}
type Path struct {
	Summary    string      `json:"summary"`
	Consumes   []string    `json:"consumes"`
	Parameters []Parameter `json:"parameters"`
}
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Schema      Schema `json:"schema"`
}
type Schema struct {
	Type       string               `json:"type"`
	Items      Items                `json:"items"`
	Properties map[string]Propertie `json:"properties"`
}
type Items struct {
	Type        string               `json:"type"`
	Description string               `json:"description"`
	Properties  map[string]Propertie `json:"properties"`
}
type Propertie struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Items       Items  `json:"items"`
}
