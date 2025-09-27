#!/usr/bin/env python3
"""
HTML to Markdown Converter for Novofon API Documentation
Converts HTML documentation to clean Markdown format
"""

import os
import re
from pathlib import Path
from typing import Optional
from bs4 import BeautifulSoup
import argparse
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class HTMLToMarkdownConverter:
    def __init__(self):
        self.conversion_rules = {
            'h1': '# ',
            'h2': '## ',
            'h3': '### ',
            'h4': '#### ',
            'h5': '##### ',
            'h6': '###### ',
        }
    
    def convert_html_to_markdown(self, html_content: str) -> str:
        """Convert HTML content to Markdown"""
        soup = BeautifulSoup(html_content, 'html.parser')
        
        # Remove navigation and sidebar elements
        for element in soup.find_all(['nav', 'aside', 'header', 'footer']):
            element.decompose()
        
        # Remove script and style tags
        for element in soup.find_all(['script', 'style']):
            element.decompose()
        
        # Get main content
        main_content = soup.find('main') or soup.find('article') or soup.find('div', class_='md-content')
        if main_content:
            content = main_content
        else:
            content = soup
        
        # Convert to markdown
        markdown = self._convert_element(content)
        
        # Clean up markdown
        markdown = self._clean_markdown(markdown)
        
        return markdown
    
    def _convert_element(self, element) -> str:
        """Convert HTML element to Markdown"""
        if element.name is None:
            return str(element)
        
        if element.name in self.conversion_rules:
            return f"{self.conversion_rules[element.name]}{self._get_text_content(element)}\n\n"
        
        elif element.name == 'p':
            return f"{self._get_text_content(element)}\n\n"
        
        elif element.name == 'table':
            return self._convert_table(element)
        
        elif element.name == 'pre':
            code = element.find('code')
            if code:
                language = self._get_code_language(code)
                return f"```{language}\n{code.get_text()}\n```\n\n"
            return f"```\n{element.get_text()}\n```\n\n"
        
        elif element.name == 'code':
            return f"`{element.get_text()}`"
        
        elif element.name == 'strong' or element.name == 'b':
            return f"**{element.get_text()}**"
        
        elif element.name == 'em' or element.name == 'i':
            return f"*{element.get_text()}*"
        
        elif element.name == 'a':
            href = element.get('href', '')
            text = element.get_text()
            if href:
                return f"[{text}]({href})"
            return text
        
        elif element.name == 'ul':
            return self._convert_list(element, ordered=False)
        
        elif element.name == 'ol':
            return self._convert_list(element, ordered=True)
        
        elif element.name == 'li':
            return f"- {self._get_text_content(element)}\n"
        
        elif element.name == 'blockquote':
            lines = self._get_text_content(element).split('\n')
            quoted_lines = [f"> {line}" for line in lines if line.strip()]
            return '\n'.join(quoted_lines) + '\n\n'
        
        elif element.name == 'hr':
            return "---\n\n"
        
        else:
            # For other elements, just get the text content
            return self._get_text_content(element)
    
    def _get_text_content(self, element) -> str:
        """Get text content from element, converting child elements"""
        if element.name is None:
            return str(element)
        
        text_parts = []
        for child in element.children:
            if hasattr(child, 'name'):
                text_parts.append(self._convert_element(child))
            else:
                text_parts.append(str(child))
        
        return ''.join(text_parts).strip()
    
    def _convert_table(self, table) -> str:
        """Convert HTML table to Markdown table"""
        rows = table.find_all('tr')
        if not rows:
            return ""
        
        markdown_rows = []
        
        # Process header row
        header_row = rows[0]
        header_cells = header_row.find_all(['th', 'td'])
        if header_cells:
            header_text = [self._get_text_content(cell).strip() for cell in header_cells]
            markdown_rows.append('| ' + ' | '.join(header_text) + ' |')
            markdown_rows.append('| ' + ' | '.join(['---'] * len(header_text)) + ' |')
        
        # Process data rows
        for row in rows[1:]:
            cells = row.find_all(['td', 'th'])
            if cells:
                cell_text = [self._get_text_content(cell).strip() for cell in cells]
                markdown_rows.append('| ' + ' | '.join(cell_text) + ' |')
        
        return '\n'.join(markdown_rows) + '\n\n'
    
    def _convert_list(self, list_element, ordered: bool = False) -> str:
        """Convert HTML list to Markdown list"""
        items = list_element.find_all('li')
        markdown_items = []
        
        for i, item in enumerate(items):
            text = self._get_text_content(item).strip()
            if ordered:
                markdown_items.append(f"{i + 1}. {text}")
            else:
                markdown_items.append(f"- {text}")
        
        return '\n'.join(markdown_items) + '\n\n'
    
    def _get_code_language(self, code_element) -> str:
        """Determine code language from code element"""
        class_list = code_element.get('class', [])
        for cls in class_list:
            if cls.startswith('language-'):
                return cls.replace('language-', '')
        
        # Check if it looks like JSON
        text = code_element.get_text()
        if '{' in text and '}' in text and '"' in text:
            return 'json'
        
        return ''
    
    def _clean_markdown(self, markdown: str) -> str:
        """Clean up markdown formatting"""
        # Remove excessive newlines
        markdown = re.sub(r'\n{3,}', '\n\n', markdown)
        
        # Fix table formatting
        markdown = re.sub(r'\|\s*\n', '|\n', markdown)
        
        # Remove empty lines at start and end
        markdown = markdown.strip()
        
        return markdown
    
    def convert_file(self, html_file: Path, output_file: Path) -> bool:
        """Convert HTML file to Markdown file"""
        try:
            with open(html_file, 'r', encoding='utf-8') as f:
                html_content = f.read()
            
            markdown_content = self.convert_html_to_markdown(html_content)
            
            # Ensure output directory exists
            output_file.parent.mkdir(parents=True, exist_ok=True)
            
            with open(output_file, 'w', encoding='utf-8') as f:
                f.write(markdown_content)
            
            logger.info(f"Converted: {html_file} -> {output_file}")
            return True
            
        except Exception as e:
            logger.error(f"Error converting {html_file}: {e}")
            return False
    
    def convert_directory(self, input_dir: Path, output_dir: Path, api_type: str):
        """Convert all HTML files in directory to Markdown"""
        logger.info(f"Converting {api_type} API HTML files to Markdown")
        
        converted_count = 0
        total_count = 0
        
        for html_file in input_dir.rglob('*.html'):
            if html_file.name == 'index.html' and html_file.parent.name != input_dir.name:
                # Skip index.html files in subdirectories
                continue
            
            total_count += 1
            
            # Create output path according to new structure: openai/{api_type}/
            relative_path = html_file.relative_to(input_dir)
            output_path = output_dir / "openai" / api_type / relative_path.with_suffix('.md')
            
            if self.convert_file(html_file, output_path):
                converted_count += 1
        
        logger.info(f"Converted {converted_count}/{total_count} files")

def main():
    parser = argparse.ArgumentParser(description='Convert HTML documentation to Markdown')
    parser.add_argument('--input', '-i', required=True, help='Input directory containing HTML files')
    parser.add_argument('--output', '-o', required=True, help='Output directory for Markdown files')
    parser.add_argument('--api-type', '-t', required=True, choices=['data', 'calls'], help='API type to convert')
    
    args = parser.parse_args()
    
    input_dir = Path(args.input)
    output_dir = Path(args.output)
    
    if not input_dir.exists():
        logger.error(f"Input directory does not exist: {input_dir}")
        return 1
    
    # Create converter and convert
    converter = HTMLToMarkdownConverter()
    converter.convert_directory(input_dir, output_dir, args.api_type)
    
    logger.info("HTML to Markdown conversion completed")
    return 0

if __name__ == '__main__':
    exit(main())
