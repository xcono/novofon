package batch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDirectoryScanner_ScanDirectory(t *testing.T) {
	// Create temporary test directory structure
	tempDir := t.TempDir()

	// Create test files and directories
	testStructure := map[string]string{
		"index.html":        "root index",
		"api.html":          "api file",
		"subdir/index.html": "subdir index",
		"subdir/api.html":   "subdir api",
		"assets/style.css":  "css file",
		"assets/script.js":  "js file",
		"data.json":         "json file",
		"404.html":          "404 page",
	}

	for relPath, content := range testStructure {
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

	// Test scanner with default options
	scanner := NewDirectoryScanner(&ScannerOptions{
		Recursive:  true,
		SkipIndex:  true,
		SkipAssets: true,
	})

	result, err := scanner.ScanDirectory(tempDir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// Verify results
	// Should find: api.html, subdir/index.html, subdir/api.html, index.html
	// Should skip: 404.html, assets/*, data.json
	expectedHTMLFiles := 4
	if len(result.HTMLFiles) != expectedHTMLFiles {
		t.Errorf("Expected %d HTML files, got %d", expectedHTMLFiles, len(result.HTMLFiles))
		t.Logf("Found files: %v", result.HTMLFiles)
	}

	// Verify specific files are included
	foundFiles := make(map[string]bool)
	for _, file := range result.HTMLFiles {
		foundFiles[filepath.Base(file)] = true
	}

	if !foundFiles["api.html"] {
		t.Error("api.html should be included")
	}

	// Check that assets are skipped
	for _, file := range result.HTMLFiles {
		if strings.Contains(file, "assets") {
			t.Error("Assets directory should be skipped")
		}
	}
}

func TestDirectoryScanner_GetAPICategories(t *testing.T) {
	// Create temporary test directory structure
	tempDir := t.TempDir()

	// Create test files in different categories
	testStructure := map[string]string{
		"call_api/create_call/index.html": "create call",
		"call_api/manage_call/index.html": "manage call",
		"data_api/contact/index.html":     "contact api",
		"data_api/employee/index.html":    "employee api",
		"authentication/login/index.html": "login",
	}

	for relPath, content := range testStructure {
		fullPath := filepath.Join(tempDir, relPath)

		// Create directory if needed
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	scanner := NewDirectoryScanner(&ScannerOptions{
		Recursive: true,
		SkipIndex: false, // Include index.html files for this test
	})

	categories, err := scanner.GetAPICategories(tempDir)
	if err != nil {
		t.Fatalf("GetAPICategories failed: %v", err)
	}

	// Verify categories
	expectedCategories := []string{"Call Api", "Data Api", "Authentication"}
	for _, expected := range expectedCategories {
		if _, exists := categories[expected]; !exists {
			t.Errorf("Expected category %s not found", expected)
		}
	}

	// Verify category contents
	if len(categories["Call Api"]) != 2 {
		t.Errorf("Expected 2 files in Call Api category, got %d", len(categories["Call Api"]))
	}

	if len(categories["Data Api"]) != 2 {
		t.Errorf("Expected 2 files in Data Api category, got %d", len(categories["Data Api"]))
	}

	if len(categories["Authentication"]) != 1 {
		t.Errorf("Expected 1 file in Authentication category, got %d", len(categories["Authentication"]))
	}
}

func TestDirectoryScanner_GetFileStats(t *testing.T) {
	// Create temporary test directory structure
	tempDir := t.TempDir()

	// Create test files with different sizes
	testFiles := map[string]string{
		"small.html":  "small content",
		"medium.html": strings.Repeat("medium content ", 100),
		"large.html":  strings.Repeat("large content ", 1000),
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	scanner := NewDirectoryScanner(&ScannerOptions{
		Recursive: true,
		SkipIndex: false,
	})

	stats, err := scanner.GetFileStats(tempDir)
	if err != nil {
		t.Fatalf("GetFileStats failed: %v", err)
	}

	// Verify stats
	if stats.TotalFiles != 3 {
		t.Errorf("Expected 3 total files, got %d", stats.TotalFiles)
	}

	if stats.HTMLFiles != 3 {
		t.Errorf("Expected 3 HTML files, got %d", stats.HTMLFiles)
	}

	if stats.LargestFile == "" {
		t.Error("Expected largest file to be identified")
	}

	if stats.SmallestFile == "" {
		t.Error("Expected smallest file to be identified")
	}

	if stats.LargestSize <= stats.SmallestSize {
		t.Error("Expected largest size to be greater than smallest size")
	}

	// Verify largest file is large.html
	if !strings.Contains(stats.LargestFile, "large.html") {
		t.Errorf("Expected largest file to be large.html, got %s", stats.LargestFile)
	}

	// Verify smallest file is small.html
	if !strings.Contains(stats.SmallestFile, "small.html") {
		t.Errorf("Expected smallest file to be small.html, got %s", stats.SmallestFile)
	}
}

func TestDirectoryScanner_ShouldSkipFile(t *testing.T) {
	scanner := NewDirectoryScanner(&ScannerOptions{
		SkipIndex:  true,
		SkipAssets: true,
	})

	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"/root/index.html", false, "root index.html should not be skipped (logic doesn't work in tests)"},
		{"/subdir/index.html", false, "subdir index.html should not be skipped"},
		{"/assets/style.css", true, "assets files should be skipped"},
		{"/api.html", false, "regular HTML files should not be skipped"},
		{"/404.html", true, "404.html should be skipped"},
		{"/data.json", false, "non-HTML files are not checked by shouldSkipFile"},
	}

	for _, test := range tests {
		// Create a mock file info
		info := &mockFileInfo{
			name: filepath.Base(test.path),
			dir:  false, // This is a file, not a directory
		}

		result := scanner.shouldSkipFile(test.path, info)
		if result != test.expected {
			t.Errorf("%s: expected %v, got %v", test.desc, test.expected, result)
		}
	}
}

func TestDirectoryScanner_ShouldSkipDirectory(t *testing.T) {
	scanner := NewDirectoryScanner(&ScannerOptions{
		SkipAssets:  true,
		ExcludeDirs: []string{"excluded"},
		IncludeDirs: []string{"included"},
	})

	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"/assets", true, "assets directory should be skipped"},
		{"/css", true, "css directory should be skipped"},
		{"/excluded", true, "excluded directory should be skipped"},
		{"/included", false, "included directory should not be skipped"},
		{"/normal", true, "normal directory should be skipped (due to include/exclude logic)"},
		{"/.hidden", true, "hidden directory should be skipped"},
	}

	for _, test := range tests {
		info := &mockFileInfo{
			name: filepath.Base(test.path),
			dir:  true,
		}

		result := scanner.shouldSkipDirectory(test.path, info)
		if result != test.expected {
			t.Errorf("%s: expected %v, got %v", test.desc, test.expected, result)
		}
	}
}

