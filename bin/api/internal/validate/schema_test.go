package validate

import (
	"testing"

	"github.com/xcono/novofon/bin/api/internal/models"
)

func TestSchemaValidator_AddSchema(t *testing.T) {
	validator := NewSchemaValidator()

	// Test adding schema from string
	schemaStr := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`

	err := validator.AddSchema("test_schema", schemaStr)
	if err != nil {
		t.Fatalf("AddSchema failed: %v", err)
	}

	// Test adding schema from map
	schemaMap := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "number",
			},
		},
		"required": []string{"id"},
	}

	err = validator.AddSchema("test_schema2", schemaMap)
	if err != nil {
		t.Fatalf("AddSchema failed: %v", err)
	}
}

func TestSchemaValidator_Validate(t *testing.T) {
	validator := NewSchemaValidator()

	// Add a test schema
	schemaStr := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number", "minimum": 0}
		},
		"required": ["name"]
	}`

	err := validator.AddSchema("test_schema", schemaStr)
	if err != nil {
		t.Fatalf("AddSchema failed: %v", err)
	}

	// Test valid data
	validData := map[string]interface{}{
		"name": "John",
		"age":  30,
	}

	result, err := validator.Validate("test_schema", validData)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if !result.Valid {
		t.Error("Expected validation to pass")
	}

	// Test invalid data (missing required field)
	invalidData := map[string]interface{}{
		"age": 30,
	}

	result, err = validator.Validate("test_schema", invalidData)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected validation to fail")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors")
	}
}

func TestSchemaValidator_Validate_NonExistentSchema(t *testing.T) {
	validator := NewSchemaValidator()

	_, err := validator.Validate("non_existent", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for non-existent schema")
	}
}

func TestSchemaValidator_GenerateSchemaFromAPIData(t *testing.T) {
	validator := NewSchemaValidator()

	apiData := &models.APIData{
		MethodInfo: &models.MethodInfo{
			Name:        "test.method",
			Title:       "Test Method",
			Description: "A test method",
		},
		RequestParams: map[string]*models.Parameter{
			"param1": {
				Name:          "param1",
				Type:          "string",
				Required:      true,
				Description:   "First parameter",
				AllowedValues: "value1, value2",
			},
			"param2": {
				Name:        "param2",
				Type:        "number",
				Required:    false,
				Description: "Second parameter",
			},
		},
	}

	schema, err := validator.GenerateSchemaFromAPIData(apiData)
	if err != nil {
		t.Fatalf("GenerateSchemaFromAPIData failed: %v", err)
	}

	// Test basic schema structure
	if schema["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Error("Expected correct $schema")
	}

	if schema["type"] != "object" {
		t.Error("Expected type to be object")
	}

	if schema["title"] != "Test Method" {
		t.Error("Expected correct title")
	}

	// Test properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Test JSON-RPC structure
	if properties["jsonrpc"] == nil {
		t.Error("Expected jsonrpc property")
	}

	if properties["method"] == nil {
		t.Error("Expected method property")
	}

	if properties["params"] == nil {
		t.Error("Expected params property")
	}

	// Test params structure
	params, ok := properties["params"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected params to be a map")
	}

	paramsProps, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected params properties to be a map")
	}

	if paramsProps["param1"] == nil {
		t.Error("Expected param1 in params")
	}

	if paramsProps["param2"] == nil {
		t.Error("Expected param2 in params")
	}

	// Test required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Expected required to be a string slice")
	}

	if len(required) != 1 || required[0] != "param1" {
		t.Error("Expected param1 to be required")
	}
}

func TestSchemaValidator_ValidateAPIData(t *testing.T) {
	validator := NewSchemaValidator()

	apiData := &models.APIData{
		MethodInfo: &models.MethodInfo{
			Name:        "test.method",
			Title:       "Test Method",
			Description: "A test method",
		},
		RequestParams: map[string]*models.Parameter{
			"param1": {
				Name:        "param1",
				Type:        "string",
				Required:    true,
				Description: "First parameter",
			},
		},
	}

	result, err := validator.ValidateAPIData(apiData)
	if err != nil {
		t.Fatalf("ValidateAPIData failed: %v", err)
	}

	// The validation should pass with the generated test data
	if !result.Valid {
		t.Error("Expected validation to pass")
		t.Logf("Validation errors: %+v", result.Errors)
	}
}

func TestSchemaValidator_GenerateParameterSchema(t *testing.T) {
	validator := NewSchemaValidator()

	tests := []struct {
		name     string
		param    *models.Parameter
		expected string
	}{
		{
			name: "string parameter",
			param: &models.Parameter{
				Type:        "string",
				Description: "A string parameter",
			},
			expected: "string",
		},
		{
			name: "number parameter",
			param: &models.Parameter{
				Type:        "number",
				Description: "A number parameter",
			},
			expected: "number",
		},
		{
			name: "boolean parameter",
			param: &models.Parameter{
				Type:        "boolean",
				Description: "A boolean parameter",
			},
			expected: "boolean",
		},
		{
			name: "parameter with enum",
			param: &models.Parameter{
				Type:          "string",
				Description:   "A parameter with enum",
				AllowedValues: "value1, value2, value3",
			},
			expected: "string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema := validator.generateParameterSchema(test.param)
			if schema["type"] != test.expected {
				t.Errorf("Expected type %s, got %s", test.expected, schema["type"])
			}
		})
	}
}

func TestContainsFormatSpec(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"E.164 format", true},
		{"International format", true},
		{"формат E.164", true},
		{"value1, value2", false},
		{"true, false", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := containsFormatSpec(test.input)
			if result != test.expected {
				t.Errorf("For input %s, expected %v, got %v", test.input, test.expected, result)
			}
		})
	}
}

func TestParseEnumValues(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"value1, value2, value3", []string{"value1", "value2", "value3"}},
		{"true, false", []string{"true", "false"}},
		{"single", []string{"single"}},
		{"", []string{}},
		{"value1, , value3", []string{"value1", "value3"}},
		{" value1 , value2 ", []string{"value1", "value2"}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := parseEnumValues(test.input)
			if len(result) != len(test.expected) {
				t.Errorf("Expected %d values, got %d", len(test.expected), len(result))
				return
			}
			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("Expected %s at index %d, got %s", expected, i, result[i])
				}
			}
		})
	}
}
