package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xcono/novofon/bin/api/internal/generate"
	"github.com/xcono/novofon/bin/api/internal/models"
	"github.com/xcono/novofon/bin/api/internal/parse"
	"github.com/xcono/novofon/bin/api/internal/validate"
	"gopkg.in/yaml.v3"
)

// BatchProcessor handles processing multiple HTML files
type BatchProcessor struct {
	parser    *parse.Parser
	generator *generate.OpenAPIGenerator
	validator *validate.SchemaValidator
	options   *BatchOptions
}

// BatchOptions configures batch processing behavior
type BatchOptions struct {
	MaxWorkers      int           // Maximum number of concurrent workers
	OutputDir       string        // Output directory for results
	Format          string        // Output format: json, yaml, openapi
	Validate        bool          // Enable validation
	GenerateOpenAPI bool          // Generate OpenAPI specs
	GenerateReport  bool          // Generate processing report
	SkipErrors      bool          // Skip files with errors
	Verbose         bool          // Enable verbose output
	Timeout         time.Duration // Processing timeout per file
}

// BatchResult represents the result of processing a single file
type BatchResult struct {
	FilePath    string                     `json:"file_path"`
	Success     bool                       `json:"success"`
	Error       string                     `json:"error,omitempty"`
	APIData     *models.APIData            `json:"api_data,omitempty"`
	OpenAPISpec *generate.OpenAPISpec      `json:"openapi_spec,omitempty"`
	Validation  *validate.ValidationResult `json:"validation,omitempty"`
	ProcessTime time.Duration              `json:"process_time"`
}

