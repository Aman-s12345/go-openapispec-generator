package generator

type Generator struct {
	config Config
}

type Config struct {
	Title       string
	Version     string
	Description string
	ServerURL   string
}

type OpenAPISpec struct {
	OpenAPI    string              `json:"openapi" yaml:"openapi"`
	Info       Info                `json:"info" yaml:"info"`
	Servers    []Server            `json:"servers" yaml:"servers"`
	Paths      map[string]PathItem `json:"paths" yaml:"paths"`
	Components Components          `json:"components" yaml:"components"`
	Tags       []Tag               `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

type Server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description" yaml:"description"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post   *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put    *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
}

type Operation struct {
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses" yaml:"responses"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
}

type Parameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      Schema      `json:"schema" yaml:"schema"`
	Example     interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
}

type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type MediaType struct {
	Schema Schema `json:"schema" yaml:"schema"`
}

type Schema struct {
	Type                 string            `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string            `json:"format,omitempty" yaml:"format,omitempty"`
	Properties           map[string]Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items                *Schema           `json:"items,omitempty" yaml:"items,omitempty"`
	Required             []string          `json:"required,omitempty" yaml:"required,omitempty"`
	AdditionalProperties interface{}       `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	Ref                  string            `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Example              interface{}       `json:"example,omitempty" yaml:"example,omitempty"`
	Description          string            `json:"description,omitempty" yaml:"description,omitempty"`
	Enum                 []interface{}     `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default              interface{}       `json:"default,omitempty" yaml:"default,omitempty"`
	AllOf                []Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf                []Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf                []Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
}

type Components struct {
	Schemas         map[string]Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type         string `json:"type" yaml:"type"`
	Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}