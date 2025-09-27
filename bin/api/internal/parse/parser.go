package parse

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/xcono/novofon/bin/api/internal/models"
)

// Parser handles HTML parsing and data extraction
type Parser struct {
	doc *goquery.Document
}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Doc returns the current document for debugging
func (p *Parser) Doc() *goquery.Document {
	return p.doc
}

// ParseHTML parses HTML content and extracts API documentation data
func (p *Parser) ParseHTML(htmlContent string) (*models.APIData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	p.doc = doc

	apiData := &models.APIData{
		RequestParams:  make(map[string]*models.Parameter),
		ResponseParams: make(map[string]*models.Parameter),
	}

	// Extract method information
	methodInfo, err := p.ExtractMethodInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to extract method info: %w", err)
	}
	apiData.MethodInfo = methodInfo

	// Extract request parameters
	requestParams, err := p.ExtractRequestParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to extract request parameters: %w", err)
	}
	apiData.RequestParams = requestParams

	// Extract response parameters
	responseParams, err := p.ExtractResponseParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to extract response parameters: %w", err)
	}
	apiData.ResponseParams = responseParams

	// Extract JSON examples
	requestJSON, responseJSON, err := p.ExtractJSONExamples()
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON examples: %w", err)
	}
	apiData.RequestJSON = requestJSON
	apiData.ResponseJSON = responseJSON

	// Extract error information
	errorInfo, err := p.ExtractErrorInformation()
	if err != nil {
		return nil, fmt.Errorf("failed to extract error information: %w", err)
	}
	apiData.ErrorInfo = errorInfo

	return apiData, nil
}

// ExtractMethodInfo extracts basic method information from HTML
func (p *Parser) ExtractMethodInfo() (*models.MethodInfo, error) {
	methodInfo := &models.MethodInfo{}

	// Extract method name from table with 'Метод' header
	methodCell := p.doc.Find("th:contains('Метод')")
	if methodCell.Length() > 0 {
		parentRow := methodCell.Parent()
		nextCell := parentRow.Find("th").Next()
		code := nextCell.Find("code")
		if code.Length() > 0 {
			methodInfo.Name = strings.Trim(code.Text(), "\"'")
		}
	}

	if methodInfo.Name == "" {
		return nil, fmt.Errorf("method name not found")
	}

	// Extract title from h1
	title := p.doc.Find("h1").First()
	if title.Length() > 0 {
		methodInfo.Title = strings.TrimSpace(title.Text())
	}

	// Extract description from table
	descCell := p.doc.Find("td:contains('Описание')")
	if descCell.Length() > 0 {
		nextCell := descCell.Next()
		if nextCell.Length() > 0 {
			methodInfo.Description = strings.TrimSpace(nextCell.Text())
		}
	}

	// Determine HTTP method based on method name
	methodInfo.HTTPMethod = p.determineHTTPMethod(methodInfo.Name)

	return methodInfo, nil
}

// ExtractRequestParameters extracts request parameters from HTML tables
func (p *Parser) ExtractRequestParameters() (map[string]*models.Parameter, error) {
	params := make(map[string]*models.Parameter)

	// Find the "Параметры запроса" section
	requestHeader := p.doc.Find("h4:contains('Параметры запроса')")
	if requestHeader.Length() == 0 {
		return params, nil // No request parameters section found
	}

	// Find the table after this header
	table := requestHeader.Next()
	if table.Length() == 0 || !table.Is("table") {
		return params, nil
	}

	// Parse table rows (skip header row)
	table.Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // Skip header row
		}

		cells := s.Find("td")
		if cells.Length() >= 4 {
			param := p.parseParameterRow(cells, true)
			if param != nil {
				params[param.Name] = param
			}
		}
	})

	return params, nil
}

// ExtractResponseParameters extracts response parameters from HTML tables
func (p *Parser) ExtractResponseParameters() (map[string]*models.Parameter, error) {
	params := make(map[string]*models.Parameter)

	// Find the "Параметры ответа" section
	responseHeader := p.doc.Find("h4:contains('Параметры ответа')")
	if responseHeader.Length() == 0 {
		return params, nil // No response parameters section found
	}

	// Find the table after this header
	table := responseHeader.Next()
	if table.Length() == 0 || !table.Is("table") {
		return params, nil
	}

	// Parse table rows (skip header row)
	table.Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // Skip header row
		}

		cells := s.Find("td")
		if cells.Length() >= 3 {
			param := p.parseParameterRow(cells, false)
			if param != nil {
				params[param.Name] = param
			}
		}
	})

	return params, nil
}

