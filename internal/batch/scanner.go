package batch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScannerOptions configures file scanning behavior
type ScannerOptions struct {
	Recursive    bool     // Scan subdirectories recursively
	IncludeDirs  []string // Directories to include (empty = all)
	ExcludeDirs  []string // Directories to exclude
	FilePatterns []string // File patterns to match (e.g., "*.html")
	MinDepth     int      // Minimum directory depth
	MaxDepth     int      // Maximum directory depth (0 = unlimited)
	SkipIndex    bool     // Skip index.html files
	SkipAssets   bool     // Skip asset directories (css, js, images, etc.)
}

// ScanResult represents the result of a directory scan
type ScanResult struct {
	TotalFiles   int      `json:"total_files"`
	TotalDirs    int      `json:"total_dirs"`
	HTMLFiles    []string `json:"html_files"`
	SkippedFiles []string `json:"skipped_files"`
	ErrorFiles   []string `json:"error_files"`
	ScanTime     string   `json:"scan_time"`
	Directories  []string `json:"directories"`
}

// DirectoryScanner scans directories for HTML files
type DirectoryScanner struct {
	options *ScannerOptions
}

// NewDirectoryScanner creates a new directory scanner
func NewDirectoryScanner(options *ScannerOptions) *DirectoryScanner {
	if options == nil {
		options = &ScannerOptions{
			Recursive:    true,
			FilePatterns: []string{"*.html"},
			SkipIndex:    true,
			SkipAssets:   true,
		}
	}

	return &DirectoryScanner{
		options: options,
	}
}

