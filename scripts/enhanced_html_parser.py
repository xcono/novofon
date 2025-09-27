#!/usr/bin/env python3
"""
Enhanced HTML Parser for Novofon API Documentation
Parses HTML documentation and generates OpenAPI specs and Markdown files
"""

import os
import re
import json
import yaml
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict
from bs4 import BeautifulSoup
import argparse
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

@dataclass
class Parameter:
    name: str
    type: str
    required: bool
    description: str
    allowed_values: Optional[str] = None

@dataclass
class APIEndpoint:
    method: str
    description: str
    available_to: str
    request_params: List[Parameter]
    response_params: List[Parameter]
    request_json: Optional[str] = None
    response_json: Optional[str] = None
    notes: Optional[str] = None

class NovofonHTMLParser:
    def __init__(self, base_path: str):
        self.base_path = Path(base_path)
        self.endpoints: Dict[str, APIEndpoint] = {}
        
    def parse_html_file(self, html_file: Path) -> Optional[APIEndpoint]:
        """Parse a single HTML file and extract API endpoint information"""
        try:
            with open(html_file, 'r', encoding='utf-8') as f:
                content = f.read()
            
            soup = BeautifulSoup(content, 'html.parser')
            
            # Extract method name from title or content
            method = self._extract_method(soup)
            if not method:
                return None
                
            # Extract description
            description = self._extract_description(soup)
            
            # Extract availability
            available_to = self._extract_availability(soup)
            
            # Extract request parameters
            request_params = self._extract_parameters(soup, "request")
            
            # Extract response parameters
            response_params = self._extract_parameters(soup, "response")
            
            # Extract JSON examples
            request_json = self._extract_json_example(soup, "request")
            response_json = self._extract_json_example(soup, "response")
            
            # Extract notes
            notes = self._extract_notes(soup)
            
            return APIEndpoint(
                method=method,
                description=description,
                available_to=available_to,
                request_params=request_params,
                response_params=response_params,
                request_json=request_json,
                response_json=response_json,
                notes=notes
            )
            
        except Exception as e:
            logger.error(f"Error parsing {html_file}: {e}")
            return None
    
    def _extract_method(self, soup: BeautifulSoup) -> Optional[str]:
        """Extract method name from HTML"""
        # Look for method in various places
        method_patterns = [
            r'<code>([^<]+)</code>',
            r'"([^"]+)"',
            r'method["\']?\s*:\s*["\']([^"\']+)["\']'
        ]
        
        text = soup.get_text()
        for pattern in method_patterns:
            matches = re.findall(pattern, text)
            for match in matches:
                if '.' in match and len(match) > 3:
                    return match
        
        # Try to extract from title
        title = soup.find('title')
        if title:
            title_text = title.get_text()
            if 'API' in title_text:
                # Extract method from title
                parts = title_text.split(' - ')
                if len(parts) > 1:
                    return parts[0].strip()
        
        return None
    
    def _extract_description(self, soup: BeautifulSoup) -> str:
        """Extract description from HTML"""
        # Look for description in table or text
        tables = soup.find_all('table')
        for table in tables:
            rows = table.find_all('tr')
            for row in rows:
                cells = row.find_all('td')
                if len(cells) >= 2:
                    if 'Описание' in cells[0].get_text():
                        return cells[1].get_text().strip()
        
        # Look for h1 or main heading
        h1 = soup.find('h1')
        if h1:
            return h1.get_text().strip()
        
        return "API endpoint"
    
    def _extract_availability(self, soup: BeautifulSoup) -> str:
        """Extract availability information"""
        text = soup.get_text()
        if 'Партнёр' in text and 'Клиент' in text:
            return 'Партнёр, Клиент'
        elif 'Партнёр' in text:
            return 'Партнёр'
        elif 'Клиент' in text:
            return 'Клиент'
        return 'Все'
    
    def _extract_parameters(self, soup: BeautifulSoup, param_type: str) -> List[Parameter]:
        """Extract parameters (request or response) from HTML"""
        parameters = []
        
        # Look for parameter tables
        tables = soup.find_all('table')
        for table in tables:
            headers = [th.get_text().strip() for th in table.find_all('th')]
            
            # Check if this is a parameter table
            if any(header in ['Название', 'Тип', 'Обязательный', 'Описание'] for header in headers):
                rows = table.find_all('tr')[1:]  # Skip header
                
                for row in rows:
                    cells = row.find_all('td')
                    if len(cells) >= 4:
                        name = cells[0].get_text().strip()
                        param_type_str = cells[1].get_text().strip()
                        required = cells[2].get_text().strip().lower() in ['да', 'yes', 'true']
                        description = cells[3].get_text().strip()
                        allowed_values = cells[4].get_text().strip() if len(cells) > 4 else None
                        
                        if name and name != 'Название':  # Skip header row
                            parameters.append(Parameter(
                                name=name,
                                type=param_type_str,
                                required=required,
                                description=description,
                                allowed_values=allowed_values
                            ))
        
        return parameters
    
    def _extract_json_example(self, soup: BeautifulSoup, example_type: str) -> Optional[str]:
        """Extract JSON example from HTML"""
        # Look for code blocks with JSON
        code_blocks = soup.find_all('pre')
        for block in code_blocks:
            code = block.find('code')
            if code and 'json' in code.get('class', []):
                return code.get_text().strip()
        
        # Look for JSON in text
        text = soup.get_text()
        json_pattern = r'\{[^{}]*"jsonrpc"[^{}]*\}'
        matches = re.findall(json_pattern, text, re.DOTALL)
        if matches:
            return matches[0]
        
        return None
    
    def _extract_notes(self, soup: BeautifulSoup) -> Optional[str]:
        """Extract notes or additional information"""
        # Look for blockquote or notes
        blockquote = soup.find('blockquote')
        if blockquote:
            return blockquote.get_text().strip()
        
        return None
    
    def parse_directory(self, directory: Path, api_type: str):
        """Parse all HTML files in a directory"""
        logger.info(f"Parsing {api_type} API directory: {directory}")
        
        for html_file in directory.rglob('*.html'):
            if html_file.name == 'index.html' and html_file.parent.name != directory.name:
                # Skip index.html files in subdirectories
                continue
                
            endpoint = self.parse_html_file(html_file)
            if endpoint:
                # Create a unique key for the endpoint
                key = f"{api_type}.{endpoint.method}"
                self.endpoints[key] = endpoint
                logger.info(f"Parsed endpoint: {key}")
    
    def generate_openapi_spec(self, endpoint: APIEndpoint, api_type: str) -> Dict[str, Any]:
        """Generate OpenAPI specification for an endpoint"""
        # Extract method and path from endpoint method
        method_parts = endpoint.method.split('.')
        if len(method_parts) >= 2:
            path = f"/{method_parts[0]}/{method_parts[1]}"
        else:
            path = f"/{endpoint.method}"
        
        # Build request schema
        request_schema = {
            "type": "object",
            "properties": {
                "jsonrpc": {"type": "string", "example": "2.0"},
                "id": {"type": "number"},
                "method": {"type": "string", "example": endpoint.method},
                "params": {
                    "type": "object",
                    "properties": {},
                    "required": []
                }
            },
            "required": ["jsonrpc", "id", "method", "params"]
        }
        
        # Add parameters to request schema
        for param in endpoint.request_params:
            param_schema = {"type": param.type}
            if param.description:
                param_schema["description"] = param.description
            if param.allowed_values:
                param_schema["enum"] = [v.strip() for v in param.allowed_values.split(',')]
            
            request_schema["properties"]["params"]["properties"][param.name] = param_schema
            if param.required:
                request_schema["properties"]["params"]["required"].append(param.name)
        
        # Build response schema
        response_schema = {
            "type": "object",
            "properties": {
                "jsonrpc": {"type": "string", "example": "2.0"},
                "id": {"type": "number"},
                "result": {
                    "type": "object",
                    "properties": {
                        "data": {
                            "type": "object",
                            "properties": {},
                            "required": []
                        }
                    }
                }
            }
        }
        
        # Add response parameters
        for param in endpoint.response_params:
            param_schema = {"type": param.type}
            if param.description:
                param_schema["description"] = param.description
            
            response_schema["properties"]["result"]["properties"]["data"]["properties"][param.name] = param_schema
            if param.required:
                response_schema["properties"]["result"]["properties"]["data"]["required"].append(param.name)
        
        return {
            "openapi": "3.0.0",
            "info": {
                "title": f"Novofon {api_type.title()} API - {endpoint.method}",
                "description": endpoint.description,
                "version": "1.0.0"
            },
            "servers": [
                {
                    "url": "https://api.novofon.com",
                    "description": "Novofon API Server"
                }
            ],
            "paths": {
                path: {
                    "post": {
                        "summary": f"{endpoint.method} endpoint",
                        "description": f"JSON-RPC 2.0 endpoint for {endpoint.method}",
                        "requestBody": {
                            "required": True,
                            "content": {
                                "application/json": {
                                    "schema": request_schema
                                }
                            }
                        },
                        "responses": {
                            "200": {
                                "description": "Successful response",
                                "content": {
                                    "application/json": {
                                        "schema": response_schema
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    
    def generate_markdown(self, endpoint: APIEndpoint, api_type: str) -> str:
        """Generate Markdown documentation for an endpoint"""
        md_content = f"# {endpoint.method}\n\n"
        md_content += f"**Описание:** {endpoint.description}\n\n"
        md_content += f"**Доступен для:** {endpoint.available_to}\n\n"
        
        if endpoint.request_params:
            md_content += "## Параметры запроса\n\n"
            md_content += "| Название | Тип | Обязательный | Описание |\n"
            md_content += "|----------|-----|--------------|----------|\n"
            
            for param in endpoint.request_params:
                required = "Да" if param.required else "Нет"
                md_content += f"| `{param.name}` | {param.type} | {required} | {param.description} |\n"
            md_content += "\n"
        
        if endpoint.response_params:
            md_content += "## Параметры ответа\n\n"
            md_content += "| Название | Тип | Обязательный | Описание |\n"
            md_content += "|----------|-----|--------------|----------|\n"
            
            for param in endpoint.response_params:
                required = "Да" if param.required else "Нет"
                md_content += f"| `{param.name}` | {param.type} | {required} | {param.description} |\n"
            md_content += "\n"
        
        if endpoint.request_json:
            md_content += "## JSON структура запроса\n\n"
            md_content += "```json\n"
            md_content += endpoint.request_json
            md_content += "\n```\n\n"
        
        if endpoint.response_json:
            md_content += "## JSON структура ответа\n\n"
            md_content += "```json\n"
            md_content += endpoint.response_json
            md_content += "\n```\n\n"
        
        if endpoint.notes:
            md_content += "## Примечания\n\n"
            md_content += f"{endpoint.notes}\n\n"
        
        return md_content
    
    def save_outputs(self, output_dir: Path):
        """Save generated OpenAPI specs and Markdown files"""
        # Create output directories according to new structure
        openapi_dir = output_dir / "docs"  # OpenAPI specs go to docs/
        markdown_dir = output_dir / "openai"  # Markdown goes to openai/
        
        openapi_dir.mkdir(parents=True, exist_ok=True)
        markdown_dir.mkdir(parents=True, exist_ok=True)
        
        # Save each endpoint
        for key, endpoint in self.endpoints.items():
            api_type, method = key.split('.', 1)
            
            # Create API-specific directories
            api_openapi_dir = openapi_dir / api_type
            api_markdown_dir = markdown_dir / api_type
            
            api_openapi_dir.mkdir(parents=True, exist_ok=True)
            api_markdown_dir.mkdir(parents=True, exist_ok=True)
            
            # Save OpenAPI spec
            spec = self.generate_openapi_spec(endpoint, api_type)
            spec_file = api_openapi_dir / f"{method.replace('.', '_')}.yaml"
            with open(spec_file, 'w', encoding='utf-8') as f:
                yaml.dump(spec, f, default_flow_style=False, allow_unicode=True)
            
            # Save Markdown
            md_content = self.generate_markdown(endpoint, api_type)
            md_file = api_markdown_dir / f"{method.replace('.', '_')}.md"
            with open(md_file, 'w', encoding='utf-8') as f:
                f.write(md_content)
            
            logger.info(f"Saved: {spec_file} and {md_file}")

def main():
    parser = argparse.ArgumentParser(description='Parse Novofon HTML documentation')
    parser.add_argument('--input', '-i', required=True, help='Input directory containing HTML files')
    parser.add_argument('--output', '-o', required=True, help='Output directory for generated files')
    parser.add_argument('--api-type', '-t', required=True, choices=['data', 'calls'], help='API type to parse')
    
    args = parser.parse_args()
    
    input_dir = Path(args.input)
    output_dir = Path(args.output)
    
    if not input_dir.exists():
        logger.error(f"Input directory does not exist: {input_dir}")
        return 1
    
    # Create parser and parse
    parser_instance = NovofonHTMLParser(input_dir)
    parser_instance.parse_directory(input_dir, args.api_type)
    
    # Save outputs
    parser_instance.save_outputs(output_dir)
    
    logger.info(f"Successfully parsed {len(parser_instance.endpoints)} endpoints")
    return 0

if __name__ == '__main__':
    exit(main())
