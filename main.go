package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xcono/novofon/internal/generate"
	"github.com/xcono/novofon/internal/parse"
	"gopkg.in/yaml.v3"
)

// BundlingConfig defines configuration for API spec bundling
type BundlingConfig struct {
	// Grouping strategy: "domain", "api_type", "none"
	GroupingStrategy string

	// Minimum files per domain to create a bundle (default: 1)
	MinFilesPerDomain int

	// Custom domain mappings (e.g., "ca_deal" -> "deal")
	DomainMappings map[string]string

	// Output directory for bundled files
	OutputDir string
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input-dir> <output-dir>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  input-dir:  Directory containing HTML files to parse\n")
		fmt.Fprintf(os.Stderr, "  output-dir: Directory to write OpenAPI YAML files\n")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Find all HTML files
	htmlFiles, err := findHTMLFiles(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding HTML files: %v\n", err)
		os.Exit(1)
	}

	parser := parse.NewParser()
	generator := generate.NewOpenAPIGenerator()

	processed := 0
	errors := 0

	for _, htmlFile := range htmlFiles {
		fmt.Printf("Processing: %s\n", htmlFile)

		// Read HTML file
		htmlContent, err := os.ReadFile(htmlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		// Parse HTML
		apiData, err := parser.ParseHTML(string(htmlContent))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		// Generate OpenAPI spec
		openAPISpec, err := generator.GenerateSpec(apiData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating OpenAPI spec for %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		// Write output file
		outputFile := getOutputFileName(htmlFile, outputDir)
		yamlContent, err := openAPISpec.ToYAML()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to YAML for %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		if err := os.WriteFile(outputFile, []byte(yamlContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputFile, err)
			errors++
			continue
		}

		fmt.Printf("Generated: %s\n", outputFile)
		processed++
	}

	fmt.Printf("\nSummary: %d files processed, %d errors\n", processed, errors)

	// Only exit with error if there were critical failures (like file read errors)
	// Parsing errors from index pages are expected and shouldn't cause failure
	// We already processed files successfully if we got here
	if processed == 0 {
		fmt.Fprintf(os.Stderr, "No files were successfully processed\n")
		os.Exit(1)
	}

	// Bundle individual spec files into unified API specs
	// Place bundled files in ./api folder from project root
	// Get the project root by going up from bin/api to the project root
	currentDir, _ := os.Getwd()
	apiDir := filepath.Join(currentDir, "spec")

	// Ensure the api directory exists
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create api directory: %v\n", err)
		apiDir = outputDir // Fallback to original output directory
	}

	config := BundlingConfig{
		GroupingStrategy:  "domain", // Use domain-based grouping
		MinFilesPerDomain: 1,        // Create bundles for any domain with 1+ files
		DomainMappings: map[string]string{
			"ca_deal":         "deal",
			"ca_contact":      "contact",
			"ca_employee":     "employee",
			"ca_sales_funnel": "sales_funnel",
		},
		OutputDir: apiDir,
	}

	if err := bundleAPISpecs(outputDir, config); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to bundle API specs: %v\n", err)
		// Don't fail the entire process for bundling errors
	}
}

func findHTMLFiles(dir string) ([]string, error) {
	var htmlFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip root-level index.html files (but allow subdirectory index.html files)
		if strings.ToLower(info.Name()) == "index.html" {
			// Only skip if this is a root-level index.html
			parentDir := filepath.Base(filepath.Dir(path))
			if parentDir == "." || parentDir == "" {
				return nil
			}
		}

		// Skip asset directories
		if strings.Contains(path, "/assets/") {
			return nil
		}

		// Include HTML files
		if strings.HasSuffix(strings.ToLower(info.Name()), ".html") {
			htmlFiles = append(htmlFiles, path)
		}

		return nil
	})

	return htmlFiles, err
}

