# OpenAPI Generator for Go Projects

A powerful Go-based tool that automatically generates OpenAPI 3.0.3 specifications from your Go source code. This tool analyzes your Go project structure, extracts API routes, models, and handlers to create comprehensive OpenAPI documentation.

## 🚀 Features

- **Automatic Route Discovery**: Scans your Go project for HTTP routes and handlers
- **Model Extraction**: Analyzes struct definitions to generate schema components
- **Query Parameter Detection**: Automatically detects and documents query parameters
- **Request/Response Mapping**: Maps request bodies and response types to OpenAPI schemas
- **Multiple Output Formats**: Supports both JSON and YAML output formats
- **Fiber Framework Support**: Optimized for Go Fiber web framework
- **Middleware Detection**: Identifies authentication and other middleware
- **Path Parameter Extraction**: Automatically extracts path parameters from routes

## 📁 Project Structure

```bash
openapi-generator/
├── internal/
│   ├── analyzer/          # Source code analysis
│   │   ├── analyzer.go    # Main analyzer logic
│   │   ├── extract.go     # Type and parameter extraction
│   │   ├── models.go      # Data models
│   │   ├── parser.go      # Go AST parsing
│   │   ├── routes.go      # Route analysis
│   │   └── utils.go       # Utility functions
│   └── generator/         # OpenAPI spec generation
│       ├── generator.go   # Main generator
│       ├── helper.go      # Operation generation helpers
│       ├── models.go      # OpenAPI spec models
│       ├── convert.go     # Path format conversion
│       ├── utils.go       # Generator utilities
│       ├── validator.go   # Spec validation
│       ├── updater.go     # Reference updates
│       └── remover.go     # Invalid reference cleanup
├── main.go               # CLI entry point
└── README.md
```

## 🛠 Installation

### Prerequisites

- Go 1.19 or higher

### Build from Source

```bash
git clone https://github.com/Aman-s12345/go-openapispec-generator.git
cd go-openapi-generator
go build -o go-openapi-generator .
```

### Install via Go

```bash
go install github.com/Aman-s12345/go-openapispec-generator@latest
```

## 📖 Usage

### Basic Usage

```bash
 ./go-openapi-generator.exe -project "Path//to//project" -output output.yaml
```

### Command Line Options

```bash
./go-openapi-generator.exe [OPTIONS]
./go-openapi-generator.exe --help

Options:
  -project string
        Path to Go project (default ".")
  -output string
        Output file path (default "openapi.yaml")
  -format string
        Output format (json|yaml) (default "yaml")
  -server string
        Server URL (default "http://localhost:3000")
  -title string
        API title (default "API Server")
  -version string
        API version (default "1.0.0")
  -description string
        API description (default "Generated API Documentation")
  -config string
        Path to configuration file
  -h    Show help
```

### Configuration File

You can use a JSON configuration file instead of command-line arguments:

```bash
{
  "project_path": "/path/to/your/project",
  "output_path": "api-docs.yaml",
  "output_format": "yaml",
  "server_url": "https://api.yourservice.com",
  "title": "Your API",
  "version": "2.0.0",
  "description": "Your API Documentation",
  "routes_pattern": "routes/**/router.go",
  "sdk_package": "models"
}
```

Run with config file:

```bash
./go-openapi-generator.exe -config config.json
```

## 🔧 Customization

### Hardcoded Tags and Descriptions

If you want to use this for any other project apart from vsa-ai. You may need to change some hardcoded tag descriptions that you may want to customize for your project. These are located in \`internal/generator/helper.


```bash
// File: internal/generator/helper.go
// Lines: ~130-150

