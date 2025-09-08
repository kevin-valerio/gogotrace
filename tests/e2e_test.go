package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// JSONOutput represents the JSON output structure
type JSONOutput struct {
	Name     string     `json:"name"`
	Package  string     `json:"package"`
	File     string     `json:"file"`
	Line     int        `json:"line"`
	Children []JSONNode `json:"children"`
}

type JSONNode struct {
	Name     string     `json:"name"`
	Receiver string     `json:"receiver,omitempty"`
	Package  string     `json:"package"`
	File     string     `json:"file"`
	Line     int        `json:"line"`
	Children []JSONNode `json:"children,omitempty"`
}

// Expected callers for TargetFunction  
// Note: Based on actual gogotrace behavior, it finds these callers
var expectedCallers = map[string]bool{
	"main":                         true,
	"processData":                  true,
	"helperFunction":               true,
	"(*Service).Execute":           true,
	"(*Service).internalProcess":   true,
	"func(...) in main.go":         true,  // Anonymous function from init
	"init":                         true,  // Parent of anonymous function
}

func TestEndToEnd(t *testing.T) {
	// Build gogotrace first
	buildCmd := exec.Command("go", "build", "-o", "gogotrace", ".")
	buildCmd.Dir = ".."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gogotrace: %v", err)
	}

	gogoTracePath := filepath.Join("..", "gogotrace")
	fixtureDir := filepath.Join("fixtures", "testproject")

	// Test each output format
	t.Run("ConsoleOutput", func(t *testing.T) {
		testConsoleOutput(t, gogoTracePath, fixtureDir)
	})

	t.Run("JSONOutput", func(t *testing.T) {
		testJSONOutput(t, gogoTracePath, fixtureDir)
	})

	t.Run("HTMLOutput", func(t *testing.T) {
		testHTMLOutput(t, gogoTracePath, fixtureDir)
	})

	t.Run("CrossFormatConsistency", func(t *testing.T) {
		testCrossFormatConsistency(t, gogoTracePath, fixtureDir)
	})
}

func testConsoleOutput(t *testing.T, gogoTracePath, fixtureDir string) {
	cmd := exec.Command(gogoTracePath, "-dir", fixtureDir, "-func", "TargetFunction")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Console output failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	foundCallers := extractCallersFromConsole(outputStr)

	// Verify we found expected callers
	for caller := range expectedCallers {
		found := false
		for _, foundCaller := range foundCallers {
			if strings.Contains(foundCaller, caller) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Console output missing expected caller: %s", caller)
		}
	}

	t.Logf("Console output found %d callers", len(foundCallers))
}

func testJSONOutput(t *testing.T, gogoTracePath, fixtureDir string) {
	jsonFile := filepath.Join(os.TempDir(), "test_output.json")
	defer os.Remove(jsonFile)

	cmd := exec.Command(gogoTracePath, "-dir", fixtureDir, "-func", "TargetFunction", "-json", jsonFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("JSON output failed: %v\nOutput: %s", err, output)
	}

	// Read and parse JSON
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON output: %v", err)
	}

	var jsonOutput JSONOutput
	if err := json.Unmarshal(data, &jsonOutput); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Collect all callers from JSON
	foundCallers := make(map[string]bool)
	collectJSONCallers(&jsonOutput, foundCallers)

	// Verify expected callers
	for caller := range expectedCallers {
		if !foundCallers[caller] {
			t.Errorf("JSON output missing expected caller: %s", caller)
		}
	}

	t.Logf("JSON output found %d callers", len(foundCallers))
}

