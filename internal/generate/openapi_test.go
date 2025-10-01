package generate

import (
	"testing"

	"github.com/xcono/novofon/internal/models"
)

func TestOpenAPIGenerator_GenerateSpec(t *testing.T) {
	// Create test API data
	apiData := &models.APIData{
		MethodInfo: &models.MethodInfo{
			Name:        "test.method",
			Title:       "Test Method",
			Description: "A test method for validation",
			HTTPMethod:  "post",
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
		ResponseParams: map[string]*models.Parameter{
			"result": {
				Name:        "result",
				Type:        "string",
				Required:    true,
				Description: "Result value",
			},
		},
		ErrorInfo: &models.ErrorInfo{
			Errors: []models.Error{
				{
					Code:        "-32602",
					Mnemonic:    "test_error",
					Description: "Test error description",
				},
			},
		},
	}

	generator := NewOpenAPIGenerator()
	spec, err := generator.GenerateSpec(apiData)
	if err != nil {
		t.Fatalf("GenerateSpec failed: %v", err)
	}

	// Test basic structure
	if spec.OpenAPI != "3.0.0" {
		t.Errorf("Expected OpenAPI version 3.0.0, got %s", spec.OpenAPI)
	}

	if spec.Info.Title != "Test Method" {
		t.Errorf("Expected title 'Test Method', got %s", spec.Info.Title)
	}

	if spec.Info.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", spec.Info.Version)
	}

	// Test path generation
	path := "/test.method"
	pathItem, exists := spec.Paths[path]
	if !exists {
		t.Fatalf("Expected path %s not found", path)
	}

	if pathItem.Post == nil {
		t.Fatal("Expected POST operation not found")
	}

	operation := pathItem.Post
	if operation.Summary != "Test Method" {
		t.Errorf("Expected summary 'Test Method', got %s", operation.Summary)
	}

	// Test request body
	if operation.RequestBody == nil {
		t.Fatal("Expected request body not found")
	}

	if !operation.RequestBody.Required {
		t.Error("Expected request body to be required")
	}

	// Test response
	if _, exists := operation.Responses["200"]; !exists {
		t.Fatal("Expected 200 response not found")
	}

	// Test error info
	if spec.XErrors == nil {
		t.Fatal("Expected error info not found")
	}

	if len(spec.XErrors.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(spec.XErrors.Errors))
	}
}

func TestOpenAPIGenerator_GenerateSpec_EmptyData(t *testing.T) {
	generator := NewOpenAPIGenerator()

	// Test with nil data
	_, err := generator.GenerateSpec(nil)
	if err == nil {
		t.Error("Expected error for nil data")
	}

	// Test with empty method info
	apiData := &models.APIData{}
	_, err = generator.GenerateSpec(apiData)
	if err == nil {
		t.Error("Expected error for empty method info")
	}
}

func TestOpenAPISpec_ToYAML(t *testing.T) {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]PathItem),
	}

	yamlData, err := spec.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	if len(yamlData) == 0 {
		t.Error("Expected non-empty YAML data")
	}

	// Check for basic YAML structure
	yamlStr := string(yamlData)
	if !contains(yamlStr, "openapi: 3.0.0") {
		t.Error("Expected OpenAPI version in YAML")
	}
	if !contains(yamlStr, "title: Test API") {
		t.Error("Expected title in YAML")
	}
}

func TestOpenAPISpec_ToJSON(t *testing.T) {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]PathItem),
	}

	jsonData, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON data")
	}

	// Check for basic JSON structure
	jsonStr := string(jsonData)
	if !contains(jsonStr, `"OpenAPI"`) {
		t.Error("Expected OpenAPI version in JSON")
	}
	if !contains(jsonStr, `"Title"`) {
		t.Error("Expected title in JSON")
	}
}

func TestOpenAPIGenerator_GenerateParameterSchema(t *testing.T) {
	generator := NewOpenAPIGenerator()

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
			schema := generator.generateParameterSchema(test.param)
			if schema.Type != test.expected {
				t.Errorf("Expected type %s, got %s", test.expected, schema.Type)
			}
		})
	}
}

func TestOpenAPIGenerator_IsValidHTTPStatusCode(t *testing.T) {
	generator := NewOpenAPIGenerator()

	tests := []struct {
		code     string
		expected bool
	}{
		{"200", true},
		{"404", true},
		{"500", true},
		{"100", true},
		{"599", true},
		{"99", false},  // Too short
		{"600", false}, // Too high
		{"abc", false}, // Not numeric
		{"2a0", false}, // Contains letter
		{"", false},    // Empty
	}

	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			result := generator.isValidHTTPStatusCode(test.code)
			if result != test.expected {
				t.Errorf("For code %s, expected %v, got %v", test.code, test.expected, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
