package generate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xcono/novofon/internal/models"
	"gopkg.in/yaml.v3"
)

// OpenAPISpec represents an OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI string              `yaml:"openapi"`
	Info    OpenAPIInfo         `yaml:"info"`
	Paths   map[string]PathItem `yaml:"paths"`
	XErrors *models.ErrorInfo   `yaml:"x-errors,omitempty"`
}

// OpenAPIInfo represents the info section of OpenAPI spec
type OpenAPIInfo struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// PathItem represents a path item in OpenAPI spec
type PathItem struct {
	Post   *Operation `yaml:"post,omitempty"`
	Get    *Operation `yaml:"get,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// Operation represents an operation in OpenAPI spec
type Operation struct {
	Summary     string              `yaml:"summary"`
	Description string              `yaml:"description"`
	RequestBody *RequestBody        `yaml:"requestBody,omitempty"`
	Responses   map[string]Response `yaml:"responses"`
	Tags        []string            `yaml:"tags,omitempty"`
}

// RequestBody represents a request body in OpenAPI spec
type RequestBody struct {
	Required bool                 `yaml:"required"`
	Content  map[string]MediaType `yaml:"content"`
}

// Response represents a response in OpenAPI spec
type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty"`
}

// MediaType represents a media type in OpenAPI spec
type MediaType struct {
	Schema Schema `yaml:"schema"`
}

