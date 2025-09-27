#!/usr/bin/env python3
"""
Enhanced HTML documentation parser for Novofon Data API.
Extracts comprehensive parameter information and generates detailed OpenAPI specifications.
"""

import os
import re
import json
import yaml
import logging
import argparse
from pathlib import Path
from typing import Dict, List, Any, Optional, Tuple
from bs4 import BeautifulSoup

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class NovofonAPIParser:
    """Enhanced parser for Novofon API documentation."""
    
    def __init__(self):
        self.supported_types = {
            'string': 'string',
            'number': 'number', 
            'boolean': 'boolean',
            'object': 'object',
            'array': 'array',
            'enum': 'string',
            'iso8601': 'string',
            'date': 'string',
            'datetime': 'string'
        }
    
    def extract_method_info(self, soup: BeautifulSoup) -> Optional[Dict[str, Any]]:
        """Extract comprehensive method information from HTML."""
        method_info = {
            'name': None,
            'title': None,
            'description': None,
            'access_level': None,
            'http_method': None
        }
        
        # Extract method name
        method_info['name'] = self._extract_method_name(soup)
        if not method_info['name']:
            return None
        
        # Extract title from h1
        method_info['title'] = self._extract_schema_title(soup)
        
        # Extract description
        method_info['description'] = self._extract_method_description(soup)
        
        # Extract access level (Кому доступен)
        method_info['access_level'] = self._extract_access_level(soup)
        
        # Determine HTTP method based on method name
        method_info['http_method'] = self._determine_http_method(method_info['name'])
        
        return method_info
    
    def _extract_method_name(self, soup: BeautifulSoup) -> Optional[str]:
        """Extract method name from HTML."""
        # Strategy 1: Look for method in table with 'Метод' header
        method_cell = soup.find('th', string='Метод')
        if method_cell:
            parent_row = method_cell.find_parent('tr')
            if parent_row:
                next_th = method_cell.find_next_sibling('th')
                if next_th:
                    code = next_th.find('code')
                    if code:
                        method_text = code.get_text().strip().strip('"\'')
                        return method_text
        
        # Strategy 2: Look for method in all tables
        tables = soup.find_all('table')
        for table in tables:
            rows = table.find_all('tr')
            for row in rows:
                cells = row.find_all(['th', 'td'])
                for cell in cells:
                    code = cell.find('code')
                    if code:
                        method_text = code.get_text().strip().strip('"\'')
                        if '.' in method_text and len(method_text.split('.')) == 2:
                            return method_text
        
        return None
    
    def _has_request_parameters_section(self, soup: BeautifulSoup) -> bool:
        """Check if the HTML contains a request parameters section."""
        # Look for "Параметры запроса" text in the document
        # This indicates it's a real API method, not an index/overview page
        text_content = soup.get_text()
        return "Параметры запроса" in text_content
    
    def _extract_method_description(self, soup: BeautifulSoup) -> Optional[str]:
        """Extract method description from HTML."""
        # Strategy 1: Look for description in table with 'Описание' header
        desc_cell = soup.find('th', string='Описание')
        if desc_cell:
            parent_row = desc_cell.find_parent('tr')
            if parent_row:
                next_cell = desc_cell.find_next_sibling('td')
                if next_cell:
                    return next_cell.get_text().strip()
        
        # Strategy 2: Look for nav element with description (find the one with actual description)
        navs = soup.find_all('nav')
        for nav in navs:
            nav_text = nav.get_text().strip()
            # Clean up nav text - remove extra whitespace and newlines
            nav_text = re.sub(r'\s+', ' ', nav_text).strip()
            
            # Look for nav that contains method description (breadcrumb style)
            if (nav_text and 
                nav_text not in ['Аккаунт', 'DATA API', 'Table of contents'] and
                not nav_text.startswith('DATA API') and
                not nav_text.startswith('Table of contents') and
                not nav_text.startswith('Previous') and
                not nav_text.startswith('Next') and
                not nav_text.startswith('Параметры метода') and
                '>' in nav_text and  # Breadcrumb navigation
                len(nav_text) < 200):  # Description should be reasonably short
                # Extract the last part after '>' (the actual method description)
                parts = nav_text.split('>')
                if len(parts) > 1:
                    return parts[-1].strip()
                return nav_text
        
        # Strategy 3: Look in title
        title = soup.find('h1')
        if title:
            return title.get_text().strip()
        
        return None
    
    def _extract_schema_title(self, soup: BeautifulSoup) -> Optional[str]:
        """Extract schema title from h1 element."""
        h1 = soup.find('h1')
        if h1:
            return h1.get_text().strip()
        return None
    
    def _extract_access_level(self, soup: BeautifulSoup) -> Optional[str]:
        """Extract access level information."""
        # Look for 'Кому доступен' in table (can be in th or td)
        access_cell = soup.find('th', string='Кому доступен') or soup.find('td', string='Кому доступен')
        if access_cell:
            parent_row = access_cell.find_parent('tr')
            if parent_row:
                # Find the next cell in the same row
                if access_cell.name == 'th':
                    next_cell = access_cell.find_next_sibling('td')
                else:
                    next_cell = access_cell.find_next_sibling('td')
                
                if next_cell:
                    return next_cell.get_text().strip()
        return None
    
    def _extract_allowed_values_from_cell(self, cell) -> Optional[str]:
        """Extract allowed values from a table cell, handling ul/li structure."""
        # Check for ul/li structure first
        ul = cell.find('ul')
        if ul:
            li_items = ul.find_all('li')
            if li_items:
                values = [li.get_text().strip() for li in li_items]
                return ', '.join(values)
        
        # Fallback to plain text
        text = cell.get_text().strip()
        return text if text else None
    
    def _determine_http_method(self, method_name: str) -> str:
        """Determine HTTP method based on method name."""
        if method_name.startswith('get.'):
            return 'get'
        elif method_name.startswith('create.'):
            return 'post'
        elif method_name.startswith('update.'):
            return 'put'
        elif method_name.startswith('delete.'):
            return 'delete'
        else:
            return 'post'  # Default for JSON-RPC
    
    def extract_request_parameters(self, soup: BeautifulSoup) -> Dict[str, Dict[str, Any]]:
        """Extract comprehensive request parameters from HTML."""
        parameters = {}
        
        # Find the "Параметры запроса" section (can be h3, h4, or h5)
        request_header = (soup.find('h3', string='Параметры запроса') or 
                         soup.find('h4', string='Параметры запроса') or 
                         soup.find('h5', string='Параметры запроса'))
        if not request_header:
            return parameters
        
        # Find the table after this header
        table = request_header.find_next('table')
        if not table:
            return parameters
        
        # Parse table rows
        rows = table.find_all('tr')[1:]  # Skip header row
        
        for row in rows:
            cells = row.find_all('td')
            if len(cells) >= 4:
                param_info = self._parse_parameter_row(cells)
                if param_info:
                    parameters[param_info['name']] = param_info
        
        return parameters
    
    def extract_response_parameters(self, soup: BeautifulSoup) -> Dict[str, Dict[str, Any]]:
        """Extract comprehensive response parameters from HTML."""
        parameters = {}
        
        # Find the "Параметры ответа" section (can be h3, h4, or h5)
        response_header = (soup.find('h3', string='Параметры ответа') or 
                          soup.find('h4', string='Параметры ответа') or 
                          soup.find('h5', string='Параметры ответа'))
        if not response_header:
            return parameters
        
        # Find the table after this header
        table = response_header.find_next('table')
        if not table:
            return parameters
        
        # Parse table rows
        rows = table.find_all('tr')[1:]  # Skip header row
        
        for row in rows:
            cells = row.find_all('td')
            if len(cells) >= 3:  # Response tables may have different structure
                param_info = self._parse_parameter_row(cells, is_response=True)
                if param_info:
                    parameters[param_info['name']] = param_info
        
        return parameters
    
    def _parse_parameter_row(self, cells: List, is_response: bool = False) -> Optional[Dict[str, Any]]:
        """Parse a single parameter row from table cells."""
        if len(cells) < 3:
            return None
        
        # Extract parameter name from first cell
        name_cell = cells[0]
        name_code = name_cell.find('code')
        if name_code:
            param_name = name_code.get_text().strip()
        else:
            # If no code tag, use the cell text directly
            param_name = name_cell.get_text().strip()
        
        if not param_name:
            return None
        
        # Extract type from second cell
        type_cell = cells[1]
        param_type = type_cell.get_text().strip()
        
        # Extract required status
        required = False
        if not is_response and len(cells) >= 3:
            required_cell = cells[2]
            required_text = required_cell.get_text().strip().lower()
            required = required_text == 'да'
        
        # Extract description and additional information
        description = ""
        additional_info = {}
        
        if is_response:
            # For response parameters, we have more columns
            if len(cells) >= 6:
                # Structure: Name, Type, Allowed Values, Filtering, Sorting, Description
                allowed_values = cells[2].get_text().strip()
                filtering = cells[3].get_text().strip()
                sorting = cells[4].get_text().strip()
                description = cells[5].get_text().strip()
                
                # Store additional information
                if allowed_values:
                    additional_info['allowed_values'] = allowed_values
                if filtering:
                    additional_info['filtering'] = filtering
                if sorting:
                    additional_info['sorting'] = sorting
            elif len(cells) >= 4:
                # Fallback: assume description is in the last cell
                description = cells[-1].get_text().strip()
        else:
            # For request parameters, check if we have "Допустимые значения" column
            if len(cells) >= 5:
                # Structure: Name, Type, Required, Allowed Values, Description
                allowed_values_cell = cells[3]
                description = cells[4].get_text().strip()
                
                # Extract allowed values - check for ul/li structure first
                allowed_values = self._extract_allowed_values_from_cell(allowed_values_cell)
                
                # Store allowed values if present
                if allowed_values:
                    additional_info['allowed_values'] = allowed_values
            elif len(cells) >= 4:
                # Fallback: assume description is in the 4th cell
                description = cells[3].get_text().strip()
        
        # Clean up description
        description = re.sub(r'\s+', ' ', description).strip()
        
        # Remove common unwanted text patterns
        unwanted_patterns = [
            r'Для получения списка пользователей клиента необходимо использовать метод "[^"]*"',
            r'Является обязательным для агента',
            r'Смотрим раздел "[^"]*"',
        ]
        
        for pattern in unwanted_patterns:
            description = re.sub(pattern, '', description, flags=re.IGNORECASE)
        
        # Clean up again after removing patterns
        description = re.sub(r'\s+', ' ', description).strip()
        
        return {
            'name': param_name,
            'type': param_type,
            'required': required,
            'description': description,
            'additional_info': additional_info
        }
    
    def extract_json_examples(self, soup: BeautifulSoup) -> Tuple[Optional[Dict], Optional[Dict]]:
        """Extract JSON request and response examples from HTML."""
        request_json = None
        response_json = None
        
        # Find JSON request example
        request_header = soup.find('h3', string='JSON структура запроса')
        if request_header:
            code_block = request_header.find_next('pre')
            if code_block:
                code = code_block.find('code')
                if code:
                    try:
                        request_json = json.loads(code.get_text().strip())
                    except json.JSONDecodeError:
                        pass
        
        # Find JSON response example
        response_header = soup.find('h3', string='JSON структура ответа')
        if response_header:
            code_block = response_header.find_next('pre')
            if code_block:
                code = code_block.find('code')
                if code:
                    try:
                        response_json = json.loads(code.get_text().strip())
                    except json.JSONDecodeError:
                        pass
        
        return request_json, response_json
    
    def extract_error_information(self, soup: BeautifulSoup) -> Dict[str, Any]:
        """Extract error information from HTML."""
        error_info = {
            'errors': [],
            'error_references': []
        }
        
        # Look for error sections
        error_headers = soup.find_all(['h3', 'h4'], string=lambda text: text and 'ошибк' in text.lower())
        
        # Also look for error sections with different text patterns
        error_headers.extend(soup.find_all(['h3', 'h4'], string=lambda text: text and 'возвращаемых ошибок' in text.lower()))
        
        for header in error_headers:
            # Find the next element after the header (could be paragraph, table, list, etc.)
            next_element = header.find_next(['p', 'table', 'ul', 'ol'])
            if next_element:
                if next_element.name == 'table':
                    # Extract errors from table
                    rows = next_element.find_all('tr')[1:]  # Skip header
                    for row in rows:
                        cells = row.find_all(['td', 'th'])
                        if len(cells) >= 2:
                            error_code = cells[0].get_text().strip()
                            error_description = cells[1].get_text().strip()
                            if error_code and error_description:
                                error_info['errors'].append({
                                    'code': error_code,
                                    'description': error_description
                                })
                elif next_element.name == 'p':
                    # Extract error references from paragraph
                    links = next_element.find_all('a')
                    for link in links:
                        text = link.get_text().strip()
                        href = link.get('href', '')
                        if text and href:
                            error_info['error_references'].append({
                                'text': text,
                                'href': href
                            })
                else:
                    # Extract errors from list
                    items = next_element.find_all('li')
                    for item in items:
                        text = item.get_text().strip()
                        if text:
                            error_info['errors'].append({
                                'description': text
                            })
        
        # Look for error references in links
        error_links = soup.find_all('a', href=lambda href: href and 'ошибк' in href.lower())
        for link in error_links:
            error_info['error_references'].append({
                'text': link.get_text().strip(),
                'href': link.get('href', '')
            })
        
        return error_info
    
    def _validate_extracted_data(self, method_info: Dict, request_params: Dict, response_params: Dict) -> bool:
        """Validate extracted data for consistency and completeness."""
        try:
            # Validate method info
            if not method_info.get('name'):
                logger.error("Method name is missing")
                return False
            
            if not method_info.get('http_method'):
                logger.error("HTTP method is missing")
                return False
            
            # Validate parameter names and types
            for param_name, param_info in request_params.items():
                if not param_name or not isinstance(param_name, str):
                    logger.error(f"Invalid parameter name: {param_name}")
                    return False
                
                if not param_info.get('type'):
                    logger.error(f"Parameter type missing for: {param_name}")
                    return False
                
                if not param_info.get('description'):
                    logger.warning(f"Parameter description missing for: {param_name}")
            
            for param_name, param_info in response_params.items():
                if not param_name or not isinstance(param_name, str):
                    logger.error(f"Invalid response parameter name: {param_name}")
                    return False
                
                if not param_info.get('type'):
                    logger.error(f"Response parameter type missing for: {param_name}")
                    return False
                
                if not param_info.get('description'):
                    logger.warning(f"Response parameter description missing for: {param_name}")
            
            return True
            
        except Exception as e:
            logger.error(f"Validation error: {e}")
            return False
    
    def parse_html_file(self, filepath: str) -> Optional[Dict[str, Any]]:
        """Parse a single HTML file and extract comprehensive API information."""
        try:
            logger.info(f"Parsing file: {filepath}")
            
            # Validate file exists and is readable
            if not os.path.exists(filepath):
                logger.error(f"File not found: {filepath}")
                return None
            
            if not os.access(filepath, os.R_OK):
                logger.error(f"File not readable: {filepath}")
                return None
            
            with open(filepath, 'r', encoding='utf-8') as f:
                content = f.read()
            
            if not content.strip():
                logger.warning(f"Empty file: {filepath}")
                return None
            
            soup = BeautifulSoup(content, 'html.parser')
            
            # Check if this is a real API method (contains "Параметры запроса")
            # Skip index/overview files that don't contain actual method parameters
            if not self._has_request_parameters_section(soup):
                logger.info(f"Skipping overview/index file (no request parameters): {filepath}")
                return None
            
            # Extract method information
            method_info = self.extract_method_info(soup)
            if not method_info:
                logger.warning(f"No method information found in: {filepath}")
                return None
            
            # Extract parameters
            request_params = self.extract_request_parameters(soup)
            response_params = self.extract_response_parameters(soup)
            
            # Extract JSON examples
            request_json, response_json = self.extract_json_examples(soup)
            
            # Extract error information
            error_info = self.extract_error_information(soup)
            
            # Validate extracted data
            if not self._validate_extracted_data(method_info, request_params, response_params):
                logger.warning(f"Validation failed for: {filepath}")
                return None
            
            logger.info(f"Successfully parsed: {filepath}")
            return {
                'method_info': method_info,
                'request_params': request_params,
                'response_params': response_params,
                'request_json': request_json,
                'response_json': response_json,
                'error_info': error_info,
                'filepath': filepath
            }
            
        except FileNotFoundError:
            logger.error(f"File not found: {filepath}")
            return None
        except PermissionError:
            logger.error(f"Permission denied: {filepath}")
            return None
        except UnicodeDecodeError as e:
            logger.error(f"Encoding error in {filepath}: {e}")
            return None
        except Exception as e:
            logger.error(f"Unexpected error parsing {filepath}: {e}")
            return None
    
    def generate_openapi_spec(self, api_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Generate comprehensive OpenAPI specification from parsed data."""
        try:
            logger.info("Generating OpenAPI specification")
            
            # Validate input data
            if not api_data:
                logger.error("No API data provided")
                return None
            
            method_info = api_data.get('method_info')
            request_params = api_data.get('request_params', {})
            response_params = api_data.get('response_params', {})
            error_info = api_data.get('error_info', {})
            
            if not method_info:
                logger.error("Method information is missing")
                return None
            
            method_name = method_info.get('name')
            http_method = method_info.get('http_method')
            
            if not method_name or not http_method:
                logger.error("Method name or HTTP method is missing")
                return None
            
            title = method_info.get('title') or f'Novofon Data API - {method_name}'
            description = method_info.get('description') or f'API endpoint for {method_name}'
            
            # Create OpenAPI spec
            spec = {
                'openapi': '3.0.0',
                'info': {
                    'title': title,
                    'version': '1.0.0',
                    'description': description
                },
                'paths': {
                    f'/{method_name}': {
                        http_method: {
                            'summary': title,
                            'description': self._generate_endpoint_description(method_info, request_params, response_params),
                            'responses': self._generate_responses(response_params, error_info)
                        }
                    }
                }
            }
            
            # Add requestBody only if it exists (not None)
            request_body = self._generate_request_body(request_params, method_name)
            if request_body is not None:
                spec['paths'][f'/{method_name}'][http_method]['requestBody'] = request_body
            
            # Add custom access field if available
            if method_info.get('access_level'):
                spec['x-access'] = method_info['access_level']
            
            # Add error information as custom field
            if error_info and (error_info.get('errors') or error_info.get('error_references')):
                spec['x-errors'] = error_info
            
            logger.info(f"Successfully generated OpenAPI spec for {method_name}")
            return spec
            
        except Exception as e:
            logger.error(f"Error generating OpenAPI spec: {e}")
            return None
    
    def _generate_endpoint_description(self, method_info: Dict, request_params: Dict, response_params: Dict) -> str:
        """Generate detailed endpoint description."""
        description_parts = []
        
        if method_info['description']:
            description_parts.append(method_info['description'])
        
        if method_info.get('access_level'):
            description_parts.append(f"**Доступ:** {method_info['access_level']}")
        
        if request_params:
            description_parts.append(f"**Параметры запроса:** {len(request_params)}")
            # Add detailed parameter information
            for name, param in request_params.items():
                required_mark = " (обязательный)" if param['required'] else " (опциональный)"
                description_parts.append(f"- `{name}` ({param['type']}){required_mark}: {param['description']}")
        
        if response_params:
            description_parts.append(f"**Параметры ответа:** {len(response_params)}")
            # Add detailed parameter information
            for name, param in response_params.items():
                description_parts.append(f"- `{name}` ({param['type']}): {param['description']}")
        
        return '\n\n'.join(description_parts)
    
    def _generate_request_body(self, request_params: Dict, method_name: str = None) -> Optional[Dict[str, Any]]:
        """Generate request body schema."""
        if not request_params:
            return None
        
        properties = {}
        required_fields = []
        
        for param_name, param_info in request_params.items():
            properties[param_name] = self._generate_parameter_schema(param_info)
            if param_info['required']:
                required_fields.append(param_name)
        
        return {
            'required': True,
            'content': {
                'application/json': {
                    'schema': {
                        'type': 'object',
                        'properties': {
                            'jsonrpc': {
                                'type': 'string',
                                'example': '2.0',
                                'description': 'JSON-RPC version'
                            },
                            'id': {
                                'type': 'number',
                                'description': 'Request identifier'
                            },
                            'method': {
                                'type': 'string',
                                'example': method_name or '',
                                'description': 'Method name'
                            },
                            'params': (lambda: {
                                'type': 'object',
                                'properties': properties,
                                **({'required': required_fields} if required_fields else {})
                            })()
                        },
                        'required': ['jsonrpc', 'id', 'method', 'params']
                    }
                }
            }
        }
    
    def _generate_responses(self, response_params: Dict, error_info: Dict = None) -> Dict[str, Any]:
        """Generate response schemas."""
        responses = {
            '200': {
                'description': 'Successful response',
                'content': {
                    'application/json': {
                        'schema': {
                            'type': 'object',
                            'properties': {
                                'jsonrpc': {
                                    'type': 'string',
                                    'example': '2.0',
                                    'description': 'JSON-RPC version'
                                },
                                'id': {
                                    'type': 'number',
                                    'description': 'Request identifier'
                                },
                                'result': {
                                    'type': 'object',
                                    'properties': {
                                        'data': self._generate_data_schema(response_params),
                                        'metadata': {
                                            'type': 'object',
                                            'description': 'Response metadata'
                                        }
                                    },
                                    'required': ['data', 'metadata']
                                }
                            },
                            'required': ['jsonrpc', 'id', 'result']
                        }
                    }
                }
            },
            '400': {
                'description': 'Bad Request',
                'content': {
                    'application/json': {
                        'schema': {
                            'type': 'object',
                            'properties': {
                                'jsonrpc': {'type': 'string'},
                                'id': {'type': 'number'},
                                'error': {
                                    'type': 'object',
                                    'properties': {
                                        'code': {'type': 'number'},
                                        'message': {'type': 'string'},
                                        'data': {'type': 'object'}
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        
        # Add specific error responses if available
        if error_info and error_info.get('errors'):
            for error in error_info['errors']:
                if 'code' in error:
                    error_code = str(error['code'])
                    # Validate that error code is a valid 3-digit HTTP status code
                    if self._is_valid_http_status_code(error_code):
                        responses[error_code] = {
                            'description': error.get('description', f'Error {error_code}'),
                            'content': {
                                'application/json': {
                                    'schema': {
                                        'type': 'object',
                                        'properties': {
                                            'jsonrpc': {'type': 'string'},
                                            'id': {'type': 'number'},
                                            'error': {
                                                'type': 'object',
                                                'properties': {
                                                    'code': {'type': 'number', 'example': int(error['code']) if error['code'].isdigit() else None},
                                                    'message': {'type': 'string', 'example': error.get('description', '')},
                                                    'data': {'type': 'object'}
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
        
        return responses
    
    def _generate_data_schema(self, response_params: Dict) -> Dict[str, Any]:
        """Generate data schema for response."""
        if not response_params:
            return {'type': 'object'}
        
        properties = {}
        required_fields = []
        
        for param_name, param_info in response_params.items():
            properties[param_name] = self._generate_parameter_schema(param_info)
            if param_info['required']:
                required_fields.append(param_name)
        
        schema = {
            'type': 'object',
            'properties': properties
        }
        
        # Only include required fields if there are any
        if required_fields:
            schema['required'] = required_fields
        
        return schema
    
    def _generate_parameter_schema(self, param_info: Dict[str, Any]) -> Dict[str, Any]:
        """Generate parameter schema with comprehensive information."""
        schema = {
            'type': self.supported_types.get(param_info['type'], 'string'),
            'description': param_info['description']
        }
        
        # Add additional information
        additional_info = param_info.get('additional_info', {})
        
        # Handle allowed values
        if 'allowed_values' in additional_info:
            allowed_values = additional_info['allowed_values']
            if allowed_values and allowed_values.strip():
                # Check if it's a format specification (like "Формат IANA zoneinfo")
                if 'формат' in allowed_values.lower() or 'format' in allowed_values.lower():
                    schema['format'] = allowed_values
                    schema['example'] = self._generate_example_for_format(allowed_values)
                # Check if it's a constraint (like "Максимум 255 символов")
                elif 'максимум' in allowed_values.lower() or 'максимальное' in allowed_values.lower():
                    # Extract numeric constraint
                    import re
                    numbers = re.findall(r'\d+', allowed_values)
                    if numbers:
                        if 'символ' in allowed_values.lower():
                            schema['maxLength'] = int(numbers[0])
                        elif 'количество' in allowed_values.lower():
                            schema['maxItems'] = int(numbers[0])
                # Check if it's a minimum constraint
                elif 'минимальное' in allowed_values.lower() or 'минимум' in allowed_values.lower():
                    import re
                    numbers = re.findall(r'\d+', allowed_values)
                    if numbers:
                        schema['minimum'] = int(numbers[0])
                else:
                    # Try to parse as enum values
                    enum_values = [v.strip() for v in allowed_values.split(',') if v.strip()]
                    if enum_values:
                        schema['enum'] = enum_values
                        schema['example'] = enum_values[0]
        
        # Add filtering information
        if 'filtering' in additional_info:
            filtering = additional_info['filtering']
            if filtering and filtering.strip():
                schema['x-filtering'] = filtering
        
        # Add sorting information
        if 'sorting' in additional_info:
            sorting = additional_info['sorting']
            if sorting and sorting.strip():
                schema['x-sorting'] = sorting
        
        # Add examples based on type if not already set
        if 'example' not in schema:
            if param_info['type'] == 'string':
                schema['example'] = 'example_string'
            elif param_info['type'] == 'number':
                schema['example'] = 123
            elif param_info['type'] == 'boolean':
                schema['example'] = True
        
        return schema
    
    def save_openapi_spec(self, spec: Dict[str, Any], filename: str) -> bool:
        """Safely save OpenAPI specification to file."""
        try:
            if not spec:
                logger.error("No OpenAPI spec to save")
                return False
            
            if not filename:
                logger.error("No filename provided")
                return False
            
            # Ensure directory exists
            os.makedirs(os.path.dirname(filename) if os.path.dirname(filename) else '.', exist_ok=True)
            
            # Write YAML file
            with open(filename, 'w', encoding='utf-8') as f:
                yaml.dump(spec, f, default_flow_style=False, allow_unicode=True, sort_keys=False)
            
            logger.info(f"OpenAPI spec saved to: {filename}")
            return True
            
        except PermissionError:
            logger.error(f"Permission denied writing to: {filename}")
            return False
        except OSError as e:
            logger.error(f"OS error saving to {filename}: {e}")
            return False
        except Exception as e:
            logger.error(f"Error saving OpenAPI spec to {filename}: {e}")
            return False
    
    def _is_valid_http_status_code(self, status_code: str) -> bool:
        """Validate that a status code is a valid 3-digit HTTP status code."""
        try:
            code = int(status_code)
            # HTTP status codes range from 100 to 599
            return 100 <= code <= 599
        except (ValueError, TypeError):
            return False
    
    def _generate_example_for_format(self, format_spec: str) -> str:
        """Generate example value based on format specification."""
        format_lower = format_spec.lower()
        
        if 'iana' in format_lower and 'zoneinfo' in format_lower:
            return 'Europe/Moscow'
        elif 'email' in format_lower:
            return 'user@example.com'
        elif 'phone' in format_lower:
            return '+7 (999) 123-45-67'
        elif 'url' in format_lower:
            return 'https://example.com'
        elif 'date' in format_lower:
            return '2024-01-01'
        elif 'time' in format_lower:
            return '12:00:00'
        elif 'datetime' in format_lower:
            return '2024-01-01T12:00:00Z'
        else:
            return 'example_value'
    
    def process_directory(self, directory: str, output_dir: str = "openapi_specs_enhanced") -> Dict[str, Any]:
        """Process all HTML files in a directory and generate OpenAPI specs."""
        results = {
            'processed': 0,
            'successful': 0,
            'failed': 0,
            'errors': []
        }
        
        try:
            if not os.path.exists(directory):
                logger.error(f"Directory not found: {directory}")
                results['errors'].append(f"Directory not found: {directory}")
                return results
            
            # Create output directory
            os.makedirs(output_dir, exist_ok=True)
            
            # Find all HTML files
            html_files = []
            for root, dirs, files in os.walk(directory):
                for file in files:
                    if file.endswith('.html') and file == 'index.html':
                        html_files.append(os.path.join(root, file))
            
            logger.info(f"Found {len(html_files)} HTML files to process")
            
            for filepath in html_files:
                results['processed'] += 1
                logger.info(f"Processing {results['processed']}/{len(html_files)}: {filepath}")
                
                try:
                    # Parse file
                    api_data = self.parse_html_file(filepath)
                    
                    if api_data:
                        # Generate OpenAPI spec
                        spec = self.generate_openapi_spec(api_data)
                        
                        if spec:
                            # Save spec
                            method_name = api_data['method_info']['name']
                            output_file = os.path.join(output_dir, f"{method_name}.yaml")
                            
                            if self.save_openapi_spec(spec, output_file):
                                results['successful'] += 1
                                logger.info(f"✓ Successfully processed: {method_name}")
                            else:
                                results['failed'] += 1
                                results['errors'].append(f"Failed to save: {method_name}")
                        else:
                            results['failed'] += 1
                            results['errors'].append(f"Failed to generate spec: {filepath}")
                    else:
                        results['failed'] += 1
                        results['errors'].append(f"Failed to parse: {filepath}")
                        
                except Exception as e:
                    results['failed'] += 1
                    results['errors'].append(f"Error processing {filepath}: {e}")
                    logger.error(f"Error processing {filepath}: {e}")
            
            logger.info(f"Processing complete: {results['successful']}/{results['processed']} successful")
            return results
            
        except Exception as e:
            logger.error(f"Error processing directory {directory}: {e}")
            results['errors'].append(f"Directory processing error: {e}")
            return results

def main():
    """Main function to run the enhanced parser."""
    # Parse command line arguments
    arg_parser = argparse.ArgumentParser(description='Enhanced HTML documentation parser for Novofon APIs')
    arg_parser.add_argument('--src', required=True, help='Source directory containing HTML documentation')
    arg_parser.add_argument('--dst', required=True, help='Destination directory for generated OpenAPI specs')
    arg_parser.add_argument('--api-type', choices=['data', 'calls'], required=True, help='Type of API to process')
    arg_parser.add_argument('--test', help='Test with a specific file')
    
    args = arg_parser.parse_args()
    
    # Initialize parser
    parser = NovofonAPIParser()
    
    # Test mode
    if args.test:
        if os.path.exists(args.test):
            print(f"Testing enhanced parser with: {args.test}")
            result = parser.parse_html_file(args.test)
            
            if result:
                print("✓ Successfully parsed file")
                print(f"Method: {result['method_info']['name']}")
                print(f"Title: {result['method_info']['title']}")
                print(f"Description: {result['method_info']['description']}")
                print(f"Access Level: {result['method_info']['access_level']}")
                print(f"Request Parameters: {len(result['request_params'])}")
                print(f"Response Parameters: {len(result['response_params'])}")
                
                # Show detailed parameter information
                print("\n=== REQUEST PARAMETERS ===")
                for name, param in result['request_params'].items():
                    print(f"  {name}: {param['type']} ({'required' if param['required'] else 'optional'})")
                    print(f"    Description: {param['description']}")
                
                print("\n=== RESPONSE PARAMETERS ===")
                for name, param in result['response_params'].items():
                    print(f"  {name}: {param['type']}")
                    print(f"    Description: {param['description']}")
                    if param['additional_info']:
                        print(f"    Additional info: {param['additional_info']}")
                
                # Show parameters with additional info
                print("\n=== PARAMETERS WITH ADDITIONAL INFO ===")
                for name, param in result['request_params'].items():
                    if param['additional_info']:
                        print(f"  {name}: {param['additional_info']}")
                for name, param in result['response_params'].items():
                    if param['additional_info']:
                        print(f"  {name}: {param['additional_info']}")
                
                # Show error information
                if result.get('error_info'):
                    print("\n=== ERROR INFORMATION ===")
                    error_info = result['error_info']
                    if error_info.get('errors'):
                        print("Errors:")
                        for error in error_info['errors']:
                            print(f"  {error}")
                    if error_info.get('error_references'):
                        print("Error references:")
                        for ref in error_info['error_references']:
                            print(f"  {ref}")
                
                # Generate OpenAPI spec
                openapi_spec = parser.generate_openapi_spec(result)
                
                if openapi_spec:
                    # Save to file
                    output_file = f"enhanced_openapi_{result['method_info']['name']}.yaml"
                    if parser.save_openapi_spec(openapi_spec, output_file):
                        print(f"✓ Generated OpenAPI spec: {output_file}")
                    else:
                        print(f"✗ Failed to save OpenAPI spec: {output_file}")
                else:
                    print("✗ Failed to generate OpenAPI spec")
            else:
                print("✗ Failed to parse file")
        else:
            print(f"Test file not found: {args.test}")
        return
    
    # Process directory
    api_name = "NOVOFON DATA API" if args.api_type == "data" else "NOVOFON CALLS API"
    print("\n" + "="*50)
    print(f"PROCESSING {api_name}")
    print("="*50)
    
    results = parser.process_directory(args.src, args.dst)
    print(f"\n{api_name} Results: {results['successful']}/{results['processed']} successful")
    if results['errors']:
        print(f"Errors: {len(results['errors'])}")
        for error in results['errors'][:5]:  # Show first 5 errors
            print(f"  - {error}")
    
    # Summary
    print("\n" + "="*50)
    print("SUMMARY")
    print("="*50)
    print(f"Total processed: {results['processed']}")
    print(f"Total successful: {results['successful']}")
    print(f"Total failed: {results['failed']}")
    print(f"Success rate: {(results['successful']/results['processed']*100):.1f}%" if results['processed'] > 0 else "N/A")

if __name__ == "__main__":
    main()