descriptions := map[string]string{
    "conversation":      "Conversation management endpoints",
    "tenant":            "Tenant configuration endpoints", 
    "voice":             "Voice management endpoints",
    "aimodel":           "AI model configuration endpoints",
    "knowledgebase":     "Knowledge base management endpoints",
    "user":              "User authentication endpoints",
    "upload":            "File upload endpoints",
    "whatsapp":          "WhatsApp integration endpoints",
    "insights":          "Analytics and insights endpoints",
    "campaign":          "Campaign management endpoints",
    "contacts":          "Contact management endpoints",
    "event":             "Event management endpoints",
    "platformproviders": "Platform provider configuration endpoints",
    "toolcall":          "Tool call management endpoints",
    "pipeline":          "Pipeline management endpoints",
    "documents":         "Document management endpoints",
    "twilio":            "Twilio integration endpoints",
    "me":                "User profile endpoints",
    "sockets":           "WebSocket endpoints",

```

### How to Customize for Your Project

1. **Fork the repository** or create your own repo:
You need to to change these hard coded value. They are the description of routes.
You can add your own description of routes.
2. **Update the tag descriptions** in \`internal/generator/helper.go\`:
```bash
   descriptions := map[string]string{
       "users":     "User management endpoints",
       "products":  "Product catalog endpoints",
       "orders":    "Order processing endpoints",
       "payments":  "Payment processing endpoints",
       // Add your own tags here
   
```
3. **Update default values** in \`main.go\`:
You can change your Project Name & Description or you can pass this data via cmd as well as we discuss above.
```bash
   title        = flag.String("title", "Your API Server", "API title")
   description  = flag.String("description", "Your API Documentation", "API description")
```
4. **Customize the project structure expectations**:
Your project must follow these bellow pattern. So you need to check the routes and if they are in other pattern. You will need to change it. The good part is you only may need to change the route definitions not the implementation.
   - Update \`routes_pattern\` if your routes are in a different location
   - Update \`sdk_package\` if your models are in a different package

## 📋 Expected Project Structure

The generator expects your Go project to follow this structure:

```bash
your-go-project/
├── sdk/                  # Models/Schemas (configurable)
│   ├── user.go
│   ├── product.go
│   └── ...
├── routes/               # Route definitions
│   ├── users/
│   │   ├── router.go     # Route registrations
│   │   └── handlers.go   # Handler functions
│   ├── products/
│   │   ├── router.go
│   │   └── handlers.go
│   \└── ...
└── main.go
```

### Route Registration Pattern

Your \`router.go\` files should contain a \`RegisterRoutes\` function:

```bash
func RegisterRoutes(router fiber.Router) {
    v1 := router.Group("/v1")

    v1.Get("/users", GetUsers)
    v1.Post("/users", CreateUser)
    v1.Get("/users/:id", GetUser)
    v1.Put("/users/:id", UpdateUser)
    v1.Delete("/users/:id", DeleteUser)

```

### Handler Function Pattern

Your handlers should follow the Fiber pattern:

```bash
func GetUsers(c *fiber.Ctx) error {
    // Query parameters format
    page := c.QueryInt("page", 1)
    limit := c.QueryInt("limit", 10)
    search := c.Query("search")

    // Call service
    users, err := userService.GetUsers(page, limit, search)
    if err != nil {
        return c.Status(500).JSON(sdk.Response(
            Success: false,
			Message: message,
        ))}
    }

    return c.JSON(users)
}


func GetAll(c *fiber.Ctx) error {
	pr := providers.GetProviders(c)
	payload, err := pr.S.Knowledgebase.GetAll(c.Context())
	log := config.GetLogger(c)
	if err != nil {
		status := http.StatusInternalServerError
		message := fmt.Errorf("failed to get knowledge bases. %w", err).Error()
		log.Errorw("error getting knowledge bases", "error", err)

// Response format
		return c.Status(status).JSON(sdk.KnowledgeBasesResponse{
			Success: false,
			Message: message,
		})
	}

	return c.Status(http.StatusOK).JSON(sdk.KnowledgeBasesResponse{
		Success: true,
		Message: "Got knowledge bases",
		Data:    payload,
	})
}
```

## 🔍 Supported Patterns

### Query Parameters

The generator automatically detects:
- \`c.Query("param")\` – String parameters
- \`c.QueryInt("param", default)\` – Integer parameters  
- \`c.QueryBool("param")\` – Boolean parameters
- \`c.QueryFloat("param")\` – Float parameters
- \`c.QueryParser(&struct{})\` – Struct-based query parsing

### Request Bodies

- \`c.BodyParser(&struct{})\` – JSON request bodies
- Anonymous structs in handler functions
- Referenced models from SDK package

### Response Types

- Direct model returns: \`c.JSON(userResponse)\`
- Service call results: \`c.JSON(service.GetUser())\`
- Standard responses: \`fiber.Map\` responses
- Error responses