// Schema represents a schema in OpenAPI spec
type Schema struct {
	Type        string            `yaml:"type,omitempty"`
	Properties  map[string]Schema `yaml:"properties,omitempty"`
	Required    []string          `yaml:"required,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Example     interface{}       `yaml:"example,omitempty"`
	Enum        []interface{}     `yaml:"enum,omitempty"`
	Format      string            `yaml:"format,omitempty"`
	MaxLength   *int              `yaml:"maxLength,omitempty"`
	MinLength   *int              `yaml:"minLength,omitempty"`
	Maximum     *float64          `yaml:"maximum,omitempty"`
	Minimum     *float64          `yaml:"minimum,omitempty"`
	Items       *Schema           `yaml:"items,omitempty"`
	XFiltering  string            `yaml:"x-filtering,omitempty"`
	XSorting    string            `yaml:"x-sorting,omitempty"`
}

// OpenAPIGenerator generates OpenAPI 3.0 specifications from parsed API data
type OpenAPIGenerator struct {
	supportedTypes map[string]string
}

// NewOpenAPIGenerator creates a new OpenAPI generator
func NewOpenAPIGenerator() *OpenAPIGenerator {
	return &OpenAPIGenerator{
		supportedTypes: map[string]string{
			"string":  "string",
			"number":  "number",
			"boolean": "boolean",
			"object":  "object",
			"array":   "array",
			"enum":    "string",
		},
	}
}

// GenerateSpec generates an OpenAPI 3.0 specification from parsed API data
func (g *OpenAPIGenerator) GenerateSpec(apiData *models.APIData) (*OpenAPISpec, error) {
	if apiData == nil || apiData.MethodInfo == nil {
		return nil, fmt.Errorf("invalid API data: method info is required")
	}

	methodInfo := apiData.MethodInfo
	title := methodInfo.Title
	if title == "" {
		title = fmt.Sprintf("Novofon API - %s", methodInfo.Name)
	}

	description := methodInfo.Description
	if description == "" {
		description = fmt.Sprintf("API endpoint for %s", methodInfo.Name)
	}

	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       title,
			Version:     "1.0.0",
			Description: description,
		},
		Paths: make(map[string]PathItem),
	}

	// Add error information if available
	if apiData.ErrorInfo != nil && len(apiData.ErrorInfo.Errors) > 0 {
		spec.XErrors = apiData.ErrorInfo
	}

	// Generate path and operation
	path := fmt.Sprintf("/%s", methodInfo.Name)
	operation := g.generateOperation(apiData)

	pathItem := PathItem{}
	switch strings.ToLower(methodInfo.HTTPMethod) {
	case "get":
		pathItem.Get = operation
	case "post":
		pathItem.Post = operation
	case "put":
		pathItem.Put = operation
	case "delete":
		pathItem.Delete = operation
	default:
		pathItem.Post = operation // Default to POST for JSON-RPC
	}

	spec.Paths[path] = pathItem

	return spec, nil
}

// cleanText removes unwanted whitespace and newlines from text content
func cleanText(text string) string {
	// Remove all types of newlines and excessive whitespace
	cleaned := strings.ReplaceAll(text, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")

	// Replace multiple spaces with single space
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}

	// Trim leading and trailing whitespace
	return strings.TrimSpace(cleaned)
}

// generateOperation generates an operation from API data
func (g *OpenAPIGenerator) generateOperation(apiData *models.APIData) *Operation {
	methodInfo := apiData.MethodInfo

	operation := &Operation{
		Summary:     methodInfo.Title,
		Description: g.generateDescription(apiData),
		Responses:   g.generateResponses(apiData),
		Tags:        []string{"novofon"},
	}

	// Add request body if there are request parameters
	if len(apiData.RequestParams) > 0 {
		operation.RequestBody = g.generateRequestBody(apiData)
	}

	return operation
}

// generateDescription generates a detailed description for the operation
func (g *OpenAPIGenerator) generateDescription(apiData *models.APIData) string {
	var parts []string

	if apiData.MethodInfo.Description != "" {
		parts = append(parts, cleanText(apiData.MethodInfo.Description))
	}

	if len(apiData.RequestParams) > 0 {
		parts = append(parts, fmt.Sprintf("**Request Parameters:** %d", len(apiData.RequestParams)))
		for name, param := range apiData.RequestParams {
			required := "optional"
			if param.Required {
				required = "required"
			}
			parts = append(parts, fmt.Sprintf("- `%s` (%s, %s): %s", name, param.Type, required, cleanText(param.Description)))
		}
	}

	if len(apiData.ResponseParams) > 0 {
		parts = append(parts, fmt.Sprintf("**Response Parameters:** %d", len(apiData.ResponseParams)))
		for name, param := range apiData.ResponseParams {
			parts = append(parts, fmt.Sprintf("- `%s` (%s): %s", name, param.Type, cleanText(param.Description)))
		}
	}

	return strings.Join(parts, "\n\n")
}

// generateRequestBody generates a request body schema
func (g *OpenAPIGenerator) generateRequestBody(apiData *models.APIData) *RequestBody {
	properties := make(map[string]Schema)
	var required []string

	// Add JSON-RPC structure
	properties["jsonrpc"] = Schema{
		Type:        "string",
		Description: "JSON-RPC version",
		Example:     "2.0",
	}
	required = append(required, "jsonrpc")

	properties["id"] = Schema{
		Type:        "number",
		Description: "Request identifier",
	}
	required = append(required, "id")

	properties["method"] = Schema{
		Type:        "string",
		Description: "Method name",
		Example:     apiData.MethodInfo.Name,
	}
	required = append(required, "method")

	// Add params schema
	paramsProperties := make(map[string]Schema)
	var paramsRequired []string

	for name, param := range apiData.RequestParams {
		schema := g.generateParameterSchema(param)
		paramsProperties[name] = schema
		if param.Required {
			paramsRequired = append(paramsRequired, name)
		}
	}

	properties["params"] = Schema{
		Type:       "object",
		Properties: paramsProperties,
		Required:   paramsRequired,
	}
	required = append(required, "params")

	return &RequestBody{
		Required: true,
		Content: map[string]MediaType{
			"application/json": {
				Schema: Schema{
					Type:       "object",
					Properties: properties,
					Required:   required,
				},
			},
		},
	}
}

// generateResponses generates response schemas
func (g *OpenAPIGenerator) generateResponses(apiData *models.APIData) map[string]Response {
	responses := make(map[string]Response)

	// Success response
	successResponse := Response{
		Description: "Successful response",
		Content: map[string]MediaType{
			"application/json": {
				Schema: g.generateSuccessResponseSchema(apiData),
			},
		},
	}
	responses["200"] = successResponse

	// Error responses
	errorResponse := Response{
		Description: "Error response",
		Content: map[string]MediaType{
			"application/json": {
				Schema: g.generateErrorResponseSchema(),
			},
		},
	}
	responses["400"] = errorResponse

	// Add specific error responses if available
	if apiData.ErrorInfo != nil {
		for _, err := range apiData.ErrorInfo.Errors {
			if err.Code != "" && g.isValidHTTPStatusCode(err.Code) {
				responses[err.Code] = Response{
					Description: err.Description,
					Content: map[string]MediaType{
						"application/json": {
							Schema: g.generateErrorResponseSchema(),
						},
					},
				}
			}
		}
	}

	return responses
}

// generateSuccessResponseSchema generates the success response schema
func (g *OpenAPIGenerator) generateSuccessResponseSchema(apiData *models.APIData) Schema {
	properties := make(map[string]Schema)

	// JSON-RPC structure
	properties["jsonrpc"] = Schema{
		Type:        "string",
		Description: "JSON-RPC version",
		Example:     "2.0",
	}

	properties["id"] = Schema{
		Type:        "number",
		Description: "Request identifier",
	}

	// Result structure
	resultProperties := make(map[string]Schema)

	// Data schema
	if len(apiData.ResponseParams) > 0 {
		dataProperties := make(map[string]Schema)
		var dataRequired []string

		for name, param := range apiData.ResponseParams {
			schema := g.generateParameterSchema(param)
			dataProperties[name] = schema
			if param.Required {
				dataRequired = append(dataRequired, name)
			}
		}

		resultProperties["data"] = Schema{
			Type:       "object",
			Properties: dataProperties,
			Required:   dataRequired,
		}
	} else {
		resultProperties["data"] = Schema{
			Type: "object",
		}
	}

	// Metadata
	resultProperties["metadata"] = Schema{
		Type:        "object",
		Description: "Response metadata",
	}

	properties["result"] = Schema{
		Type:       "object",
		Properties: resultProperties,
		Required:   []string{"data", "metadata"},
	}

	return Schema{
		Type:       "object",
		Properties: properties,
		Required:   []string{"jsonrpc", "id", "result"},
	}
}

// generateErrorResponseSchema generates the error response schema
func (g *OpenAPIGenerator) generateErrorResponseSchema() Schema {
	properties := make(map[string]Schema)

	properties["jsonrpc"] = Schema{
		Type: "string",
	}

	properties["id"] = Schema{
		Type: "number",
	}

	errorProperties := make(map[string]Schema)
	errorProperties["code"] = Schema{
		Type: "number",
	}
	errorProperties["message"] = Schema{
		Type: "string",
	}
	errorProperties["data"] = Schema{
		Type: "object",
	}

	properties["error"] = Schema{
		Type:       "object",
		Properties: errorProperties,
	}

	return Schema{
		Type:       "object",
		Properties: properties,
	}
}

// generateParameterSchema generates a schema for a parameter
func (g *OpenAPIGenerator) generateParameterSchema(param *models.Parameter) Schema {
	schema := Schema{
		Type:        g.supportedTypes[param.Type],
		Description: cleanText(param.Description),
	}

	// Handle array types - add items schema
	if param.Type == "array" {
		// Use ArrayItemType from parsed parameter data
		itemType := param.ArrayItemType
		if itemType == "" {
			// Fallback to default if not determined
			itemType = "string"
		}

		schema.Items = &Schema{
			Type: itemType,
		}

		// Add example for array items
		switch itemType {
		case "string":
			schema.Items.Example = "example_string"
		case "number":
			schema.Items.Example = 123
		case "boolean":
			schema.Items.Example = true
		}
	}

	// Handle allowed values
	if param.AllowedValues != "" {
		// Check if it's a format specification
		if strings.Contains(strings.ToLower(param.AllowedValues), "формат") ||
			strings.Contains(strings.ToLower(param.AllowedValues), "format") {
			schema.Format = param.AllowedValues
		} else {
			// Try to parse as enum values
			enumValues := strings.Split(param.AllowedValues, ",")
			if len(enumValues) > 1 {
				var enum []interface{}
				for _, val := range enumValues {
					enum = append(enum, strings.TrimSpace(val))
				}
				schema.Enum = enum
				if len(enum) > 0 {
					schema.Example = enum[0]
				}
			}
		}
	}

	// Add example based on type if not already set
	if schema.Example == nil {
		switch param.Type {
		case "string":
			schema.Example = "example_string"
		case "number":
			schema.Example = 123
		case "boolean":
			schema.Example = true
		case "array":
			// For arrays, set example as empty array
			schema.Example = []interface{}{}
		}
	}

	return schema
}

// isValidHTTPStatusCode checks if a string is a valid HTTP status code
func (g *OpenAPIGenerator) isValidHTTPStatusCode(code string) bool {
	// Basic validation for 3-digit HTTP status codes
	if len(code) != 3 {
		return false
	}

	// Check if it's a valid HTTP status code range (100-599)
	if code[0] < '1' || code[0] > '5' {
		return false
	}

	// Check if all characters are digits
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// ToYAML converts the OpenAPI spec to YAML format
func (spec *OpenAPISpec) ToYAML() ([]byte, error) {
	return yaml.Marshal(spec)
}

// ToJSON converts the OpenAPI spec to JSON format
func (spec *OpenAPISpec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}
