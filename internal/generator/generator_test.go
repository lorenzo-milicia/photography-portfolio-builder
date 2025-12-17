package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.lorenzomilicia.dev/photography-portfolio-builder/assets"
)

// TestCaseResult holds the validation results for a test case
type TestCaseResult struct {
	Name        string
	OutputDir   string
	HasIndex    bool
	HasAbout    bool
	HasCSS      bool
	HasJS       bool
	ProjectDirs []string
	Errors      []string
}

// TestGenerateSites tests the site generation for all test cases in testdata
func TestGenerateSites(t *testing.T) {
	testdataDir := "testdata"

	// Discover all test case directories
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	var testCases []string
	for _, entry := range entries {
		if entry.IsDir() {
			testCases = append(testCases, entry.Name())
		}
	}

	if len(testCases) == 0 {
		t.Fatal("No test cases found in testdata directory")
	}

	t.Logf("Found %d test case(s): %v", len(testCases), testCases)

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			testSiteGeneration(t, testCase, testdataDir)
		})
	}
}

// testSiteGeneration tests the generation of a single test case
func testSiteGeneration(t *testing.T, testCase string, testdataDir string) {
	contentDir := filepath.Join(testdataDir, testCase)
	outputDir := filepath.Join(os.TempDir(), "generator-test-output", testCase)

	// Clean up any existing output
	if err := os.RemoveAll(outputDir); err != nil {
		t.Logf("Warning: Failed to clean up existing output: %v", err)
	}

	// Create generator
	gen := NewGenerator(contentDir, outputDir, assets.TemplatesFS, assets.StaticFS)

	// Generate the site
	err := gen.Generate("", "")
	if err != nil {
		t.Fatalf("Failed to generate site: %v", err)
	}

	// Validate the output
	result := validateGeneratedSite(t, outputDir)

	// Core validations that should pass for all test cases
	if !result.HasIndex {
		t.Error("Missing index.html in output")
	}

	if !result.HasAbout {
		t.Error("Missing about/index.html in output")
	}

	if !result.HasCSS {
		t.Error("Missing static/css/site.css in output")
	}

	if !result.HasJS {
		t.Error("Missing static/js/site.js in output")
	}

	// Log what was generated
	t.Logf("Generated site structure:")
	t.Logf("  - Index: %v", result.HasIndex)
	t.Logf("  - About: %v", result.HasAbout)
	t.Logf("  - CSS: %v", result.HasCSS)
	t.Logf("  - JS: %v", result.HasJS)
	t.Logf("  - Project directories: %d", len(result.ProjectDirs))

	if len(result.Errors) > 0 {
		t.Logf("Validation warnings:")
		for _, errMsg := range result.Errors {
			t.Logf("  - %s", errMsg)
		}
	}

	// Clean up after test
	defer func() {
		if err := os.RemoveAll(outputDir); err != nil {
			t.Logf("Warning: Failed to clean up output directory: %v", err)
		}
	}()
}

// validateGeneratedSite checks the generated output structure
func validateGeneratedSite(t *testing.T, outputDir string) TestCaseResult {
	result := TestCaseResult{
		OutputDir: outputDir,
		Errors:    make([]string, 0),
	}

	// Check for index.html
	indexPath := filepath.Join(outputDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		result.HasIndex = true
		// Validate it's not empty and contains basic HTML structure
		content, err := os.ReadFile(indexPath)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to read index.html")
		} else if len(content) == 0 {
			result.Errors = append(result.Errors, "index.html is empty")
		} else if !strings.Contains(string(content), "<!DOCTYPE html>") {
			result.Errors = append(result.Errors, "index.html missing DOCTYPE")
		}
	}

	// Check for about page
	aboutPath := filepath.Join(outputDir, "about", "index.html")
	if _, err := os.Stat(aboutPath); err == nil {
		result.HasAbout = true
		content, err := os.ReadFile(aboutPath)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to read about/index.html")
		} else if len(content) == 0 {
			result.Errors = append(result.Errors, "about/index.html is empty")
		}
	}

	// Check for CSS
	cssPath := filepath.Join(outputDir, "static", "css", "site.css")
	if _, err := os.Stat(cssPath); err == nil {
		result.HasCSS = true
	}

	// Check for JS
	jsPath := filepath.Join(outputDir, "static", "js", "site.js")
	if _, err := os.Stat(jsPath); err == nil {
		result.HasJS = true
	}

	// Find project directories
	entries, err := os.ReadDir(outputDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "static" && entry.Name() != "about" && entry.Name() != "favicon" {
				projectIndexPath := filepath.Join(outputDir, entry.Name(), "index.html")
				if _, err := os.Stat(projectIndexPath); err == nil {
					result.ProjectDirs = append(result.ProjectDirs, entry.Name())
				}
			}
		}
	}

	return result
}