// ExtractJSONExamples extracts JSON request and response examples
func (p *Parser) ExtractJSONExamples() (map[string]interface{}, map[string]interface{}, error) {
	var requestJSON, responseJSON map[string]interface{}

	// Find JSON request example
	requestHeader := p.doc.Find("h4:contains('Пример запроса')")
	if requestHeader.Length() > 0 {
		codeBlock := requestHeader.Next()
		if codeBlock.Is("pre") {
			code := codeBlock.Find("code")
			if code.Length() > 0 {
				jsonStr := code.Text()
				requestJSON = make(map[string]interface{})
				if err := json.Unmarshal([]byte(jsonStr), &requestJSON); err != nil {
					// If JSON parsing fails, return empty map but don't error
					requestJSON = make(map[string]interface{})
				}
			}
		}
	}

	// Find JSON response example
	responseHeader := p.doc.Find("h4:contains('Пример ответа')")
	if responseHeader.Length() > 0 {
		codeBlock := responseHeader.Next()
		if codeBlock.Is("pre") {
			code := codeBlock.Find("code")
			if code.Length() > 0 {
				jsonStr := code.Text()
				responseJSON = make(map[string]interface{})
				if err := json.Unmarshal([]byte(jsonStr), &responseJSON); err != nil {
					// If JSON parsing fails, return empty map but don't error
					responseJSON = make(map[string]interface{})
				}
			}
		}
	}

	return requestJSON, responseJSON, nil
}

// ExtractErrorInformation extracts error information from HTML tables
func (p *Parser) ExtractErrorInformation() (*models.ErrorInfo, error) {
	errorInfo := &models.ErrorInfo{
		Errors: make([]models.Error, 0),
	}

	// Find the error section
	errorHeader := p.doc.Find("h4:contains('Список возвращаемых ошибок')")
	if errorHeader.Length() == 0 {
		return errorInfo, nil // No error section found
	}

	// Find the table after this header
	table := errorHeader.Next()
	if table.Length() == 0 || !table.Is("table") {
		return errorInfo, nil
	}

	// Parse error rows (skip header row)
	table.Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // Skip header row
		}

		cells := s.Find("td")
		if cells.Length() >= 3 {
			error := models.Error{
				Code:        strings.TrimSpace(cells.Eq(1).Text()),
				Mnemonic:    strings.TrimSpace(cells.Eq(2).Text()),
				Description: strings.TrimSpace(cells.Eq(3).Text()),
			}
			errorInfo.Errors = append(errorInfo.Errors, error)
		}
	})

	return errorInfo, nil
}

// parseParameterRow parses a single parameter row from table cells
func (p *Parser) parseParameterRow(cells *goquery.Selection, isRequest bool) *models.Parameter {
	if cells.Length() < 3 {
		return nil
	}

	param := &models.Parameter{}

	// Extract parameter name from first cell
	nameCell := cells.Eq(0)
	nameCode := nameCell.Find("code")
	if nameCode.Length() > 0 {
		param.Name = strings.TrimSpace(nameCode.Text())
	} else {
		param.Name = strings.TrimSpace(nameCell.Text())
	}

	if param.Name == "" {
		return nil
	}

	// Extract type from second cell
	typeCell := cells.Eq(1)
	param.Type = strings.TrimSpace(typeCell.Text())

	// Extract required status
	if cells.Length() >= 3 {
		requiredCell := cells.Eq(2)
		requiredText := strings.ToLower(strings.TrimSpace(requiredCell.Text()))
		param.Required = requiredText == "да"
	}

	// Extract description and additional information
	if isRequest && cells.Length() >= 5 {
		// Request parameters: Name, Type, Required, Allowed Values, Description
		allowedValuesCell := cells.Eq(3)
		param.AllowedValues = strings.TrimSpace(allowedValuesCell.Text())

		descriptionCell := cells.Eq(4)
		param.Description = strings.TrimSpace(descriptionCell.Text())
	} else if !isRequest && cells.Length() >= 4 {
		// Response parameters: Name, Type, Required, Description
		descriptionCell := cells.Eq(3)
		param.Description = strings.TrimSpace(descriptionCell.Text())
	} else if cells.Length() >= 4 {
		// Fallback: assume description is in the last cell
		descriptionCell := cells.Eq(cells.Length() - 1)
		param.Description = strings.TrimSpace(descriptionCell.Text())
	}

	return param
}

// determineHTTPMethod determines HTTP method based on method name
func (p *Parser) determineHTTPMethod(methodName string) string {
	if strings.HasPrefix(methodName, "get.") {
		return "get"
	} else if strings.HasPrefix(methodName, "create.") {
		return "post"
	} else if strings.HasPrefix(methodName, "update.") {
		return "put"
	} else if strings.HasPrefix(methodName, "delete.") {
		return "delete"
	}
	return "post" // Default for JSON-RPC
}