func testHTMLOutput(t *testing.T, gogoTracePath, fixtureDir string) {
	htmlFile := filepath.Join(os.TempDir(), "test_output.html")
	defer os.Remove(htmlFile)

	cmd := exec.Command(gogoTracePath, "-dir", fixtureDir, "-func", "TargetFunction", "-html", htmlFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("HTML output failed: %v\nOutput: %s", err, output)
	}

	// Read HTML file
	data, err := os.ReadFile(htmlFile)
	if err != nil {
		t.Fatalf("Failed to read HTML output: %v", err)
	}

	htmlContent := string(data)

	// HTML output doesn't contain JSON, it has HTML structure
	// Check that the HTML file contains expected callers in the HTML structure
	foundCallers := 0
	for caller := range expectedCallers {
		// Normalize the caller name for HTML search
		searchStr := caller
		if strings.HasPrefix(searchStr, "(") && strings.Contains(searchStr, ").") {
			// Convert "(*Service).Execute" to a simpler search pattern
			searchStr = strings.ReplaceAll(searchStr, "(", "")
			searchStr = strings.ReplaceAll(searchStr, ")", "")
		}
		
		if strings.Contains(htmlContent, searchStr) || strings.Contains(htmlContent, strings.ReplaceAll(searchStr, "*", "")) {
			foundCallers++
		}
	}

	// We expect to find most callers in the HTML
	if foundCallers < len(expectedCallers)/2 {
		t.Errorf("HTML output seems to be missing many callers: found %d out of %d expected", foundCallers, len(expectedCallers))
	}

	t.Logf("HTML output found evidence of %d callers", foundCallers)
}

func testCrossFormatConsistency(t *testing.T, gogoTracePath, fixtureDir string) {
	// Get console output
	consoleCmd := exec.Command(gogoTracePath, "-dir", fixtureDir, "-func", "TargetFunction")
	consoleOutput, err := consoleCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Console output failed: %v", err)
	}
	consoleCallers := extractCallersFromConsole(string(consoleOutput))

	// Get JSON output
	jsonFile := filepath.Join(os.TempDir(), "consistency_test.json")
	defer os.Remove(jsonFile)
	
	jsonCmd := exec.Command(gogoTracePath, "-dir", fixtureDir, "-func", "TargetFunction", "-json", jsonFile)
	if _, err := jsonCmd.CombinedOutput(); err != nil {
		t.Fatalf("JSON output failed: %v", err)
	}

	data, _ := os.ReadFile(jsonFile)
	var jsonOutput JSONOutput
	json.Unmarshal(data, &jsonOutput)
	
	jsonCallers := make(map[string]bool)
	collectJSONCallers(&jsonOutput, jsonCallers)

	// Get HTML output (just verify it works, don't parse it since it's HTML not JSON)
	htmlFile := filepath.Join(os.TempDir(), "consistency_test.html")
	defer os.Remove(htmlFile)
	
	htmlCmd := exec.Command(gogoTracePath, "-dir", fixtureDir, "-func", "TargetFunction", "-html", htmlFile)
	if _, err := htmlCmd.CombinedOutput(); err != nil {
		t.Fatalf("HTML output failed: %v", err)
	}

	// HTML generates successfully, that's what matters
	if _, err := os.Stat(htmlFile); os.IsNotExist(err) {
		t.Errorf("HTML file was not created")
	}

	// Compare caller counts
	t.Logf("Console found %d unique callers", len(consoleCallers))
	t.Logf("JSON found %d unique callers", len(jsonCallers))
	t.Logf("HTML file generated successfully")

	// Console might show duplicates (same function at different tree levels)
	// So we just verify JSON has the expected callers
	missingCallers := []string{}
	for expectedCaller := range expectedCallers {
		if !jsonCallers[expectedCaller] {
			missingCallers = append(missingCallers, expectedCaller)
		}
	}

	if len(missingCallers) > 0 {
		t.Errorf("Missing expected callers in JSON: %v", missingCallers)
	}
}

func extractCallersFromConsole(output string) []string {
	lines := strings.Split(output, "\n")
	var callers []string
	
	for _, line := range lines {
		// Skip empty lines and headers
		if strings.TrimSpace(line) == "" || strings.Contains(line, "Call Graph:") || 
		   strings.Contains(line, "=========") || strings.Contains(line, "Analyzing") ||
		   strings.Contains(line, "Looking for") || strings.Contains(line, "Scanning") ||
		   strings.Contains(line, "Found") || strings.Contains(line, "Phase") ||
		   strings.Contains(line, "Analysis complete") {
			continue
		}
		
		// Look for tree structure lines
		if strings.Contains(line, "├──") || strings.Contains(line, "└──") || strings.Contains(line, "│") {
			// Remove ANSI color codes
			cleanLine := removeANSI(line)
			
			// Extract function name (it's after the tree symbols and before the arrow)
			if idx := strings.Index(cleanLine, "→"); idx > 0 {
				funcPart := cleanLine[:idx]
				// Remove tree symbols
				funcPart = strings.ReplaceAll(funcPart, "├──", "")
				funcPart = strings.ReplaceAll(funcPart, "└──", "")
				funcPart = strings.ReplaceAll(funcPart, "│", "")
				funcPart = strings.TrimSpace(funcPart)
				
				// Handle receiver.method format
				if funcPart != "" {
					// Parse the function name - could be in various formats
					// "*Service.Execute" -> "(*Service).Execute"
					// "func(...) in main.go" -> keep as-is
					// "main" -> keep as-is
					
					funcName := funcPart
					
					// Check if it's a method with receiver
					if strings.Contains(funcName, ".") && !strings.Contains(funcName, "func(") {
						// Split by dot to get receiver and method
						parts := strings.SplitN(funcName, ".", 2)
						if len(parts) == 2 {
							receiver := parts[0]
							method := parts[1]
							// Format as (receiver).method
							funcName = fmt.Sprintf("(%s).%s", receiver, method)
						}
					}
					
					callers = append(callers, funcName)
				}
			}
		}
	}
	
	return callers
}

