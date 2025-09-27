# Novofon HTML Parser

A Go-based HTML parser for extracting API documentation from Novofon HTML files. This tool parses HTML documentation and extracts comprehensive API information including method details, parameters, and error information.

## 🚀 Features

- **HTML Parsing**: Robust HTML parsing using goquery library
- **Method Extraction**: Extracts method name, title, description, and HTTP method
- **Parameter Parsing**: Parses request and response parameters with types, requirements, and descriptions
- **Error Information**: Extracts error codes, mnemonics, and descriptions
- **JSON Parsing**: Extracts and parses JSON request/response examples
- **OpenAPI Generation**: Generates OpenAPI 3.0 specifications
- **JSON Schema Validation**: Validates data against JSON schemas using gojsonschema
- **YAML Support**: Output in YAML format using go-yaml
- **Multiple Output Formats**: JSON, YAML, and OpenAPI specifications
- **CLI Tool**: Command-line interface with comprehensive options
- **Comprehensive Testing**: High test coverage across all packages
- **Debugging Support**: Integrated with Delve debugger

## 📦 Installation

### Prerequisites

- Go 1.19 or later
- Git

### Build from Source

```bash
git clone https://github.com/xcono/novofon.git
cd novofon/bin/api
make dev-setup
make build
```

## 🛠️ Usage

### CLI Tool

```bash
# Show help and available options
./parser -help

# Parse HTML file and output to stdout (JSON format)
./parser -file <path-to-html-file>

# Parse with verbose output
./parser -file <path-to-html-file> -verbose

# Parse and save output to file
./parser -file <path-to-html-file> -output result.json -verbose

# Output in YAML format
./parser -file <path-to-html-file> -format yaml -output result.yaml -verbose

# Generate OpenAPI specification
./parser -file <path-to-html-file> -openapi api.yaml -verbose

# Validate parsed data against JSON schema
./parser -file <path-to-html-file> -validate -verbose

# Output OpenAPI format directly
./parser -file <path-to-html-file> -format openapi -verbose
```

### Programmatic Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/xcono/novofon/bin/api/internal/parse"
)

func main() {
    // Read HTML content
    htmlContent := `<html>...</html>`
    
    // Parse HTML
    parser := parse.NewParser()
    apiData, err := parser.ParseHTML(htmlContent)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use parsed data
    fmt.Printf("Method: %s\n", apiData.MethodInfo.Name)
    fmt.Printf("Title: %s\n", apiData.MethodInfo.Title)
    fmt.Printf("Request Parameters: %d\n", len(apiData.RequestParams))
    fmt.Printf("Response Parameters: %d\n", len(apiData.ResponseParams))
}
```

## 🧪 Testing

### Run Tests

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make test-coverage-html
```

### Test Coverage

Current test coverage:
- **Parse Package**: 87.1%
- **Generate Package**: 92.9%
- **Validate Package**: 79.4%
- **Batch Package**: 75.6%

The test suite includes:
- HTML parsing validation
- Method information extraction
- Request/response parameter parsing
- Error information extraction
- HTTP method determination
- Parameter row parsing
- JSON parsing and validation
- OpenAPI specification generation
- JSON schema validation
- YAML output generation
- Batch processing functionality
- Directory scanning and file handling

## 🐛 Debugging

### Using Delve Debugger

```bash
# Debug tests
make debug-test

# Debug CLI tool
make debug-parser

# Manual debugging
dlv test ./internal/parse -- -test.v
dlv debug main.go -- -help
```

### VS Code Integration

The project supports VS Code debugging:
- Debug Tests: `make debug-test`
- Debug Parser: `make debug-parser`
- Debug All Tests: Run tests with delve

## 📁 Project Structure

```
bin/api/
├── main.go                      # CLI tool (simplified structure)
├── internal/
│   ├── parse/
│   │   ├── parser.go           # Core parsing logic
│   │   └── parser_test.go      # Comprehensive tests
│   ├── generate/
│   │   ├── openapi.go          # OpenAPI generation
│   │   └── openapi_test.go     # OpenAPI tests
│   ├── validate/
│   │   ├── schema.go           # JSON schema validation
│   │   └── schema_test.go      # Validation tests
│   ├── models/
│   │   └── api.go              # Data structures
│   └── batch/
│       ├── processor.go        # Batch processing
│       ├── processor_test.go   # Batch tests
│       ├── scanner.go          # Directory scanning
│       └── scanner_test.go     # Scanner tests
├── openapi.yaml                 # Example OpenAPI output
├── Makefile                     # Build and test automation
├── go.mod                       # Go module definition
└── README.md                   # This file
```

## 🔧 Development

### Setup Development Environment

```bash
make dev-setup
```

This will:
- Install development tools (Delve, golangci-lint)
- Download dependencies
- Set up the development environment

### Available Make Targets

```bash
make help
```

Key targets:
- `test` - Run tests
- `build` - Build CLI tool
- `lint` - Run linter
- `fmt` - Format code
- `debug-test` - Debug tests
- `debug-parser` - Debug CLI tool
- `clean` - Clean build artifacts
- `ci` - Run CI/CD pipeline

### Code Quality

The project follows Go best practices:
- Comprehensive error handling
- Type safety with structs
- Extensive test coverage
- Clean code organization
- Proper documentation

## 📊 Output Format

The parser outputs structured JSON data:

```json
{
  "method_info": {
    "name": "start.simple_call",
    "title": "Start simple call",
    "description": "Звонок на любые номера кроме собственных виртуальных...",
    "http_method": "post"
  },
  "request_params": {
    "access_token": {
      "name": "access_token",
      "type": "string",
      "required": true,
      "description": "Ключ сессии аутентификации"
    },
    "first_call": {
      "name": "first_call",
      "type": "string",
      "required": true,
      "allowed_values": "contact, operator",
      "description": "Определяет номер, на который нужно дозвониться..."
    }
  },
  "response_params": {
    "call_session_id": {
      "name": "call_session_id",
      "type": "number",
      "required": true,
      "description": "Уникальный идентификатор сессии звонка"
    }
  },
  "error_info": {
    "errors": [
      {
        "code": "-32602",
        "mnemonic": "tts_text_exceeded",
        "description": "Длина сообщения превысила допустимое ограничение..."
      }
    ]
  }
}
```

## 🚀 Performance

- **Fast Execution**: Tests run in ~0.005s
- **Memory Efficient**: Uses goquery's efficient DOM parsing
- **Type Safe**: Compile-time error checking
- **Maintainable**: Clean, readable code structure
- **High Coverage**: 87.1% parse, 92.9% generate, 79.4% validate

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

### Development Guidelines

- Follow Go project rules in `.cursorrules`
- Maintain test coverage above 80%
- Use meaningful commit messages
- Document public APIs
- Handle errors properly

## 📝 License

This project is licensed under the MIT License.

## 🔗 Related Projects

- [goquery](https://github.com/PuerkitoBio/goquery) - HTML parsing library
- [gojsonschema](https://github.com/xeipuuv/gojsonschema) - JSON schema validation
- [go-yaml](https://github.com/goccy/go-yaml) - YAML processing library
- [Delve](https://github.com/go-delve/delve) - Go debugger
- [golangci-lint](https://github.com/golangci/golangci-lint) - Go linter

## 📞 Support

For issues and questions:
- Create an issue on GitHub
- Check the test suite for usage examples
- Review the debugging documentation

---

**Status**: ✅ Task 2 Complete - JSON parsing, OpenAPI generation, and schema validation with comprehensive testing
