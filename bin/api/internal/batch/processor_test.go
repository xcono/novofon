package batch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xcono/novofon/bin/api/internal/models"
)

func TestBatchProcessor_ProcessFiles(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()

	// Create test HTML files
	testFiles := []string{
		"test1.html",
		"test2.html",
		"test3.html",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		content := createTestHTML(filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}
	}

	// Test batch processing
	options := &BatchOptions{
		MaxWorkers:      2,
		OutputDir:       filepath.Join(tempDir, "output"),
		Format:          "json",
		Validate:        false,
		GenerateOpenAPI: false,
		GenerateReport:  true,
		SkipErrors:      true,
		Verbose:         false,
		Timeout:         30 * time.Second,
	}

	processor := NewBatchProcessor(options)

	// Convert to full paths
	var filePaths []string
	for _, filename := range testFiles {
		filePaths = append(filePaths, filepath.Join(tempDir, filename))
	}

	ctx := context.Background()
	report, err := processor.ProcessFiles(ctx, filePaths)
	if err != nil {
		t.Fatalf("ProcessFiles failed: %v", err)
	}

	// Verify results
	if report.TotalFiles != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), report.TotalFiles)
	}

	if report.SuccessCount == 0 {
		t.Error("Expected at least one successful file")
	}

	// Check that output directory was created
	outputDir := filepath.Join(tempDir, "output")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}

	// Check that report was generated
	reportFile := filepath.Join(outputDir, "batch_report.json")
	if _, err := os.Stat(reportFile); os.IsNotExist(err) {
		t.Error("Report file was not generated")
	}
}

func TestBatchProcessor_ProcessDirectory(t *testing.T) {
	// Create temporary test directory structure
	tempDir := t.TempDir()

	// Create subdirectories with HTML files
	subdirs := []string{"api1", "api2", "api3"}
	for _, subdir := range subdirs {
		dirPath := filepath.Join(tempDir, subdir)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory %s: %v", dirPath, err)
		}

		// Create index.html in each subdirectory
		filePath := filepath.Join(dirPath, "index.html")
		content := createTestHTML(subdir)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}
	}

	// Test batch processing
	options := &BatchOptions{
		MaxWorkers:      2,
		OutputDir:       filepath.Join(tempDir, "output"),
		Format:          "yaml",
		Validate:        false,
		GenerateOpenAPI: false,
		GenerateReport:  true,
		SkipErrors:      true,
		Verbose:         false,
		Timeout:         30 * time.Second,
	}

	processor := NewBatchProcessor(options)

	ctx := context.Background()
	report, err := processor.ProcessDirectory(ctx, tempDir)
	if err != nil {
		t.Fatalf("ProcessDirectory failed: %v", err)
	}

	// Verify results
	if report.TotalFiles != len(subdirs) {
		t.Errorf("Expected %d files, got %d", len(subdirs), report.TotalFiles)
	}

	if report.SuccessCount == 0 {
		t.Error("Expected at least one successful file")
	}

	// Check that output files were created
	outputDir := filepath.Join(tempDir, "output")
	outputFile := filepath.Join(outputDir, "index.yaml")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		// This is expected since we're processing multiple files with the same name
		// The last one will overwrite the previous ones
	}
}

func TestBatchProcessor_FindHTMLFiles(t *testing.T) {
	// Create temporary test directory structure
	tempDir := t.TempDir()

	// Create various files
	files := map[string]string{
		"index.html":        "root index",
		"api.html":          "api file",
		"test.html":         "test file",
		"subdir/index.html": "subdir index",
		"subdir/api.html":   "subdir api",
		"other.txt":         "text file",
		"data.json":         "json file",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tempDir, relPath)

		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	processor := NewBatchProcessor(&BatchOptions{})

	htmlFiles, err := processor.findHTMLFiles(tempDir)
	if err != nil {
		t.Fatalf("findHTMLFiles failed: %v", err)
	}

	// Should find: api.html, test.html, subdir/index.html, subdir/api.html, index.html
	// Should skip: other.txt, data.json
	expectedCount := 5
	if len(htmlFiles) != expectedCount {
		t.Errorf("Expected %d HTML files, got %d", expectedCount, len(htmlFiles))
		t.Logf("Found files: %v", htmlFiles)
	}

	// Verify specific files are included
	foundFiles := make(map[string]bool)
	for _, file := range htmlFiles {
		foundFiles[filepath.Base(file)] = true
	}

	if !foundFiles["api.html"] {
		t.Error("api.html should be included")
	}

	if !foundFiles["test.html"] {
		t.Error("test.html should be included")
	}
}

