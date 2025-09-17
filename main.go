package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Aman-s12345/go-openapispec-generator/internal/analyzer"
	"github.com/Aman-s12345/go-openapispec-generator/internal/generator"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectPath   string `json:"project_path"`
	OutputPath    string `json:"output_path"`
	OutputFormat  string `json:"output_format"`
	ServerURL     string `json:"server_url"`
	Title         string `json:"title"`
	Version       string `json:"version"`
	Description   string `json:"description"`
	RoutesPattern string `json:"routes_pattern"`
	SDKPackage    string `json:"sdk_package"`
}

func main() {
	// cmd line flags
	var (
		configPath   = flag.String("config", "", "Path to configuration file")
		projectPath  = flag.String("project", ".", "Path to Go project")
		outputPath   = flag.String("output", "openapi.yaml", "Output file path")
		outputFormat = flag.String("format", "yaml", "Output format (json|yaml)")
		serverURL    = flag.String("server", "http://localhost:3000", "Server URL")
		title        = flag.String("title", "VSA API Server", "API title")
		version      = flag.String("version", "1.0.0", "API version")
		description  = flag.String("description", "Voice Service API Server", "API description")
		help         = flag.Bool("h", false, "Show help")
	)
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		return
	}

	var config Config

	if *configPath != "" {
		if err := loadConfig(*configPath, &config); err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		config = Config{
			ProjectPath:  *projectPath,
			OutputPath:   *outputPath,
			OutputFormat: *outputFormat,
			ServerURL:    *serverURL,
			Title:        *title,
			Version:      *version,
			Description:  *description,
			// Default pattern for routes and SDK
			RoutesPattern: "routes/**/router.go",
			SDKPackage:    "sdk",
		}
	}

	if _, err := os.Stat(config.ProjectPath); os.IsNotExist(err) {
		log.Fatalf("Project path does not exist: %s", config.ProjectPath)
	}

	// Check for SDK directory
	sdkPath := filepath.Join(config.ProjectPath, "sdk")
	if _, err := os.Stat(sdkPath); os.IsNotExist(err) {
		fmt.Printf("WARNING: SDK directory not found at: %s\n", sdkPath)
	} else {
		fmt.Printf("SDK directory found: %s\n", sdkPath)
	}

	// Check for routes directory
	routesPath := filepath.Join(config.ProjectPath, "routes")
	if _, err := os.Stat(routesPath); os.IsNotExist(err) {
		fmt.Printf("WARNING: Routes directory not found at: %s\n", routesPath)
	} else {
		fmt.Printf("Routes directory found: %s\n", routesPath)
	}
	projectAnalyzer := analyzer.New(config.ProjectPath, config.SDKPackage, config.RoutesPattern)
	analysis, err := projectAnalyzer.Analyze()
	if err != nil {
		log.Fatalf("Failed to analyze project: %v", err)
	}

	specGenerator := generator.New(generator.Config{
		Title:       config.Title,
		Version:     config.Version,
		Description: config.Description,
		ServerURL:   config.ServerURL,
	})
	spec := specGenerator.Generate(analysis)
	if err := writeOutput(spec, config.OutputPath, config.OutputFormat); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}
	// Verify the file was created
	if _, err := os.Stat(config.OutputPath); err == nil {
		info, _ := os.Stat(config.OutputPath)
		fmt.Printf("Output file size: %d bytes\n", info.Size())
	} else {
		fmt.Printf("ERROR: Output file was not created: %v\n", err)
	}
}

func loadConfig(configPath string, config *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	return nil
}

func writeOutput(spec interface{}, outputPath, format string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	switch format {
	case "json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(spec); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	case "yaml":
		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		if err := encoder.Encode(spec); err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (supported: json, yaml)", format)
	}

	return nil
}
