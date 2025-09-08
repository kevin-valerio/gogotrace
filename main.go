package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogotrace/gogotrace/analyzer"
	"github.com/gogotrace/gogotrace/output"
	"github.com/gogotrace/gogotrace/tree"
)

func main() {
	var (
		targetDir  string
		signature  string
		jsonOutput string
		htmlOutput string
		noTests    bool
		help       bool
		listFuncs  string
		showParams bool
	)

	flag.StringVar(&targetDir, "dir", ".", "Directory to analyze")
	flag.StringVar(&signature, "func", "", "Function signature to trace (required)")
	flag.StringVar(&jsonOutput, "json", "", "Output results to JSON file")
	flag.StringVar(&htmlOutput, "html", "", "Output results to HTML file")
	flag.BoolVar(&noTests, "no-test", false, "Exclude test functions from results")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.StringVar(&listFuncs, "list", "", "List functions matching pattern")
	flag.BoolVar(&showParams, "params", false, "Show function parameters in output")
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Show debug information")

	flag.Parse()

	if help || (signature == "" && listFuncs == "") {
		printUsage()
		os.Exit(0)
	}

	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving directory path: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Directory does not exist: %s\n", targetDir)
		os.Exit(1)
	}

	fmt.Printf("Analyzing directory: %s\n", targetDir)
	fmt.Printf("Looking for function: %s\n", signature)
	if noTests {
		fmt.Println("Excluding test functions")
	}
	fmt.Println()

	a := analyzer.NewAnalyzer()

	if err := a.LoadPackages(targetDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading packages: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()

	if listFuncs != "" {
		fmt.Println("Functions matching pattern:")
		for _, fn := range a.GetFunctions() {
			if strings.Contains(fn.Signature, listFuncs) || strings.Contains(fn.Name, listFuncs) {
				fmt.Printf("  %s in %s\n", fn.Signature, fn.FullPath)
			}
		}
		return
	}

	callTree := tree.NewCallTree(a, noTests)
	if err := callTree.Build(signature); err != nil {
		fmt.Fprintf(os.Stderr, "Error building call tree: %v\n", err)
		os.Exit(1)
	}

	if debug {
		fmt.Println("\nDebug: Call graph analysis")
		fmt.Printf("Root function: %s\n", callTree.Root.Function.Name)
		fmt.Printf("Root has %d direct children\n", len(callTree.Root.Children))
		for i, child := range callTree.Root.Children {
			fmt.Printf("  Child %d: %s (has %d children)\n", i+1, child.Function.Name, len(child.Children))
		}
	}

	if jsonOutput != "" {
		fmt.Printf("Writing JSON output to: %s\n", jsonOutput)
		formatter := output.NewJSONFormatter(jsonOutput)
		if err := formatter.Format(callTree); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON output: %v\n", err)
			os.Exit(1)
		}
	}

	if htmlOutput != "" {
		fmt.Printf("Writing HTML output to: %s\n", htmlOutput)
		formatter := output.NewHTMLFormatter(htmlOutput)
		if err := formatter.Format(callTree); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing HTML output: %v\n", err)
			os.Exit(1)
		}
	}

	if jsonOutput == "" && htmlOutput == "" {
		fmt.Println("\n┌─ Reverse Call Graph")
		fmt.Println("└───────────────────────────────────────────────────")
		formatter := output.NewConsoleFormatter(os.Stdout, showParams)
		if err := formatter.Format(callTree); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\nAnalysis complete!")
}

func printUsage() {
	fmt.Println("GoGoTrace - Go Reverse Call Graph Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gogotrace -func \"<function signature>\" [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -func string")
	fmt.Println("        Function signature to trace (required)")
	fmt.Println("  -dir string")
	fmt.Println("        Directory to analyze (default \".\")")
	fmt.Println("  -json string")
	fmt.Println("        Output results to JSON file")
	fmt.Println("  -html string")
	fmt.Println("        Output results to HTML file")
	fmt.Println("  -no-test")
	fmt.Println("        Exclude test functions from results")
	fmt.Println("  -params")
	fmt.Println("        Show function parameters in output")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gogotrace -func \"func (r *inboxMultiplexer) advanceSequencerMsg()\"")
	fmt.Println("  gogotrace -func \"func WithAPIServer(api *server.Server) Opt\" -no-test")
	fmt.Println("  gogotrace -func \"func Process()\" -json output.json")
	fmt.Println("  gogotrace -func \"func main()\" -html callgraph.html")
	fmt.Println("  gogotrace -dir ~/myproject -func \"func Init()\" -no-test")
}