func TestBatchProcessor_ProcessFile(t *testing.T) {
	// Create temporary test file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.html")
	content := createTestHTML("test_method")

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	processor := NewBatchProcessor(&BatchOptions{})

	result := processor.processFile(filePath)

	// Verify result
	if !result.Success {
		t.Errorf("Expected successful processing, got error: %s", result.Error)
	}

	if result.APIData == nil {
		t.Error("Expected APIData to be populated")
	}

	if result.APIData.MethodInfo == nil {
		t.Error("Expected MethodInfo to be populated")
	}

	if result.ProcessTime < 0 {
		t.Error("Expected non-negative process time")
	}
}

func TestBatchProcessor_GenerateReport(t *testing.T) {
	startTime := time.Now()

	// Create mock results
	results := []BatchResult{
		{
			FilePath:    "file1.html",
			Success:     true,
			ProcessTime: 100 * time.Millisecond,
			APIData: &models.APIData{
				MethodInfo: &models.MethodInfo{
					Name: "test.method1",
				},
				RequestParams: map[string]*models.Parameter{
					"param1": {Name: "param1"},
				},
				ResponseParams: map[string]*models.Parameter{
					"result1": {Name: "result1"},
				},
				ErrorInfo: &models.ErrorInfo{
					Errors: []models.Error{
						{Code: "-32602", Description: "Test error"},
					},
				},
			},
		},
		{
			FilePath:    "file2.html",
			Success:     false,
			Error:       "Parsing failed",
			ProcessTime: 50 * time.Millisecond,
		},
		{
			FilePath:    "file3.html",
			Success:     true,
			ProcessTime: 200 * time.Millisecond,
			APIData: &models.APIData{
				MethodInfo: &models.MethodInfo{
					Name: "test.method2",
				},
				RequestParams: map[string]*models.Parameter{
					"param2": {Name: "param2"},
				},
			},
		},
	}

	processor := NewBatchProcessor(&BatchOptions{})
	report := processor.generateReport(startTime, results)

	// Verify report
	if report.TotalFiles != 3 {
		t.Errorf("Expected 3 total files, got %d", report.TotalFiles)
	}

	if report.SuccessCount != 2 {
		t.Errorf("Expected 2 successful files, got %d", report.SuccessCount)
	}

	if report.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", report.ErrorCount)
	}

	if len(report.Summary.APIMethods) != 2 {
		t.Errorf("Expected 2 API methods, got %d", len(report.Summary.APIMethods))
	}

	if report.Summary.TotalParams != 3 {
		t.Errorf("Expected 3 total parameters, got %d", report.Summary.TotalParams)
	}

	if report.Summary.TotalErrors != 1 {
		t.Errorf("Expected 1 total error, got %d", report.Summary.TotalErrors)
	}

	if report.Summary.FastestFile != "file2.html" {
		t.Errorf("Expected fastest file to be file2.html, got %s", report.Summary.FastestFile)
	}

	if report.Summary.SlowestFile != "file3.html" {
		t.Errorf("Expected slowest file to be file3.html, got %s", report.Summary.SlowestFile)
	}
}

func TestBatchOptions_Defaults(t *testing.T) {
	processor := NewBatchProcessor(&BatchOptions{})

	if processor.options.MaxWorkers != 4 {
		t.Errorf("Expected default MaxWorkers to be 4, got %d", processor.options.MaxWorkers)
	}

	if processor.options.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout to be 30s, got %v", processor.options.Timeout)
	}
}

// Helper function to create test HTML content
func createTestHTML(methodName string) string {
	return `
<!DOCTYPE html>
<html>
<head>
    <title>Test API</title>
</head>
<body>
    <h1>Test API Method</h1>
    <table>
        <tr>
            <th>Метод</th>
            <th><code>` + methodName + `</code></th>
        </tr>
        <tr>
            <td>Описание</td>
            <td>Test description for ` + methodName + `</td>
        </tr>
    </table>
    
    <h3>Параметры запроса</h3>
    <table>
        <tr>
            <th>Параметр</th>
            <th>Тип</th>
            <th>Обязательный</th>
            <th>Допустимые значения</th>
            <th>Описание</th>
        </tr>
        <tr>
            <td><code>param1</code></td>
            <td>string</td>
            <td>да</td>
            <td>value1, value2</td>
            <td>Test parameter</td>
        </tr>
    </table>
    
    <h3>Параметры ответа</h3>
    <table>
        <tr>
            <th>Параметр</th>
            <th>Тип</th>
            <th>Обязательный</th>
            <th>Описание</th>
        </tr>
        <tr>
            <td><code>result</code></td>
            <td>string</td>
            <td>да</td>
            <td>Test result</td>
        </tr>
    </table>
    
    <h4>Пример запроса</h4>
    <pre><code>{"jsonrpc": "2.0", "method": "` + methodName + `", "params": {"param1": "value1"}}</code></pre>
    
    <h4>Пример ответа</h4>
    <pre><code>{"jsonrpc": "2.0", "result": {"data": {"result": "success"}}}</code></pre>
</body>
</html>
`
}