func getOutputFileName(htmlFile, outputDir string) string {
	// Extract relative path from HTML file starting from the API type folder
	// e.g., temp-html/data_api/authentication/login_user/index.html
	// Should extract: authentication/login_user/index.html

	// Find the API type folder (data_api or call_api)
	parts := strings.Split(htmlFile, "/")
	var startIdx int
	for i, part := range parts {
		if part == "data_api" || part == "call_api" {
			startIdx = i + 1
			break
		}
	}

	if startIdx == 0 || startIdx >= len(parts) {
		// Fallback to original logic if structure doesn't match expected pattern
		relPath := strings.TrimPrefix(htmlFile, filepath.Dir(filepath.Dir(htmlFile))+"/")
		fileName := strings.ReplaceAll(relPath, "/", ".")
		fileName = strings.TrimSuffix(fileName, ".html")
		fileName = strings.TrimSuffix(fileName, ".index")
		if !strings.HasSuffix(fileName, ".yaml") {
			fileName += ".yaml"
		}
		return filepath.Join(outputDir, fileName)
	}

	// Extract the relevant path parts (domain/method/index.html)
	relevantParts := parts[startIdx:]

	// Convert path separators and remove .html extension
	fileName := strings.Join(relevantParts, ".")
	fileName = strings.TrimSuffix(fileName, ".html")

	// Remove .index suffix if present for cleaner naming
	fileName = strings.TrimSuffix(fileName, ".index")

	// Add API type prefix to preserve the original API type information
	// This helps with bundling later
	if strings.Contains(htmlFile, "/data_api/") || strings.Contains(htmlFile, "\\data_api\\") {
		fileName = "data." + fileName
	} else if strings.Contains(htmlFile, "/call_api/") || strings.Contains(htmlFile, "\\call_api\\") {
		fileName = "calls." + fileName
	}

	// Ensure it ends with .yaml
	if !strings.HasSuffix(fileName, ".yaml") {
		fileName += ".yaml"
	}

	return filepath.Join(outputDir, fileName)
}

// APIType represents the type of API (data or calls)
type APIType string

const (
	DataAPI  APIType = "data"
	CallsAPI APIType = "calls"
)

// APIFileInfo contains information about an API file
type APIFileInfo struct {
	Domain   string
	APIType  APIType
	FilePath string
}

// extractAPIFileInfo extracts domain and API type from a YAML filename
// e.g., "data.contact.get_contacts.yaml" -> "contact", "data"
// e.g., "calls.authentication.login_user.yaml" -> "authentication", "calls"
func extractAPIFileInfo(filename string, mappings map[string]string) APIFileInfo {
	baseName := filepath.Base(filename)
	// Remove .yaml/.yml extension
	nameWithoutExt := strings.TrimSuffix(strings.TrimSuffix(baseName, ".yaml"), ".yml")

	// Split by dot
	parts := strings.Split(nameWithoutExt, ".")

	var domain string
	var apiType APIType

	if len(parts) >= 2 {
		// Check if first part is API type prefix
		if parts[0] == "data" {
			apiType = DataAPI
			domain = parts[1] // Second part is domain
		} else if parts[0] == "calls" {
			apiType = CallsAPI
			domain = parts[1] // Second part is domain
		} else {
			// No API type prefix, assume data and first part is domain
			apiType = DataAPI
			domain = parts[0]
		}
	} else if len(parts) == 1 {
		// Only one part, assume data and use it as domain
		apiType = DataAPI
		domain = parts[0]
	} else {
		// Fallback
		apiType = DataAPI
		domain = "unknown"
	}

	// Apply domain mappings if they exist
	if mappedDomain, exists := mappings[domain]; exists {
		domain = mappedDomain
	}

	return APIFileInfo{
		Domain:   domain,
		APIType:  apiType,
		FilePath: filename,
	}
}

// extractDomain extracts the domain from a YAML filename (legacy function for backward compatibility)
// e.g., "contact.get_contacts.yaml" -> "contact"
// e.g., "employee.create_employees.yaml" -> "employee"
func extractDomain(filename string, mappings map[string]string) string {
	info := extractAPIFileInfo(filename, mappings)
	return info.Domain
}