// ScanDirectory scans a directory for HTML files
func (ds *DirectoryScanner) ScanDirectory(rootPath string) (*ScanResult, error) {
	result := &ScanResult{
		HTMLFiles:    []string{},
		SkippedFiles: []string{},
		ErrorFiles:   []string{},
		Directories:  []string{},
	}

	// Check if root path exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", rootPath)
	}

	// Walk the directory
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.ErrorFiles = append(result.ErrorFiles, path)
			return nil // Continue processing other files
		}

		// Count directories
		if info.IsDir() {
			result.TotalDirs++
			result.Directories = append(result.Directories, path)

			// Check if we should skip this directory
			if ds.shouldSkipDirectory(path, info) {
				return filepath.SkipDir
			}

			return nil
		}

		// Count files
		result.TotalFiles++

		// Check if this is an HTML file
		if ds.isHTMLFile(path, info) {
			// Check if we should skip this file
			if ds.shouldSkipFile(path, info) {
				result.SkippedFiles = append(result.SkippedFiles, path)
				return nil
			}

			result.HTMLFiles = append(result.HTMLFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	return result, nil
}

// isHTMLFile checks if a file is an HTML file
func (ds *DirectoryScanner) isHTMLFile(path string, info os.FileInfo) bool {
	// Check file extension
	if !strings.HasSuffix(strings.ToLower(path), ".html") {
		return false
	}

	// Check file patterns if specified
	if len(ds.options.FilePatterns) > 0 {
		matched := false
		for _, pattern := range ds.options.FilePatterns {
			if matched, _ := filepath.Match(pattern, info.Name()); matched {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// shouldSkipFile determines if a file should be skipped
func (ds *DirectoryScanner) shouldSkipFile(path string, info os.FileInfo) bool {
	fileName := strings.ToLower(info.Name())

	// Skip index files if requested (but allow them in subdirectories)
	if ds.options.SkipIndex && fileName == "index.html" {
		// Only skip root-level index.html files
		// Check if the file is directly in the root directory being scanned
		dir := filepath.Dir(path)
		// If the directory is the same as the file path (minus the filename), it's root level
		if filepath.Dir(dir) == "." || filepath.Dir(dir) == "" {
			return true
		}
	}

	// Skip 404 files
	if fileName == "404.html" {
		return true
	}

	// Skip files in asset directories
	if ds.options.SkipAssets {
		dir := filepath.Dir(path)
		dirName := strings.ToLower(filepath.Base(dir))
		assetDirs := []string{"assets", "css", "js", "javascripts", "stylesheets", "images", "img"}
		for _, assetDir := range assetDirs {
			if dirName == assetDir {
				return true
			}
		}
	}

	return false
}

// shouldSkipDirectory determines if a directory should be skipped
func (ds *DirectoryScanner) shouldSkipDirectory(path string, info os.FileInfo) bool {
	dirName := strings.ToLower(info.Name())

	// Skip hidden directories
	if strings.HasPrefix(dirName, ".") {
		return true
	}

	// Skip asset directories if requested
	if ds.options.SkipAssets {
		assetDirs := []string{"assets", "css", "js", "javascripts", "stylesheets", "images", "img", "node_modules", "vendor"}
		for _, assetDir := range assetDirs {
			if dirName == assetDir {
				return true
			}
		}
	}

	// Check exclude directories
	for _, excludeDir := range ds.options.ExcludeDirs {
		if strings.Contains(path, excludeDir) {
			return true
		}
	}

	// Check include directories (if specified)
	if len(ds.options.IncludeDirs) > 0 {
		included := false
		for _, includeDir := range ds.options.IncludeDirs {
			if strings.Contains(path, includeDir) {
				included = true
				break
			}
		}
		if !included {
			return true
		}
	}

	// Check depth limits
	if ds.options.MaxDepth > 0 {
		depth := ds.getDepth(path)
		if depth > ds.options.MaxDepth {
			return true
		}
	}

	if ds.options.MinDepth > 0 {
		depth := ds.getDepth(path)
		if depth < ds.options.MinDepth {
			return true
		}
	}

	return false
}

// getDepth calculates the directory depth
func (ds *DirectoryScanner) getDepth(path string) int {
	parts := strings.Split(path, string(filepath.Separator))
	depth := 0
	for _, part := range parts {
		if part != "" && part != "." {
			depth++
		}
	}
	return depth
}

// GetAPICategories scans and categorizes HTML files by API type
func (ds *DirectoryScanner) GetAPICategories(rootPath string) (map[string][]string, error) {
	result, err := ds.ScanDirectory(rootPath)
	if err != nil {
		return nil, err
	}

	categories := make(map[string][]string)

	for _, filePath := range result.HTMLFiles {
		// Extract category from path
		category := ds.extractCategory(filePath, rootPath)
		categories[category] = append(categories[category], filePath)
	}

	return categories, nil
}

// extractCategory extracts the API category from a file path
func (ds *DirectoryScanner) extractCategory(filePath, rootPath string) string {
	// Remove root path
	relPath, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		return "unknown"
	}

	// Split path into parts
	parts := strings.Split(relPath, string(filepath.Separator))

	// Find the first meaningful directory (skip empty parts)
	for _, part := range parts {
		if part != "" && part != "." {
			// Clean up the category name
			category := strings.ReplaceAll(part, "_", " ")
			category = strings.Title(category)
			return category
		}
	}

	return "root"
}

// GetFileStats provides statistics about the scanned files
func (ds *DirectoryScanner) GetFileStats(rootPath string) (*FileStats, error) {
	result, err := ds.ScanDirectory(rootPath)
	if err != nil {
		return nil, err
	}

	stats := &FileStats{
		TotalFiles:   result.TotalFiles,
		TotalDirs:    result.TotalDirs,
		HTMLFiles:    len(result.HTMLFiles),
		SkippedFiles: len(result.SkippedFiles),
		ErrorFiles:   len(result.ErrorFiles),
		Categories:   make(map[string]int),
		LargestFile:  "",
		SmallestFile: "",
		LargestSize:  0,
		SmallestSize: 0,
	}

	// Analyze file sizes and categorize
	for _, filePath := range result.HTMLFiles {
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		size := info.Size()

		// Track largest and smallest files
		if stats.LargestFile == "" || size > stats.LargestSize {
			stats.LargestFile = filePath
			stats.LargestSize = size
		}
		if stats.SmallestFile == "" || size < stats.SmallestSize {
			stats.SmallestFile = filePath
			stats.SmallestSize = size
		}

		// Categorize by directory
		category := ds.extractCategory(filePath, rootPath)
		stats.Categories[category]++
	}

	return stats, nil
}

// FileStats provides file statistics
type FileStats struct {
	TotalFiles   int            `json:"total_files"`
	TotalDirs    int            `json:"total_dirs"`
	HTMLFiles    int            `json:"html_files"`
	SkippedFiles int            `json:"skipped_files"`
	ErrorFiles   int            `json:"error_files"`
	Categories   map[string]int `json:"categories"`
	LargestFile  string         `json:"largest_file"`
	SmallestFile string         `json:"smallest_file"`
	LargestSize  int64          `json:"largest_size"`
	SmallestSize int64          `json:"smallest_size"`
}
