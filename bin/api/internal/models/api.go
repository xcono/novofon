package models

// MethodInfo represents basic method information extracted from HTML
type MethodInfo struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AccessLevel string `json:"access_level"`
	HTTPMethod  string `json:"http_method"`
}

// Parameter represents a single parameter (request or response)
type Parameter struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Required       bool              `json:"required"`
	Description    string            `json:"description"`
	AllowedValues  string            `json:"allowed_values,omitempty"`
	AdditionalInfo map[string]string `json:"additional_info,omitempty"`
}

// APIData represents complete parsed API data from HTML documentation
type APIData struct {
	MethodInfo     *MethodInfo            `json:"method_info"`
	RequestParams  map[string]*Parameter  `json:"request_params"`
	ResponseParams map[string]*Parameter  `json:"response_params"`
	RequestJSON    map[string]interface{} `json:"request_json,omitempty"`
	ResponseJSON   map[string]interface{} `json:"response_json,omitempty"`
	ErrorInfo      *ErrorInfo             `json:"error_info,omitempty"`
}

// ErrorInfo represents error information extracted from HTML
type ErrorInfo struct {
	Errors []Error `json:"errors"`
}

// Error represents a single error from the documentation
type Error struct {
	Code        string `json:"code"`
	Mnemonic    string `json:"mnemonic"`
	Description string `json:"description"`
}