// bundleAPISpecs combines individual OpenAPI spec files into unified specs
func bundleAPISpecs(outputDir string, config BundlingConfig) error {
	// Find all yaml files in the output directory
	yamlFiles, err := findYAMLFiles(outputDir)
	if err != nil {
		return fmt.Errorf("failed to find YAML files: %w", err)
	}

	if len(yamlFiles) == 0 {
		return fmt.Errorf("no YAML files found to bundle")
	}

	// Choose bundling strategy based on configuration
	switch config.GroupingStrategy {
	case "domain":
		return bundleByDomain(yamlFiles, config)
	case "api_type":
		return bundleByAPIType(yamlFiles, outputDir)
	case "none":
		fmt.Println("Bundling disabled by configuration")
		return nil
	default:
		return fmt.Errorf("unknown grouping strategy: %s", config.GroupingStrategy)
	}
}

// bundleByDomain groups files by domain and creates separate bundled files
func bundleByDomain(yamlFiles []string, config BundlingConfig) error {
	// Group files by domain and API type
	domainGroups := make(map[string]map[APIType][]string)

	for _, file := range yamlFiles {
		info := extractAPIFileInfo(file, config.DomainMappings)

		if domainGroups[info.Domain] == nil {
			domainGroups[info.Domain] = make(map[APIType][]string)
		}
		domainGroups[info.Domain][info.APIType] = append(domainGroups[info.Domain][info.APIType], file)
	}

	// Create bundled files for each domain and API type combination
	for domain, apiTypeGroups := range domainGroups {
		for apiType, files := range apiTypeGroups {
			if len(files) < config.MinFilesPerDomain {
				fmt.Printf("Skipping domain '%s' (%s): only %d files (minimum: %d)\n", domain, apiType, len(files), config.MinFilesPerDomain)
				continue
			}

			// Create domain-specific bundled file with API type suffix for calls
			var bundledFile string
			var title string
			var description string

			if apiType == CallsAPI {
				bundledFile = filepath.Join(config.OutputDir, fmt.Sprintf("%s_calls.yaml", domain))
				title = fmt.Sprintf("Novofon %s Calls API", strings.Title(domain))
				description = fmt.Sprintf("Combined %s Calls API specifications", strings.Title(domain))
			} else {
				bundledFile = filepath.Join(config.OutputDir, fmt.Sprintf("%s.yaml", domain))
				title = fmt.Sprintf("Novofon %s API", strings.Title(domain))
				description = fmt.Sprintf("Combined %s API specifications", strings.Title(domain))
			}

			if err := createBundledSpec(files, bundledFile, title, description); err != nil {
				return fmt.Errorf("failed to bundle %s %s API specs: %w", domain, apiType, err)
			}

			fmt.Printf("Bundled %d %s %s API specs into: %s\n", len(files), domain, apiType, bundledFile)
		}
	}

	return nil
}

// bundleByAPIType groups files by API type (data vs calls) - original implementation
func bundleByAPIType(yamlFiles []string, outputDir string) error {
	// Group files by API type (data vs calls)
	dataFiles := []string{}
	callFiles := []string{}

	for _, file := range yamlFiles {
		// Check if file is in a data or calls subdirectory
		if strings.Contains(file, "/data/") || strings.Contains(file, "\\data\\") {
			dataFiles = append(dataFiles, file)
		} else if strings.Contains(file, "/calls/") || strings.Contains(file, "\\calls\\") {
			callFiles = append(callFiles, file)
		}
	}

	// Bundle data API files - place at top level of outputDir parent
	if len(dataFiles) > 0 {
		// Place bundled file at the same level as data/ and calls/ directories
		parentDir := filepath.Dir(outputDir)
		bundledFile := filepath.Join(parentDir, "data.yaml")
		if err := createBundledSpec(dataFiles, bundledFile, "Novofon Data API", "Combined Data API specifications"); err != nil {
			return fmt.Errorf("failed to bundle data API specs: %w", err)
		}
		fmt.Printf("Bundled %d Data API specs into: %s\n", len(dataFiles), bundledFile)
	}

	// Bundle call API files - place at top level of outputDir parent
	if len(callFiles) > 0 {
		// Place bundled file at the same level as data/ and calls/ directories
		parentDir := filepath.Dir(outputDir)
		bundledFile := filepath.Join(parentDir, "calls.yaml")
		if err := createBundledSpec(callFiles, bundledFile, "Novofon Call API", "Combined Call API specifications"); err != nil {
			return fmt.Errorf("failed to bundle call API specs: %w", err)
		}
		fmt.Printf("Bundled %d Call API specs into: %s\n", len(callFiles), bundledFile)
	}

	return nil
}

