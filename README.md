# Novofon Documentation Generator

This set of scripts automatically generates documentation and OpenAPI specifications from Novofon API HTML documentation.

## Components

### 1. `enhanced_html_parser.py`
HTML documentation parser that extracts API endpoint information and generates OpenAPI specifications.

**Functionality:**
- Parses HTML documentation files
- Extracts request and response parameters
- Generates OpenAPI 3.0 specifications
- Creates structured data about API endpoints

**Usage:**
```bash
python scripts/enhanced_html_parser.py \
  --input docs/novofon/data_api \
  --output . \
  --api-type data
```

### 2. `html_to_markdown_converter.py`
HTML to Markdown converter for creating readable documentation.

**Functionality:**
- Converts HTML to clean Markdown
- Preserves table structure
- Processes code blocks
- Removes navigation elements

**Usage:**
```bash
python scripts/html_to_markdown_converter.py \
  --input docs/novofon/data_api \
  --output docs \
  --api-type data
```

## GitHub Action

The `.github/workflows/novofon.yaml` workflow automatically:

1. **Clones documentation** from `novofon/novofon.github.io`
2. **Generates OpenAPI specs** for Data and Call API
3. **Converts HTML to Markdown**
4. **Creates file structure**
5. **Commits changes** to the repository

## Output File Structure

```
docs/
├── openapi/
│   ├── data/           # Markdown documentation for Data API
│   └── calls/          # Markdown documentation for Call API
└── spec/
    ├── data/           # OpenAPI specifications for Data API
    └── calls/          # OpenAPI specifications for Call API
```

## Installing Dependencies

```bash
pip install -r scripts/requirements.txt
```

## Local Run

```bash
# Generate OpenAPI specs
python scripts/enhanced_html_parser.py \
  --input docs/novofon/data_api \
  --output . \
  --api-type data

# Convert to Markdown
python scripts/html_to_markdown_converter.py \
  --input docs/novofon/data_api \
  --output docs \
  --api-type data
```

## Supported API Types

- `data` - Data API documentation
- `calls` - Call API documentation

## Parser Features

- **Automatic extraction** of parameters from HTML tables
- **JSON-RPC 2.0 format** support
- **Validation** of required parameters
- **Processing** of various data types
- **Preservation** of documentation structure and hierarchy
