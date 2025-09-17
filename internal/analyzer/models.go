package analyzer

type Analysis struct {
	Routes []Route
	Models map[string]Model
}

type Route struct {
	Path        string
	Method      string
	Handler     string
	Middleware  []string
	RequestBody *Model
	Response    *Model
	Parameters  []Parameter
	Tags        []string
}

type Parameter struct {
	Name        string
	In          string // "path", "query", "header"
	Required    bool
	Type        string
	Example     string
	Description string
	Default     interface{}
	Enum        []string
}

type QueryParameter struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Default     interface{}
	Enum        []string
}

type Model struct {
	Name        string
	Package     string
	Fields      []Field
	Description string
}

type Field struct {
	Name        string
	Type        string
	JSONTag     string
	OriginalType string
	Required    bool
	Description string
	Example     interface{}
}

type HandlerInfo struct {
	Name            string
	RequestType     string
	ResponseType    string
	Package         string
	QueryParameters []QueryParameter
	AnonymousRequestModel *Model 
}

type RouteGroup struct {
	Variable string // variable name like "v1", "v2"
	BasePath string // base path like "/v1", "/v2"
}