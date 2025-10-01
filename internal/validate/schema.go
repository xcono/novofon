package validate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xcono/novofon/internal/models"
	"github.com/xeipuuv/gojsonschema"
)

// SchemaValidator validates JSON data against JSON schemas
type SchemaValidator struct {
	schemas map[string]*gojsonschema.Schema
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas: make(map[string]*gojsonschema.Schema),
	}
}

// AddSchema adds a JSON schema to the validator
func (v *SchemaValidator) AddSchema(name string, schemaData interface{}) error {
	var loader gojsonschema.JSONLoader

	switch data := schemaData.(type) {
	case string:
		loader = gojsonschema.NewStringLoader(data)
	case []byte:
		loader = gojsonschema.NewBytesLoader(data)
	case map[string]interface{}:
		loader = gojsonschema.NewGoLoader(data)
	default:
		return fmt.Errorf("unsupported schema data type: %T", schemaData)
	}

	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	v.schemas[name] = schema
	return nil
}

// Validate validates JSON data against a named schema
func (v *SchemaValidator) Validate(schemaName string, data interface{}) (*ValidationResult, error) {
	schema, exists := v.schemas[schemaName]
	if !exists {
		return nil, fmt.Errorf("schema %s not found", schemaName)
	}

	var loader gojsonschema.JSONLoader
	switch d := data.(type) {
	case string:
		loader = gojsonschema.NewStringLoader(d)
	case []byte:
		loader = gojsonschema.NewBytesLoader(d)
	case map[string]interface{}:
		loader = gojsonschema.NewGoLoader(d)
	default:
		// Try to marshal to JSON first
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
		}
		loader = gojsonschema.NewBytesLoader(jsonData)
	}

	result, err := schema.Validate(loader)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &ValidationResult{
		Valid:  result.Valid(),
		Errors: v.convertErrors(result.Errors()),
	}, nil
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field       string `json:"field"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Value       string `json:"value,omitempty"`
}

// convertErrors converts gojsonschema errors to our error format
func (v *SchemaValidator) convertErrors(errors []gojsonschema.ResultError) []ValidationError {
	var validationErrors []ValidationError

	for _, err := range errors {
		validationError := ValidationError{
			Field:       err.Field(),
			Type:        err.Type(),
			Description: err.Description(),
		}

		if err.Value() != nil {
			validationError.Value = fmt.Sprintf("%v", err.Value())
		}

		validationErrors = append(validationErrors, validationError)
	}

	return validationErrors
}

// GenerateSchemaFromAPIData generates a JSON schema from parsed API data
func (v *SchemaValidator) GenerateSchemaFromAPIData(apiData *models.APIData) (map[string]interface{}, error) {
	if apiData == nil || apiData.MethodInfo == nil {
		return nil, fmt.Errorf("invalid API data")
	}

	schema := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"type":        "object",
		"title":       apiData.MethodInfo.Title,
		"description": apiData.MethodInfo.Description,
	}

	// JSON-RPC structure
	properties := map[string]interface{}{
		"jsonrpc": map[string]interface{}{
			"type":        "string",
			"description": "JSON-RPC version",
			"const":       "2.0",
		},
		"id": map[string]interface{}{
			"type":        "number",
			"description": "Request identifier",
		},
		"method": map[string]interface{}{
			"type":        "string",
			"description": "Method name",
			"const":       apiData.MethodInfo.Name,
		},
	}

	// Add params schema
	if len(apiData.RequestParams) > 0 {
		paramsProperties := make(map[string]interface{})
		var required []string

		for name, param := range apiData.RequestParams {
			paramSchema := v.generateParameterSchema(param)
			paramsProperties[name] = paramSchema
			if param.Required {
				required = append(required, name)
			}
		}

		properties["params"] = map[string]interface{}{
			"type":       "object",
			"properties": paramsProperties,
			"required":   required,
		}
	}

	schema["properties"] = properties
	schema["required"] = []string{"jsonrpc", "id", "method"}

	return schema, nil
}

// generateParameterSchema generates a JSON schema for a parameter
func (v *SchemaValidator) generateParameterSchema(param *models.Parameter) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        v.mapParameterType(param.Type),
		"description": param.Description,
	}

	// Handle allowed values
	if param.AllowedValues != "" {
		// Check if it's a format specification
		if containsFormatSpec(param.AllowedValues) {
			schema["format"] = param.AllowedValues
		} else {
			// Try to parse as enum values
			enumValues := parseEnumValues(param.AllowedValues)
			if len(enumValues) > 0 {
				schema["enum"] = enumValues
			}
		}
	}

	// Add type-specific constraints
	switch param.Type {
	case "string":
		if param.AllowedValues == "" {
			schema["example"] = "example_string"
		}
	case "number":
		schema["example"] = 123
	case "boolean":
		schema["example"] = true
	}

	return schema
}

// mapParameterType maps our parameter types to JSON schema types
func (v *SchemaValidator) mapParameterType(paramType string) string {
	switch paramType {
	case "string":
		return "string"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "object":
		return "object"
	case "array":
		return "array"
	default:
		return "string" // Default fallback
	}
}

// containsFormatSpec checks if the allowed values contain format specifications
func containsFormatSpec(allowedValues string) bool {
	lower := strings.ToLower(allowedValues)
	return strings.Contains(lower, "формат") ||
		strings.Contains(lower, "format") ||
		strings.Contains(lower, "e.164") ||
		strings.Contains(lower, "international")
}

// parseEnumValues parses comma-separated enum values
func parseEnumValues(allowedValues string) []string {
	values := strings.Split(allowedValues, ",")
	var enum []string

	for _, val := range values {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			enum = append(enum, trimmed)
		}
	}

	return enum
}

// ValidateAPIData validates API data against generated schemas
func (v *SchemaValidator) ValidateAPIData(apiData *models.APIData) (*ValidationResult, error) {
	// Generate schema from API data
	schema, err := v.GenerateSchemaFromAPIData(apiData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	// Add schema to validator
	schemaName := fmt.Sprintf("api_%s", apiData.MethodInfo.Name)
	if err := v.AddSchema(schemaName, schema); err != nil {
		return nil, fmt.Errorf("failed to add schema: %w", err)
	}

	// Create test data for validation
	testData := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  apiData.MethodInfo.Name,
	}

	if len(apiData.RequestParams) > 0 {
		params := make(map[string]interface{})
		for name, param := range apiData.RequestParams {
			// Add example values based on type
			switch param.Type {
			case "string":
				params[name] = "example"
			case "number":
				params[name] = 123
			case "boolean":
				params[name] = true
			default:
				params[name] = "example"
			}
		}
		testData["params"] = params
	}

	// Validate the test data
	return v.Validate(schemaName, testData)
}