// BatchReport represents the overall batch processing report
type BatchReport struct {
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	TotalFiles   int           `json:"total_files"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	SkippedCount int           `json:"skipped_count"`
	TotalTime    time.Duration `json:"total_time"`
	Results      []BatchResult `json:"results"`
	Summary      BatchSummary  `json:"summary"`
}

// BatchSummary provides summary statistics
type BatchSummary struct {
	APIMethods  []string `json:"api_methods"`
	ErrorTypes  []string `json:"error_types"`
	AverageTime float64  `json:"average_time_ms"`
	FastestFile string   `json:"fastest_file"`
	SlowestFile string   `json:"slowest_file"`
	TotalParams int      `json:"total_parameters"`
	TotalErrors int      `json:"total_errors"`
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(options *BatchOptions) *BatchProcessor {
	if options.MaxWorkers <= 0 {
		options.MaxWorkers = 4 // Default to 4 workers
	}
	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Second // Default 30 second timeout
	}

	return &BatchProcessor{
		parser:    parse.NewParser(),
		generator: generate.NewOpenAPIGenerator(),
		validator: validate.NewSchemaValidator(),
		options:   options,
	}
}

// ProcessDirectory processes all HTML files in a directory
func (bp *BatchProcessor) ProcessDirectory(ctx context.Context, dirPath string) (*BatchReport, error) {
	startTime := time.Now()

	// Find all HTML files
	htmlFiles, err := bp.findHTMLFiles(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find HTML files: %w", err)
	}

	if len(htmlFiles) == 0 {
		return &BatchReport{
			StartTime:    startTime,
			EndTime:      time.Now(),
			TotalFiles:   0,
			SuccessCount: 0,
			ErrorCount:   0,
			TotalTime:    time.Since(startTime),
			Results:      []BatchResult{},
			Summary:      BatchSummary{},
		}, nil
	}

	if bp.options.Verbose {
		fmt.Printf("Found %d HTML files to process\n", len(htmlFiles))
	}

	// Create output directory if needed
	if bp.options.OutputDir != "" {
		if err := os.MkdirAll(bp.options.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Process files
	results := bp.processFiles(ctx, htmlFiles)

	// Generate report
	report := bp.generateReport(startTime, results)

	// Save report if requested
	if bp.options.GenerateReport {
		if err := bp.saveReport(report); err != nil {
			return nil, fmt.Errorf("failed to save report: %w", err)
		}
	}

	return report, nil
}

// ProcessFiles processes a list of specific files
func (bp *BatchProcessor) ProcessFiles(ctx context.Context, filePaths []string) (*BatchReport, error) {
	startTime := time.Now()

	if len(filePaths) == 0 {
		return &BatchReport{
			StartTime:    startTime,
			EndTime:      time.Now(),
			TotalFiles:   0,
			SuccessCount: 0,
			ErrorCount:   0,
			TotalTime:    time.Since(startTime),
			Results:      []BatchResult{},
			Summary:      BatchSummary{},
		}, nil
	}

	if bp.options.Verbose {
		fmt.Printf("Processing %d files\n", len(filePaths))
	}

	// Create output directory if needed
	if bp.options.OutputDir != "" {
		if err := os.MkdirAll(bp.options.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Process files
	results := bp.processFiles(ctx, filePaths)

	// Generate report
	report := bp.generateReport(startTime, results)

	// Save report if requested
	if bp.options.GenerateReport {
		if err := bp.saveReport(report); err != nil {
			return nil, fmt.Errorf("failed to save report: %w", err)
		}
	}

	return report, nil
}

// findHTMLFiles recursively finds all HTML files in a directory
func (bp *BatchProcessor) findHTMLFiles(dirPath string) ([]string, error) {
	var htmlFiles []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-HTML files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".html") {
			return nil
		}

		// Skip root-level index.html files (but allow subdirectory index.html files)
		if strings.HasSuffix(strings.ToLower(info.Name()), "index.html") {
			// Only skip root-level index.html files
			dir := filepath.Dir(path)
			// If the directory is the same as the file path (minus the filename), it's root level
			if filepath.Dir(dir) == "." || filepath.Dir(dir) == "" {
				return nil
			}
		}

		htmlFiles = append(htmlFiles, path)
		return nil
	})

	return htmlFiles, err
}

// processFiles processes files concurrently
func (bp *BatchProcessor) processFiles(ctx context.Context, filePaths []string) []BatchResult {
	// Create channels for work distribution
	fileChan := make(chan string, len(filePaths))
	resultChan := make(chan BatchResult, len(filePaths))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < bp.options.MaxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			bp.worker(ctx, workerID, fileChan, resultChan)
		}(i)
	}

	// Send files to workers
	go func() {
		defer close(fileChan)
		for _, filePath := range filePaths {
			select {
			case fileChan <- filePath:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Close result channel when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []BatchResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// worker processes files from the input channel
func (bp *BatchProcessor) worker(ctx context.Context, workerID int, fileChan <-chan string, resultChan chan<- BatchResult) {
	for filePath := range fileChan {
		select {
		case <-ctx.Done():
			return
		default:
			result := bp.processFile(filePath)
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// processFile processes a single HTML file
func (bp *BatchProcessor) processFile(filePath string) BatchResult {
	startTime := time.Now()
	result := BatchResult{
		FilePath: filePath,
		Success:  false,
	}

	defer func() {
		result.ProcessTime = time.Since(startTime)
	}()

	// Read HTML file
	htmlContent, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result
	}

	// Parse HTML
	apiData, err := bp.parser.ParseHTML(string(htmlContent))
	if err != nil {
		if bp.options.SkipErrors {
			result.Error = fmt.Sprintf("parsing failed: %v", err)
			return result
		}
		result.Error = fmt.Sprintf("parsing failed: %v", err)
		return result
	}

	result.APIData = apiData
	result.Success = true

	// Generate OpenAPI spec if requested
	if bp.options.GenerateOpenAPI {
		spec, err := bp.generator.GenerateSpec(apiData)
		if err != nil {
			result.Error = fmt.Sprintf("OpenAPI generation failed: %v", err)
			if !bp.options.SkipErrors {
				return result
			}
		} else {
			result.OpenAPISpec = spec
		}
	}

	// Validate if requested
	if bp.options.Validate {
		validation, err := bp.validator.ValidateAPIData(apiData)
		if err != nil {
			result.Error = fmt.Sprintf("validation failed: %v", err)
			if !bp.options.SkipErrors {
				return result
			}
		} else {
			result.Validation = validation
		}
	}

	// Save individual file output if output directory is specified
	if bp.options.OutputDir != "" {
		if err := bp.saveFileOutput(result); err != nil {
			result.Error = fmt.Sprintf("failed to save output: %v", err)
			if !bp.options.SkipErrors {
				return result
			}
		}
	}

	return result
}

// saveFileOutput saves the output for a single file
func (bp *BatchProcessor) saveFileOutput(result BatchResult) error {
	if !result.Success || result.APIData == nil {
		return nil
	}

	// Generate output filename based on method name
	var outputFile string
	methodName := result.APIData.MethodInfo.Name
	if methodName == "" {
		// Fallback to file basename if method name is empty
		methodName = strings.TrimSuffix(filepath.Base(result.FilePath), ".html")
	}

	// Replace dots with underscores for valid filenames
	safeMethodName := strings.ReplaceAll(methodName, ".", "_")

	switch bp.options.Format {
	case "yaml":
		outputFile = filepath.Join(bp.options.OutputDir, safeMethodName+".yaml")
	case "openapi":
		outputFile = filepath.Join(bp.options.OutputDir, safeMethodName+".yaml")
	default:
		outputFile = filepath.Join(bp.options.OutputDir, safeMethodName+".json")
	}

	// Generate output data
	var outputData []byte
	var err error

	switch bp.options.Format {
	case "yaml":
		outputData, err = yaml.Marshal(result.APIData)
	case "openapi":
		if result.OpenAPISpec != nil {
			outputData, err = result.OpenAPISpec.ToYAML()
		} else {
			// Generate spec if not already generated
			spec, specErr := bp.generator.GenerateSpec(result.APIData)
			if specErr != nil {
				return specErr
			}
			outputData, err = spec.ToYAML()
		}
	default:
		outputData, err = json.MarshalIndent(result.APIData, "", "  ")
	}

	if err != nil {
		return err
	}

	// Write output file
	return os.WriteFile(outputFile, outputData, 0644)
}

// generateReport creates a comprehensive processing report
func (bp *BatchProcessor) generateReport(startTime time.Time, results []BatchResult) *BatchReport {
	endTime := time.Now()

	report := &BatchReport{
		StartTime:  startTime,
		EndTime:    endTime,
		TotalFiles: len(results),
		TotalTime:  endTime.Sub(startTime),
		Results:    results,
	}

	// Count successes and errors
	var totalParams, totalErrors int
	var apiMethods []string
	var errorTypes []string
	var fastestFile, slowestFile string
	var fastestTime, slowestTime time.Duration

	for _, result := range results {
		if result.Success {
			report.SuccessCount++
			if result.APIData != nil && result.APIData.MethodInfo != nil {
				apiMethods = append(apiMethods, result.APIData.MethodInfo.Name)
				totalParams += len(result.APIData.RequestParams) + len(result.APIData.ResponseParams)
				if result.APIData.ErrorInfo != nil {
					totalErrors += len(result.APIData.ErrorInfo.Errors)
				}
			}
		} else {
			report.ErrorCount++
			if result.Error != "" {
				errorTypes = append(errorTypes, result.Error)
			}
		}

		// Track fastest and slowest files
		if fastestFile == "" || result.ProcessTime < fastestTime {
			fastestFile = result.FilePath
			fastestTime = result.ProcessTime
		}
		if slowestFile == "" || result.ProcessTime > slowestTime {
			slowestFile = result.FilePath
			slowestTime = result.ProcessTime
		}
	}

	// Calculate average time
	var totalProcessTime time.Duration
	for _, result := range results {
		totalProcessTime += result.ProcessTime
	}

	report.Summary = BatchSummary{
		APIMethods:  apiMethods,
		ErrorTypes:  errorTypes,
		AverageTime: float64(totalProcessTime.Milliseconds()) / float64(len(results)),
		FastestFile: fastestFile,
		SlowestFile: slowestFile,
		TotalParams: totalParams,
		TotalErrors: totalErrors,
	}

	return report
}

// saveReport saves the processing report
func (bp *BatchProcessor) saveReport(report *BatchReport) error {
	if bp.options.OutputDir == "" {
		return nil
	}

	reportFile := filepath.Join(bp.options.OutputDir, "batch_report.json")
	reportData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(reportFile, reportData, 0644)
}