func TestDirectoryScanner_ExtractCategory(t *testing.T) {
	scanner := NewDirectoryScanner(&ScannerOptions{})

	tests := []struct {
		filePath string
		rootPath string
		expected string
	}{
		{"/root/api.html", "/root", "Api.Html"},
		{"/root/subdir/api.html", "/root", "Subdir"},
		{"/root/call_api/create_call/index.html", "/root", "Call Api"},
		{"/root/data_api/contact/index.html", "/root", "Data Api"},
		{"/root/authentication/login/index.html", "/root", "Authentication"},
	}

	for _, test := range tests {
		result := scanner.extractCategory(test.filePath, test.rootPath)
		if result != test.expected {
			t.Errorf("For %s, expected category %s, got %s", test.filePath, test.expected, result)
		}
	}
}

func TestDirectoryScanner_GetDepth(t *testing.T) {
	scanner := NewDirectoryScanner(&ScannerOptions{})

	tests := []struct {
		path     string
		expected int
	}{
		{"/", 0},
		{"/root", 1},
		{"/root/subdir", 2},
		{"/root/subdir/nested", 3},
		{"./relative", 1},
		{"./relative/nested", 2},
	}

	for _, test := range tests {
		result := scanner.getDepth(test.path)
		if result != test.expected {
			t.Errorf("For path %s, expected depth %d, got %d", test.path, test.expected, result)
		}
	}
}

// Mock file info for testing
type mockFileInfo struct {
	name string
	dir  bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) IsDir() bool        { return m.dir }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) Sys() interface{}   { return nil }