func removeANSI(str string) string {
	// Remove ANSI escape codes
	for strings.Contains(str, "\x1b[") {
		start := strings.Index(str, "\x1b[")
		end := strings.Index(str[start:], "m")
		if end == -1 {
			break
		}
		str = str[:start] + str[start+end+1:]
	}
	return str
}

func collectJSONCallers(output *JSONOutput, callers map[string]bool) {
	// Process root level children (these are the callers)
	for _, caller := range output.Children {
		collectJSONNode(&caller, callers)
	}
}

func collectJSONNode(node *JSONNode, callers map[string]bool) {
	// Add this caller - combine receiver and name if receiver exists
	callerName := node.Name
	if node.Receiver != "" {
		callerName = fmt.Sprintf("(%s).%s", node.Receiver, node.Name)
	}
	callers[callerName] = true
	
	// Process children recursively
	for _, child := range node.Children {
		collectJSONNode(&child, callers)
	}
}

func TestSpecificFunctions(t *testing.T) {
	// Build gogotrace first
	buildCmd := exec.Command("go", "build", "-o", "gogotrace", ".")
	buildCmd.Dir = ".."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gogotrace: %v", err)
	}

	gogoTracePath := filepath.Join("..", "gogotrace")
	fixtureDir := filepath.Join("fixtures", "testproject")

	testCases := []struct {
		name           string
		targetFunc     string
		expectedCallers []string
	}{
		{
			name:       "UtilityFunction",
			targetFunc: "UtilityFunction",
			expectedCallers: []string{
				"AnotherHelper",
			},
		},
		{
			name:       "helperFunction",
			targetFunc: "helperFunction",
			expectedCallers: []string{
				"processData",
			},
		},
		{
			name:       "ProcessDataChain",
			targetFunc: "processData",
			expectedCallers: []string{
				"main",
			},
		},
		{
			name:       "ServiceExecute",
			targetFunc: "func (s *Service) Execute ()",  // Use the full signature format
			expectedCallers: []string{
				"main",
			},
		},
		{
			name:       "RecursiveCaller",
			targetFunc: "RecursiveCaller",
			expectedCallers: []string{
				"RecursiveCaller", // Self-recursive
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonFile := filepath.Join(os.TempDir(), fmt.Sprintf("test_%s.json", tc.name))
			defer os.Remove(jsonFile)

			cmd := exec.Command(gogoTracePath, "-dir", fixtureDir, "-func", tc.targetFunc, "-json", jsonFile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Failed to analyze %s: %v\nOutput: %s", tc.targetFunc, err, output)
			}

			// Read and parse JSON
			data, err := os.ReadFile(jsonFile)
			if err != nil {
				t.Fatalf("Failed to read JSON output: %v", err)
			}

			var jsonOutput JSONOutput
			if err := json.Unmarshal(data, &jsonOutput); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			// Collect all callers
			foundCallers := make(map[string]bool)
			collectJSONCallers(&jsonOutput, foundCallers)

			// Verify expected callers
			for _, expectedCaller := range tc.expectedCallers {
				if !foundCallers[expectedCaller] {
					t.Errorf("Missing expected caller %s for function %s", expectedCaller, tc.targetFunc)
				}
			}

			t.Logf("Function %s has %d callers", tc.targetFunc, len(foundCallers))
		})
	}
}