// TestGenerateWithCustomTemplates tests generation with custom template overrides
func TestGenerateWithCustomTemplates(t *testing.T) {
	// Use basic_site as base
	contentDir := filepath.Join("testdata", "basic_site")
	outputDir := filepath.Join(os.TempDir(), "generator-test-custom-templates")

	// Clean up
	defer os.RemoveAll(outputDir)
	if err := os.RemoveAll(outputDir); err != nil {
		t.Logf("Warning: Failed to clean up existing output: %v", err)
	}

	// Create a custom templates directory
	customTemplatesDir := filepath.Join(os.TempDir(), "custom-templates-test")
	defer os.RemoveAll(customTemplatesDir)

	if err := os.MkdirAll(customTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create custom templates directory: %v", err)
	}

	// Create a custom footer override
	customFooter := `{{define "index-footer"}}
<footer class="custom-test-footer">
    <p>Custom Footer Test</p>
</footer>
{{end}}`

	customTemplatePath := filepath.Join(customTemplatesDir, "custom-footer.html")
	if err := os.WriteFile(customTemplatePath, []byte(customFooter), 0644); err != nil {
		t.Fatalf("Failed to write custom template: %v", err)
	}

	// Create generator with custom templates
	gen := NewGenerator(contentDir, outputDir, assets.TemplatesFS, assets.StaticFS)
	gen.SetTemplatesDir(customTemplatesDir)

	// Generate
	if err := gen.Generate("", ""); err != nil {
		t.Fatalf("Failed to generate site with custom templates: %v", err)
	}

	// Verify the custom template was applied
	indexPath := filepath.Join(outputDir, "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read generated index.html: %v", err)
	}

	if !strings.Contains(string(content), "custom-test-footer") {
		t.Error("Custom footer template was not applied")
	}

	if !strings.Contains(string(content), "Custom Footer Test") {
		t.Error("Custom footer content not found in output")
	}

	t.Log("Custom template override successfully applied")
}

// TestValidateLayoutLogic tests the grid layout validation
func TestValidateLayoutLogic(t *testing.T) {
	contentDir := filepath.Join("testdata", "with_grid")
	outputDir := filepath.Join(os.TempDir(), "generator-test-grid")

	defer os.RemoveAll(outputDir)
	if err := os.RemoveAll(outputDir); err != nil {
		t.Logf("Warning: Failed to clean up existing output: %v", err)
	}

	gen := NewGenerator(contentDir, outputDir, assets.TemplatesFS, assets.StaticFS)

	if err := gen.Generate("", ""); err != nil {
		t.Fatalf("Failed to generate site with grid layout: %v", err)
	}

	// Verify project page was generated
	projectPath := filepath.Join(outputDir, "grid-project", "index.html")
	content, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("Failed to read project page: %v", err)
	}

	contentStr := string(content)

	// Verify grid structure is present
	if !strings.Contains(contentStr, "gallery-grid") {
		t.Error("Gallery grid structure not found in project page")
	}

	// Verify grid width is configured
	if !strings.Contains(contentStr, "grid-template-columns: repeat(12, 1fr)") {
		t.Error("Grid width not properly configured")
	}

	// Verify mobile grid is present
	if !strings.Contains(contentStr, "grid-template-columns: repeat(6, 1fr)") {
		t.Error("Mobile grid not properly configured")
	}

	t.Log("Grid layout validation passed")
}

// TestGenerateWithImageURLPrefix tests generation with custom image URL prefix
func TestGenerateWithImageURLPrefix(t *testing.T) {
	contentDir := filepath.Join("testdata", "basic_site")
	outputDir := filepath.Join(os.TempDir(), "generator-test-cdn")

	defer os.RemoveAll(outputDir)
	if err := os.RemoveAll(outputDir); err != nil {
		t.Logf("Warning: Failed to clean up existing output: %v", err)
	}

	gen := NewGenerator(contentDir, outputDir, assets.TemplatesFS, assets.StaticFS)

	cdnURL := "https://cdn.example.com"
	if err := gen.Generate("", cdnURL); err != nil {
		t.Fatalf("Failed to generate site with CDN URL: %v", err)
	}

	// Verify the site was generated (basic check)
	indexPath := filepath.Join(outputDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index.html not generated: %v", err)
	}

	t.Log("Site generation with CDN URL completed successfully")
}

// TestDirectoryStructure validates the complete output directory structure
func TestDirectoryStructure(t *testing.T) {
	contentDir := filepath.Join("testdata", "basic_site")
	outputDir := filepath.Join(os.TempDir(), "generator-test-structure")

	defer os.RemoveAll(outputDir)
	if err := os.RemoveAll(outputDir); err != nil {
		t.Logf("Warning: Failed to clean up existing output: %v", err)
	}

	gen := NewGenerator(contentDir, outputDir, assets.TemplatesFS, assets.StaticFS)

	if err := gen.Generate("", ""); err != nil {
		t.Fatalf("Failed to generate site: %v", err)
	}

	// Required files and directories
	requiredPaths := []string{
		"index.html",
		"about/index.html",
		"static/css/site.css",
		"static/js/site.js",
		"sample-project/index.html",
	}

	for _, path := range requiredPaths {
		fullPath := filepath.Join(outputDir, path)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("Required path missing: %s", path)
		}
	}

	t.Log("Directory structure validation passed")
}