// findYAMLFiles finds all YAML files in a directory recursively
func findYAMLFiles(dir string) ([]string, error) {
	var yamlFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(info.Name()), ".yaml") || strings.HasSuffix(strings.ToLower(info.Name()), ".yml") {
			yamlFiles = append(yamlFiles, path)
		}

		return nil
	})

	return yamlFiles, err
}

// createBundledSpec creates a single OpenAPI spec from multiple individual specs
func createBundledSpec(inputFiles []string, outputFile, title, description string) error {
	// Create the base bundled spec structure
	bundledSpec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       title,
			"version":     "1.0.0",
			"description": description,
		},
		"paths": make(map[string]interface{}),
	}

	// Process each input file
	for _, inputFile := range inputFiles {
		content, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to read %s: %v\n", inputFile, err)
			continue
		}

		var spec map[string]interface{}
		if err := yaml.Unmarshal(content, &spec); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse %s: %v\n", inputFile, err)
			continue
		}

		// Merge paths from this spec into the bundled spec
		if paths, ok := spec["paths"].(map[string]interface{}); ok {
			bundledPaths := bundledSpec["paths"].(map[string]interface{})
			for path, pathItem := range paths {
				if _, exists := bundledPaths[path]; exists {
					fmt.Fprintf(os.Stderr, "Warning: Path %s already exists, skipping from %s\n", path, inputFile)
					continue
				}
				bundledPaths[path] = pathItem
			}
		}

		// Merge components if they exist
		if components, ok := spec["components"].(map[string]interface{}); ok {
			if bundledSpec["components"] == nil {
				bundledSpec["components"] = make(map[string]interface{})
			}
			bundledComponents := bundledSpec["components"].(map[string]interface{})

			for componentType, componentData := range components {
				if bundledComponents[componentType] == nil {
					bundledComponents[componentType] = make(map[string]interface{})
				}
				targetComponents := bundledComponents[componentType].(map[string]interface{})

				if sourceComponents, ok := componentData.(map[string]interface{}); ok {
					for name, component := range sourceComponents {
						if _, exists := targetComponents[name]; !exists {
							targetComponents[name] = component
						}
					}
				}
			}
		}

		// Merge x-errors if they exist
		if xerrors, ok := spec["x-errors"]; ok {
			if bundledSpec["x-errors"] == nil {
				bundledSpec["x-errors"] = map[string]interface{}{
					"errors": []interface{}{},
				}
			}

			if bundledErrors, ok := bundledSpec["x-errors"].(map[string]interface{}); ok {
				if sourceErrors, ok := xerrors.(map[string]interface{}); ok {
					if sourceErrorList, ok := sourceErrors["errors"].([]interface{}); ok {
						if bundledErrorList, ok := bundledErrors["errors"].([]interface{}); ok {
							// Avoid duplicate errors
							for _, sourceError := range sourceErrorList {
								bundledErrors["errors"] = append(bundledErrorList, sourceError)
							}
						}
					}
				}
			}
		}
	}

	// Write the bundled spec
	bundledContent, err := yaml.Marshal(bundledSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal bundled spec: %w", err)
	}

	if err := os.WriteFile(outputFile, bundledContent, 0644); err != nil {
		return fmt.Errorf("failed to write bundled spec: %w", err)
	}

	return nil
}
