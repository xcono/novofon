package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xcono/novofon/bin/api/internal/generate"
	"github.com/xcono/novofon/bin/api/internal/parse"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input-dir> <output-dir>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  input-dir:  Directory containing HTML files to parse\n")
		fmt.Fprintf(os.Stderr, "  output-dir: Directory to write OpenAPI YAML files\n")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Find all HTML files
	htmlFiles, err := findHTMLFiles(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding HTML files: %v\n", err)
		os.Exit(1)
	}

	parser := parse.NewParser()
	generator := generate.NewOpenAPIGenerator()

	processed := 0
	errors := 0

	for _, htmlFile := range htmlFiles {
		fmt.Printf("Processing: %s\n", htmlFile)

		// Read HTML file
		htmlContent, err := os.ReadFile(htmlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		// Parse HTML
		apiData, err := parser.ParseHTML(string(htmlContent))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		// Generate OpenAPI spec
		openAPISpec, err := generator.GenerateSpec(apiData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating OpenAPI spec for %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		// Write output file
		outputFile := getOutputFileName(htmlFile, outputDir)
		yamlContent, err := openAPISpec.ToYAML()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to YAML for %s: %v\n", htmlFile, err)
			errors++
			continue
		}

		if err := os.WriteFile(outputFile, []byte(yamlContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputFile, err)
			errors++
			continue
		}

		fmt.Printf("Generated: %s\n", outputFile)
		processed++
	}

	fmt.Printf("\nSummary: %d files processed, %d errors\n", processed, errors)

	if errors > 0 {
		os.Exit(1)
	}
}

func findHTMLFiles(dir string) ([]string, error) {
	var htmlFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip root-level index.html files (but allow subdirectory index.html files)
		if strings.ToLower(info.Name()) == "index.html" {
			// Only skip if this is a root-level index.html
			parentDir := filepath.Base(filepath.Dir(path))
			if parentDir == "." || parentDir == "" {
				return nil
			}
		}

		// Skip asset directories
		if strings.Contains(path, "/assets/") {
			return nil
		}

		// Include HTML files
		if strings.HasSuffix(strings.ToLower(info.Name()), ".html") {
			htmlFiles = append(htmlFiles, path)
		}

		return nil
	})

	return htmlFiles, err
}

func getOutputFileName(htmlFile, outputDir string) string {
	// Extract relative path from HTML file
	relPath := strings.TrimPrefix(htmlFile, filepath.Dir(filepath.Dir(htmlFile))+"/")

	// Convert path separators and remove .html extension
	fileName := strings.ReplaceAll(relPath, "/", ".")
	fileName = strings.TrimSuffix(fileName, ".html")

	// Ensure it ends with .yaml
	if !strings.HasSuffix(fileName, ".yaml") {
		fileName += ".yaml"
	}

	return filepath.Join(outputDir, fileName)
}
